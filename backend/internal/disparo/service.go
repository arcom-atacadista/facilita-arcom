package disparo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"facilitaarcom/internal/acesso"
	"facilitaarcom/internal/cobranca"
)

const (
	// IntervaloEntreDisparos é a trava anti-spam por devedor.
	IntervaloEntreDisparos = 72 * time.Hour

	// ValidadeNaFila é quanto tempo uma mensagem pode esperar na fila antes
	// de ser descartada. Importa especialmente enquanto não há canal: no dia
	// em que um for ligado, a fila não pode despejar cobrança de semanas
	// atrás sobre clientes que já pagaram.
	ValidadeNaFila = 7 * 24 * time.Hour

	// MaxTentativas por mensagem antes de desistir.
	MaxTentativas = 3

	// LoteDoWorker é quanto cada rodada processa. Pequeno de propósito: o
	// worker roda de minuto em minuto e é melhor escoar aos poucos do que
	// segurar uma transação longa.
	LoteDoWorker = 50
)

var (
	ErrDisparoRecente = errors.New("já houve disparo recente para esta dívida")
	ErrSemTelefone    = errors.New("cliente sem telefone válido")
	ErrSemCampanha    = errors.New("nenhuma campanha ativa para esta faixa de atraso")
	ErrForaDaRegua    = errors.New("dívida fora da régua de cobrança")
)

// Historico registra na conversa o que a régua mandou, para a mesa mostrar o
// diálogo completo e não só o lado do cliente.
//
// É interface e não o serviço concreto para o disparo não depender do pacote
// de conversa — e para o teste conseguir observar o registro sem banco.
// Nulo desliga: a entrega continua funcionando, só não vira histórico.
type Historico interface {
	RegistrarDisparoEnviado(ctx context.Context, telefone, wamid, texto string) error
}

type Service struct {
	repo      *Repo
	cobranca  *cobranca.Service
	repoCob   *cobranca.Repo
	canal     Canal
	historico Historico
	log       *slog.Logger
	agora     func() time.Time
}

