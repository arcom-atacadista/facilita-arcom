package cobranca

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"facilitaarcom/internal/acesso"
)

const (
	// Validade do link de negociação. O modelo antigo gerava um token que
	// valia para sempre: qualquer link vazado de um WhatsApp antigo continuava
	// mostrando nome, contrato e valor do devedor.
	ValidadeLink = 30 * 24 * time.Hour

	bytesToken = 32

	PorPaginaPadrao = 20
	PorPaginaMaximo = 100
)

var (
	ErrAcordoJaExiste      = errors.New("já existe acordo ativo para esta dívida")
	ErrDividaNaoNegociavel = errors.New("dívida não está aberta")
	ErrSemTelefone         = errors.New("cliente sem telefone")
)

// ErroAlcada é a recusa por desconto acima do teto de quem está lançando.
// Carrega os dois números para a tela poder explicar em vez de só negar.
type ErroAlcada struct {
	Pedido float64
	Teto   float64
}

// ErrSemPosicaoAberta é o cliente sem nenhum título em atraso aberto: não há
// o que negociar. Vira 404 na tela, junto com o caso de estar fora do escopo
// da carteira — os dois respondem igual de propósito.
var ErrSemPosicaoAberta = errors.New("cliente sem títulos em atraso abertos")

func (e *ErroAlcada) Error() string {
	return fmt.Sprintf("desconto de %.0f%% acima da alçada de %.0f%%", e.Pedido, e.Teto)
}

type Service struct {
	repo   *Repo
	appURL string
	agora  func() time.Time
}

func NovoService(repo *Repo, appURL string) *Service {
	return &Service{
		repo:   repo,
		appURL: strings.TrimRight(appURL, "/"),
		agora:  func() time.Time { return time.Now().UTC() },
	}
}

func hashToken(token string) []byte {
	soma := sha256.Sum256([]byte(token))
	return soma[:]
}

// ListarDividas devolve a carteira que o usuário pode ver. O escopo é
// aplicado no repo, sempre — não é opção de filtro.
func (s *Service) ListarDividas(ctx context.Context, u acesso.Usuario, f FiltroDividas) ([]DividaResposta, int64, error) {
	f.Pagina, f.PorPagina = normalizarPaginacao(f.Pagina, f.PorPagina)

	agora := s.agora()
	dividas, total, err := s.repo.ListarDividas(ctx, u, f, agora)
	if err != nil {
		return nil, 0, err
	}

	itens := make([]DividaResposta, 0, len(dividas))
	for _, d := range dividas {
		itens = append(itens, RespostaDeDivida(d, agora))
	}
	return itens, total, nil
}

func normalizarPaginacao(pagina, porPagina int) (int, int) {
	if pagina < 1 {
		pagina = 1
	}
	if porPagina < 1 {
		porPagina = PorPaginaPadrao
	}
	// O contrato manda recusar acima do máximo, mas quem chama já validou o
	// parâmetro; aqui é o piso de segurança contra um Limit gigante.
	if porPagina > PorPaginaMaximo {
		porPagina = PorPaginaMaximo
	}
	return pagina, porPagina
}

// OfertasDoCliente calcula a condição vigente para a posição consolidada de um
// CNPJ, limitada pela alçada de quem consulta.
//
// Recebe o cliente, e não uma dívida, porque o acordo é sempre do CNPJ: um
// cliente com seis títulos vencidos fecha um acordo, não seis. O recorte de
// carteira é aplicado na consulta, então posição de cliente fora do escopo
// volta vazia — e o handler traduz isso em 404, nunca 403.
func (s *Service) OfertasDoCliente(ctx context.Context, u acesso.Usuario, clienteID uuid.UUID) (Posicao, Ofertas, *AcordoResposta, error) {
	dividas, err := s.repo.DividasAbertasDoUsuario(ctx, u, clienteID)
	if err != nil {
		return Posicao{}, Ofertas{}, nil, err
	}

	// O acordo em vigor vai junto: depois de fechado, todos os títulos viram
	// "negociado" e a posição aberta fica vazia. Sem devolver o acordo, a tela
	// diria "sem título em atraso" logo depois de o analista fechar um — o que
	// pareceria que o acordo não foi gravado.
	acordo, err := s.repo.AcordoAtivoDoCliente(ctx, clienteID)
	if err != nil {
		return Posicao{}, Ofertas{}, nil, err
	}

	var resposta *AcordoResposta
	if acordo != nil {
		r := RespostaDeAcordo(*acordo)
		resposta = &r
	}

	if len(dividas) == 0 {
		if resposta != nil {
			// Posição zerada com acordo em vigor é o estado normal de quem
			// acabou de negociar, não um erro.
			return Posicao{}, Ofertas{}, resposta, nil
		}
		return Posicao{}, Ofertas{}, nil, ErrSemPosicaoAberta
	}

	posicao := ConsolidarPosicao(dividas, s.agora())
	politica, err := s.repo.PoliticaDaFaixa(ctx, posicao.DiasAtrasoMaximo)
	if err != nil {
		return Posicao{}, Ofertas{}, nil, err
	}
	return posicao, CalcularOfertas(posicao, politica, u.AlcadaMaxima), resposta, nil
}

// OfertasDaDivida é a mesma coisa a partir de um título: resolve o cliente
// dele (com escopo) e devolve a posição consolidada do CNPJ. A tela da dívida
// mostra o título aberto, mas a condição oferecida é sempre a do conjunto.
func (s *Service) OfertasDaDivida(ctx context.Context, u acesso.Usuario, id uuid.UUID) (Divida, Posicao, Ofertas, error) {
	d, err := s.repo.DividaDoUsuario(ctx, u, id)
	if err != nil {
		return Divida{}, Posicao{}, Ofertas{}, err
	}
	posicao, ofertas, _, err := s.OfertasDoCliente(ctx, u, d.ClienteID)
	if err != nil {
		return Divida{}, Posicao{}, Ofertas{}, err
	}
	return d, posicao, ofertas, nil
}

// GerarLink cria (ou renova) o link público de negociação de uma dívida e
// devolve a URL completa. O token em claro existe só neste retorno: o banco
// guarda o hash, então nem um dump do banco devolve links utilizáveis.
func (s *Service) GerarLink(ctx context.Context, u acesso.Usuario, dividaID uuid.UUID) (string, error) {
	d, err := s.repo.DividaDoUsuario(ctx, u, dividaID)
	if err != nil {
		return "", err
	}

	bruto := make([]byte, bytesToken)
	if _, err := rand.Read(bruto); err != nil {
		return "", fmt.Errorf("gerar token de negociação: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(bruto)

	if err := s.repo.DefinirToken(ctx, d.ID, hashToken(token), s.agora().Add(ValidadeLink)); err != nil {
		return "", err
	}
	return s.appURL + "/negociar/" + token, nil
}

// FecharAcordoOperador é o acordo lançado na mesa. Diferente do fluxo do
// cliente, aqui o operador escolhe o desconto — e é onde a alçada precisa
// valer. No modelo Supabase a alçada existia só como número na tela: nada no
// servidor impedia gravar um acordo com desconto acima dela.
func (s *Service) FecharAcordoOperador(ctx context.Context, u acesso.Usuario, e EntradaFecharAcordo) (Acordo, error) {
	if e.DescontoPct > u.AlcadaMaxima {
		return Acordo{}, &ErroAlcada{Pedido: e.DescontoPct, Teto: u.AlcadaMaxima}
	}

	// O escopo de carteira é aplicado na consulta: cliente que não é da
	// carteira de quem fecha volta vazio, e vira 404 em vez de 403.
	dividas, err := s.repo.DividasAbertasDoUsuario(ctx, u, e.ClienteID)
	if err != nil {
		return Acordo{}, err
	}
	if len(dividas) == 0 {
		// Distinguir os dois motivos de "não há posição aberta" importa: fechar
		// o acordo é o que zera a posição, então responder 404 na segunda
		// tentativa diria ao operador que o cliente não é da carteira dele,
		// quando na verdade ele acabou de negociar.
		acordo, erroAcordo := s.repo.AcordoAtivoDoCliente(ctx, e.ClienteID)
		if erroAcordo != nil {
			return Acordo{}, erroAcordo
		}
		if acordo != nil {
			return Acordo{}, ErrAcordoJaExiste
		}
		return Acordo{}, ErrSemPosicaoAberta
	}

	posicao := ConsolidarPosicao(dividas, s.agora())
	politica, err := s.repo.PoliticaDaFaixa(ctx, posicao.DiasAtrasoMaximo)
	if err != nil {
		return Acordo{}, err
	}

	// O operador também não passa do que a política daquela faixa permite em
	// número de parcelas — alçada é sobre desconto, não sobre prazo.
	maxParcelas := 1
	if politica != nil {
		maxParcelas = politica.MaxParcelas
	}
	if e.Parcelas > maxParcelas {
		return Acordo{}, ErroDeCampo("parcelas",
			fmt.Sprintf("A política desta faixa permite no máximo %d parcela(s).", maxParcelas))
	}

	// Desconto sobre encargos, nunca sobre o saldo inteiro. Mesma conta que a
	// oferta mostra na tela, para o que o operador fecha bater com o que ele viu.
	oferta := montarOferta(posicao, e.DescontoPct, e.Parcelas, entradaPctDa(politica, e.Parcelas))
	if teto := MaxParcelasPara(oferta.ValorTotal); e.Parcelas > teto {
		return Acordo{}, ErroDeCampo("parcelas",
			fmt.Sprintf("Com este valor, o máximo é %d parcela(s) — cada parcela precisa ser de pelo menos R$ %.2f.", teto, ValorMinimoParcela))
	}

	return s.gravar(ctx, dividas, Acordo{
		ClienteID:     e.ClienteID,
		TipoPagamento: e.TipoPagamento,
		DescontoPct:   e.DescontoPct,
		Entrada:       oferta.Entrada,
		Parcelas:      e.Parcelas,
		ValorTotal:    oferta.ValorTotal,
		Origem:        OrigemOperador,
		CriadoPor:     &u.ID,
	})
}

// entradaPctDa devolve a entrada mínima da política, e zero no pagamento à
// vista — entrada num acordo de uma parcela só é a própria parcela.
func entradaPctDa(p *Politica, parcelas int) float64 {
	if p == nil || parcelas <= 1 {
		return 0
	}
	return p.EntradaMinimaPct
}

// gravar monta as parcelas e persiste tudo numa transação. Compartilhado
// entre o acordo do operador e o do cliente (pacote negociacao).
func (s *Service) gravar(ctx context.Context, dividas []Divida, base Acordo) (Acordo, error) {
	agora := s.agora()

	base.ID = uuid.New()
	base.Status = StatusAcordoAtivo
	base.CriadoEm = agora
	base.AtualizadoEm = agora

	valores := DividirParcelas(base.ValorTotal, base.Parcelas)
	if valores == nil {
		// Só chega aqui se alguma validação acima deixou passar — melhor
		// falhar explícito do que gravar acordo sem parcela.
		return Acordo{}, ErroDeCampo("parcelas", "Número de parcelas incompatível com o valor do acordo.")
	}
	datas := VencimentosDasParcelas(agora, base.Parcelas)

	parcelas := make([]Parcela, 0, base.Parcelas)
	for i := range base.Parcelas {
		parcelas = append(parcelas, Parcela{
			ID:         uuid.New(),
			AcordoID:   base.ID,
			Numero:     i + 1,
			Valor:      valores[i],
			Vencimento: datas[i],
			CriadoEm:   agora,
		})
	}

	// A cobertura guarda a foto do saldo de cada título: a sincronização diária
	// atualiza a dívida, e sem a foto o acordo deixaria de bater com a soma dos
	// títulos no dia seguinte.
	base.Cobertura = make([]AcordoDivida, 0, len(dividas))
	for _, d := range dividas {
		base.Cobertura = append(base.Cobertura, AcordoDivida{
			AcordoID:         base.ID,
			DividaID:         d.ID,
			SaldoNoAcordo:    d.ValorOriginal,
			EncargosNoAcordo: d.ValorEncargos,
		})
	}

	if err := s.repo.GravarAcordo(ctx, &base, parcelas); err != nil {
		// O índice único parcial já barrou um acordo ativo concorrente — só
		// traduzimos para um conflito que a tela sabe explicar.
		if EhDuplicado(err) {
			return Acordo{}, ErrAcordoJaExiste
		}
		return Acordo{}, err
	}

	base.Lista = parcelas
	return base, nil
}

// FecharAcordoCliente é o aceite vindo da tela pública. O cliente não escolhe
// desconto: recebe o que a política da faixa dele oferece.
//
// Sem escopo de carteira aqui: quem autoriza é a posse do token do link, não
// uma sessão. A dívida do token serve para achar o CNPJ; o acordo cobre a
// posição inteira dele.
func (s *Service) FecharAcordoCliente(ctx context.Context, d Divida, tipo string, parcelas int) (Acordo, error) {
	if d.Status != StatusDividaAberta {
		return Acordo{}, ErrDividaNaoNegociavel
	}

	dividas, err := s.repo.DividasAbertasDoCliente(ctx, d.ClienteID)
	if err != nil {
		return Acordo{}, err
	}
	if len(dividas) == 0 {
		return Acordo{}, ErrSemPosicaoAberta
	}

	posicao := ConsolidarPosicao(dividas, s.agora())
	politica, err := s.repo.PoliticaDaFaixa(ctx, posicao.DiasAtrasoMaximo)
	if err != nil {
		return Acordo{}, err
	}

	// Alçada 100 porque não há operador na jogada: o cliente recebe o que a
	// política concede, e a política é o próprio teto.
	ofertas := CalcularOfertas(posicao, politica, 100)

	var oferta Oferta
	switch tipo {
	case "avista":
		oferta, parcelas = ofertas.Avista, 1
	case "parcelado":
		if ofertas.Parcelado == nil {
			return Acordo{}, ErroDeCampo("tipo", "Parcelamento indisponível para esta condição.")
		}
		oferta = *ofertas.Parcelado
		// oferta.MaxParcelas já vem limitado por MaxParcelasPara dentro de
		// CalcularOfertas, então este intervalo nunca aceita um número que
		// geraria parcela abaixo do piso.
		if parcelas < 2 || parcelas > oferta.MaxParcelas {
			return Acordo{}, ErroDeCampo("parcelas",
				fmt.Sprintf("Escolha entre 2 e %d parcelas.", oferta.MaxParcelas))
		}
	default:
		return Acordo{}, ErroDeCampo("tipo", "Condição inválida.")
	}

	pagamento := PagamentoPix
	if parcelas > 1 {
		pagamento = PagamentoBoleto
	}

	return s.gravar(ctx, dividas, Acordo{
		ClienteID:     d.ClienteID,
		TipoPagamento: pagamento,
		DescontoPct:   oferta.DescontoPct,
		Entrada:       oferta.Entrada,
		Parcelas:      parcelas,
		ValorTotal:    oferta.ValorTotal,
		Origem:        OrigemCliente,
	})
}

func (s *Service) ListarAcordos(ctx context.Context, u acesso.Usuario, pagina, porPagina int) ([]AcordoResposta, int64, error) {
	pagina, porPagina = normalizarPaginacao(pagina, porPagina)
	acordos, total, err := s.repo.ListarAcordos(ctx, u, pagina, porPagina)
	if err != nil {
		return nil, 0, err
	}
	itens := make([]AcordoResposta, 0, len(acordos))
	for _, a := range acordos {
		itens = append(itens, RespostaDeAcordo(a))
	}
	return itens, total, nil
}

// --- políticas ---

func (s *Service) ListarPoliticas(ctx context.Context) ([]PoliticaResposta, error) {
	ps, err := s.repo.ListarPoliticas(ctx)
	if err != nil {
		return nil, err
	}
	itens := make([]PoliticaResposta, 0, len(ps))
	for _, p := range ps {
		itens = append(itens, RespostaDePolitica(p))
	}
	return itens, nil
}

func (s *Service) AtualizarPolitica(ctx context.Context, id uuid.UUID, e EntradaAtualizarPolitica) (Politica, error) {
	p, err := s.repo.PoliticaPorID(ctx, id)
	if err != nil {
		return Politica{}, err
	}

	if e.Nome != nil {
		p.Nome = strings.TrimSpace(*e.Nome)
	}
	if e.DescontoAvista != nil {
		p.DescontoAvista = *e.DescontoAvista
	}
	if e.DescontoParcelado != nil {
		p.DescontoParcelado = *e.DescontoParcelado
	}
	if e.MaxParcelas != nil {
		p.MaxParcelas = *e.MaxParcelas
	}
	if e.EntradaMinimaPct != nil {
		p.EntradaMinimaPct = *e.EntradaMinimaPct
	}

	// As faixas (faixa_min/faixa_max) não são editáveis por aqui: mexer nelas
	// pode colidir com a política vizinha, e a constraint do banco devolveria
	// um 500. Mudança de faixa é decisão de régua, não de tela de política.
	if err := s.repo.SalvarPolitica(ctx, &p); err != nil {
		return Politica{}, err
	}
	return p, nil
}

// ErroDeCampoCobranca é o erro de validação de negócio deste pacote — o
// handler traduz para o 422 do contrato.
type ErroDeCampoCobranca struct {
	Campo    string
	Mensagem string
}

func (e *ErroDeCampoCobranca) Error() string { return e.Campo + ": " + e.Mensagem }

func ErroDeCampo(campo, mensagem string) error {
	return &ErroDeCampoCobranca{Campo: campo, Mensagem: mensagem}
}