// NovoService recebe o canal de saída. canal nil é o estado atual do projeto
// (ver canal.go): tudo funciona, a fila enche, e nada é enviado nem marcado
// como enviado.
func NovoService(repo *Repo, repoCob *cobranca.Repo, svcCob *cobranca.Service, canal Canal, historico Historico, log *slog.Logger) *Service {
	return &Service{
		repo: repo, repoCob: repoCob, cobranca: svcCob,
		canal: canal, historico: historico, log: log,
		agora: func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) TemCanal() bool { return s.canal != nil }

// NomeDoCanal identifica quem entrega. Sem canal configurado, o registro
// guarda "pendente" — não "whatsapp", que sugeriria que saiu por lá.
func (s *Service) NomeDoCanal() string {
	if s.canal == nil {
		return "pendente"
	}
	return s.canal.Nome()
}

// Enfileirar monta a mensagem da faixa da dívida e coloca na fila. Não envia:
// quem envia é o worker.
func (s *Service) Enfileirar(ctx context.Context, u acesso.Usuario, dividaID uuid.UUID) (Disparo, error) {
	d, err := s.repoCob.DividaDoUsuario(ctx, u, dividaID)
	if err != nil {
		return Disparo{}, err
	}
	if d.Status != cobranca.StatusDividaAberta {
		return Disparo{}, cobranca.ErrDividaNaoNegociavel
	}

	agora := s.agora()
	dias := cobranca.DiasAtraso(d.Vencimento, agora)
	if cobranca.FaixaDe(dias) == cobranca.FaixaFora {
		return Disparo{}, ErrForaDaRegua
	}

	if d.Cliente == nil {
		return Disparo{}, ErrSemTelefone
	}
	var telefone string
	if d.Cliente.Telefone != nil {
		telefone = TelefoneComDDI(*d.Cliente.Telefone)
	}
	if !TelefoneValido(telefone) {
		return Disparo{}, ErrSemTelefone
	}

	recente, err := s.repo.HouveDisparoRecente(ctx, d.ID, agora.Add(-IntervaloEntreDisparos))
	if err != nil {
		return Disparo{}, err
	}
	if recente {
		return Disparo{}, ErrDisparoRecente
	}

	campanha, err := s.repoCob.CampanhaDaFaixa(ctx, dias)
	if err != nil {
		return Disparo{}, err
	}
	if campanha == nil {
		return Disparo{}, ErrSemCampanha
	}

	// O link é gerado agora e vale 30 dias — ver cobranca.GerarLink. Cada
	// disparo renova o token, então o link da mensagem anterior deixa de
	// valer, o que é o comportamento desejado.
	link, err := s.cobranca.GerarLink(ctx, u, d.ID)
	if err != nil {
		return Disparo{}, err
	}

	variaveis := Variaveis{
		Tratamento:   cobranca.NomeDeTratamento(d.Cliente.Nome, d.Cliente.Documento),
		NomeCompleto: d.Cliente.Nome,
		Contrato:     d.Contrato,
		Dias:         dias,
		Valor:        d.ValorOriginal,
		Link:         link,
	}
	texto := MontarMensagem(campanha.Template, variaveis)

	// Os parâmetros vão na ordem que a campanha declara, porque é essa a
	// ordem dos marcadores {{1}}, {{2}}... no template aprovado na Meta.
	parametros, err := ValoresNaOrdem(campanha.Parametros, variaveis)
	if err != nil {
		return Disparo{}, err
	}

	disparo := Disparo{
		ID:           uuid.New(),
		DividaID:     d.ID,
		CampanhaID:   &campanha.ID,
		Telefone:     telefone,
		Mensagem:     texto,
		TemplateMeta: campanha.TemplateMeta,
		Idioma:       idiomaOu(campanha.Idioma),
		Parametros:   parametros,
		Canal:        s.NomeDoCanal(),
		Status:       StatusNaFila,
		AgendadoPara: agora,
		CriadoPor:    &u.ID,
		CriadoEm:     agora,
	}
	if err := s.repo.Criar(ctx, &disparo); err != nil {
		return Disparo{}, err
	}
	return disparo, nil
}

func (s *Service) Listar(ctx context.Context, u acesso.Usuario, pagina, porPagina int) ([]DisparoResposta, int64, error) {
	if pagina < 1 {
		pagina = 1
	}
	if porPagina < 1 || porPagina > cobranca.PorPaginaMaximo {
		porPagina = cobranca.PorPaginaPadrao
	}

	disparos, total, err := s.repo.Listar(ctx, u, pagina, porPagina)
	if err != nil {
		return nil, 0, err
	}
	itens := make([]DisparoResposta, 0, len(disparos))
	for _, d := range disparos {
		itens = append(itens, Responder(d))
	}
	return itens, total, nil
}

func (s *Service) Resumo(ctx context.Context) (map[string]any, error) {
	porStatus, err := s.repo.ContarPorStatus(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"porStatus": porStatus,
		"canal":     s.NomeDoCanal(),
		"ativo":     s.TemCanal(),
	}, nil
}

// ProcessarFila é uma rodada do worker: descarta o que envelheceu e tenta
// entregar o que está pronto. Devolve quantas mensagens saíram.
func (s *Service) ProcessarFila(ctx context.Context) (int, error) {
	agora := s.agora()

	if n, err := s.repo.CancelarVencidos(ctx, agora.Add(-ValidadeNaFila)); err != nil {
		return 0, fmt.Errorf("cancelar disparos vencidos: %w", err)
	} else if n > 0 {
		s.log.WarnContext(ctx, "disparos cancelados por tempo na fila", "quantidade", n)
	}

	// Sem canal, a rodada termina aqui: a fila permanece intacta, esperando.
	// Marcar como enviado (ou como erro) seria mentir sobre o que aconteceu.
	if s.canal == nil {
		return 0, nil
	}

	reservados, err := s.repo.Reservar(ctx, agora, LoteDoWorker)
	if err != nil {
		return 0, fmt.Errorf("reservar disparos: %w", err)
	}

	enviados := 0
	for _, d := range reservados {
		if s.entregar(ctx, d) {
			enviados++
		}
	}
	return enviados, nil
}

func (s *Service) entregar(ctx context.Context, d Disparo) bool {
	msg := Mensagem{
		Telefone:   d.Telefone,
		Texto:      d.Mensagem,
		Idioma:     d.Idioma,
		Parametros: d.Parametros,
	}
	if d.TemplateMeta != nil {
		msg.Template = *d.TemplateMeta
	}

	referencia, err := s.canal.Enviar(ctx, msg)
	if err == nil {
		if err := s.repo.MarcarEnviado(ctx, d.ID, referencia, s.agora()); err != nil {
			s.log.ErrorContext(ctx, "falha ao registrar disparo enviado", "erro", err, "disparo_id", d.ID)
		}
		// O histórico é conveniência da mesa: falhar aqui não pode desfazer
		// um envio que já aconteceu de verdade.
		if s.historico != nil {
			if err := s.historico.RegistrarDisparoEnviado(ctx, d.Telefone, referencia, d.Mensagem); err != nil {
				s.log.WarnContext(ctx, "disparo entregue mas não entrou no histórico da conversa",
					"erro", err, "disparo_id", d.ID)
			}
		}
		return true
	}

	// O erro do provedor vai pro log inteiro, mas só uma versão curta é
	// gravada: erro_detalhe aparece na tela do operador.
	s.log.ErrorContext(ctx, "falha ao enviar disparo", "erro", err, "disparo_id", d.ID, "mensagem", msg)

	// Recusa permanente (template não aprovado, número inválido, token
	// expirado) não melhora com nova tentativa: insistir três vezes só gasta
	// tempo e enche o log. Vai direto para erro, com o motivo à vista.
	var permanente *ErroMeta
	if errors.As(err, &permanente) && permanente.Permanente {
		if err := s.repo.MarcarErro(ctx, d.ID, motivoDoErro(permanente), nil); err != nil {
			s.log.ErrorContext(ctx, "falha ao registrar erro permanente de disparo", "erro", err, "disparo_id", d.ID)
		}
		return false
	}

	var proxima *time.Time
	if d.Tentativas < MaxTentativas {
		// Backoff exponencial simples: 5, 10, 20 minutos.
		espera := time.Duration(1<<uint(d.Tentativas)) * 5 * time.Minute
		t := s.agora().Add(espera)
		proxima = &t
	}

	if err := s.repo.MarcarErro(ctx, d.ID, motivoDoErro(err), proxima); err != nil {
		s.log.ErrorContext(ctx, "falha ao registrar erro de disparo", "erro", err, "disparo_id", d.ID)
	}
	return false
}

// idiomaOu garante um idioma válido mesmo em campanha antiga sem a coluna
// preenchida.
func idiomaOu(idioma string) string {
	if idioma == "" {
		return "pt_BR"
	}
	return idioma
}

// motivoDoErro é o texto curto que vai para a tela do operador. Erro da Meta
// vira uma explicação acionável; qualquer outro fica genérico, porque o
// detalhe pode carregar dado interno e já está no log.
func motivoDoErro(err error) string {
	var meta *ErroMeta
	if !errors.As(err, &meta) {
		return "falha na entrega pelo canal de mensagem"
	}

	switch {
	case meta.Codigo == 190:
		return "credencial do WhatsApp expirada — renovar o token com o time de TI"
	case meta.Codigo == 131026:
		return "número não recebe mensagem no WhatsApp"
	case meta.Codigo >= 132000 && meta.Codigo <= 132015:
		return "template não aprovado na Meta ou com número de parâmetros diferente do cadastrado"
	case meta.Status == 429:
		return "limite de envio da Meta atingido — será tentado de novo"
	default:
		return fmt.Sprintf("a Meta recusou o envio (código %d)", meta.Codigo)
	}
}

// ValoresNaOrdem resolve os nomes declarados na campanha para os valores
// daquele disparo, na ordem dos marcadores do template.
//
// Nome desconhecido é erro, e não string vazia: um template aprovado com
// cinco marcadores recebendo um parâmetro em branco é recusado pela Meta com
// uma mensagem genérica, e o rastro se perde. Melhor falhar no
// enfileiramento, onde dá para ver qual campanha está mal configurada.
func ValoresNaOrdem(nomes []string, v Variaveis) ([]string, error) {
	valores := make([]string, 0, len(nomes))
	for _, nome := range nomes {
		switch nome {
		case "nome":
			valores = append(valores, v.Tratamento)
		case "nome_completo":
			valores = append(valores, v.NomeCompleto)
		case "contrato":
			valores = append(valores, v.Contrato)
		case "dias":
			valores = append(valores, fmt.Sprintf("%d", v.Dias))
		case "valor":
			valores = append(valores, Moeda(v.Valor))
		case "link":
			valores = append(valores, v.Link)
		default:
			return nil, fmt.Errorf("campanha declara o parâmetro %q, que não existe", nome)
		}
	}
	return valores, nil
}
