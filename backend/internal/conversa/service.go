package conversa

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"facilitaarcom/internal/acesso"
	"facilitaarcom/internal/cobranca"
)

// Saida entrega a resposta do analista ao cliente.
//
// É interface, e não o canal concreto do pacote de disparo, pelo mesmo motivo
// do Historico de lá: mesa e régua compartilham o mesmo canal sem que um
// pacote passe a depender do outro, e o teste observa a resposta sem falar com
// a Meta.
//
// Nula deixa a mesa em modo leitura — o analista lê o que o cliente escreveu e
// a tentativa de responder é recusada com erro claro, em vez de a resposta
// desaparecer em silêncio.
type Saida interface {
	EnviarTextoLivre(ctx context.Context, telefone, texto string) (wamid string, err error)
}

type Service struct {
	repo  *Repo
	saida Saida
	log   *slog.Logger
	agora func() time.Time
}

func NovoService(repo *Repo, saida Saida, log *slog.Logger) *Service {
	return &Service{repo: repo, saida: saida, log: log, agora: func() time.Time { return time.Now().UTC() }}
}

// horaDaMeta converte o timestamp em segundos que a Meta manda como string.
func horaDaMeta(valor string, padrao time.Time) time.Time {
	segundos, err := strconv.ParseInt(valor, 10, 64)
	if err != nil || segundos <= 0 {
		return padrao
	}
	return time.Unix(segundos, 0).UTC()
}

// Processar aplica um evento inteiro do webhook.
//
// Devolve erro apenas em falha que valha a Meta reentregar (banco fora do
// ar). Evento que não sabemos tratar é ignorado com registro, e não vira erro
// — senão a Meta reentregaria para sempre o mesmo evento indigesto.
func (s *Service) Processar(ctx context.Context, ev Evento) error {
	for _, entry := range ev.Entry {
		for _, mudanca := range entry.Changes {
			if err := s.processarValor(ctx, mudanca.Value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) processarValor(ctx context.Context, v Value) error {
	for _, m := range v.Messages {
		if err := s.entrada(ctx, v, m); err != nil {
			return err
		}
	}
	for _, st := range v.Statuses {
		if err := s.status(ctx, st); err != nil {
			return err
		}
	}
	return nil
}

// entrada registra a fala do cliente e abre a janela de atendimento.
func (s *Service) entrada(ctx context.Context, v Value, m MsgMeta) error {
	agora := s.agora()
	ocorrida := horaDaMeta(m.Timestamp, agora)

	var nomePerfil *string
	for _, c := range v.Contacts {
		if c.WaID == m.From && c.Profile.Name != "" {
			nome := c.Profile.Name
			nomePerfil = &nome
			break
		}
	}

	conversa, err := s.repo.AbrirOuAtualizar(ctx, m.From, nomePerfil, agora)
	if err != nil {
		return err
	}

	// Reconhecer o número liga a conversa à carteira. Não achar não é erro:
	// a mensagem continua valendo e aparece para a coordenação.
	if conversa.ClienteID == nil {
		if clienteID, err := s.repo.ClientePorTelefone(ctx, m.From); err == nil {
			if err := s.repo.LigarAoCliente(ctx, conversa.ID, clienteID); err != nil {
				s.log.WarnContext(ctx, "não consegui ligar a conversa ao cliente", "erro", err, "conversa", conversa)
			}
		} else if !EhNaoEncontrado(err) {
			return err
		}
	}

	// A resposta do cliente abre 24 horas de janela. A Meta confirma o fim
	// exato no aviso de status; até lá, 24h a partir de agora é a conta certa.
	expira := ocorrida.Add(24 * time.Hour)

	texto := textoDaMensagem(m)
	msg := Mensagem{
		ID:         uuid.New(),
		ConversaID: conversa.ID,
		Direcao:    DirecaoEntrada,
		Wamid:      &m.ID,
		Tipo:       tipoOuTexto(m.Type),
		Texto:      texto,
		OcorridaEm: ocorrida,
		CriadoEm:   agora,
	}

	nova, err := s.repo.RegistrarEntrada(ctx, &msg, &expira)
	if err != nil {
		return err
	}
	if !nova {
		// Reentrega da Meta: já tínhamos essa mensagem.
		return nil
	}

	s.log.InfoContext(ctx, "mensagem recebida do cliente", "conversa", conversa, "tipo", msg.Tipo)
	return nil
}

// textoDaMensagem extrai o que dá para mostrar. Tipos que não são texto
// (áudio, imagem, documento) ficam registrados pelo tipo, sem conteúdo — a
// mídia exige um download autenticado que não faz parte desta etapa.
func textoDaMensagem(m MsgMeta) *string {
	switch {
	case m.Text.Body != "":
		t := m.Text.Body
		return &t
	case m.Button.Text != "":
		t := m.Button.Text
		return &t
	default:
		return nil
	}
}

func tipoOuTexto(tipo string) string {
	if tipo == "" {
		return "text"
	}
	return tipo
}

// status aplica o aviso de entrega do que saiu.
func (s *Service) status(ctx context.Context, st StatusMeta) error {
	nosso := traduzirStatus(st.Status)
	if nosso == "" {
		return nil
	}

	quando := horaDaMeta(st.Timestamp, s.agora())

	var codigo *int
	var detalhe *string
	if len(st.Errors) > 0 {
		c := st.Errors[0].Code
		codigo = &c
		d := "a Meta não conseguiu entregar a mensagem"
		if st.Errors[0].Title != "" {
			d = st.Errors[0].Title
		}
		detalhe = &d
	}

	if err := s.repo.AtualizarStatusDaSaida(ctx, st.ID, nosso, codigo); err != nil {
		return err
	}
	if err := s.repo.MarcarEntregaDoDisparo(ctx, st.ID, nosso, quando, detalhe); err != nil {
		return err
	}

	// A Meta informa o fim exato da janela junto do status; é mais confiável
	// que a nossa conta de 24h e substitui ela quando vem.
	if st.Conversation.ExpirationTimestamp != "" && st.RecipientID != "" {
		expira := horaDaMeta(st.Conversation.ExpirationTimestamp, time.Time{})
		if !expira.IsZero() {
			if err := s.atualizarJanela(ctx, st.RecipientID, expira); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) atualizarJanela(ctx context.Context, telefone string, expira time.Time) error {
	c, err := s.repo.PorTelefone(ctx, telefone)
	if err != nil {
		if EhNaoEncontrado(err) {
			return nil // status de número sem conversa aberta: nada a fazer
		}
		return err
	}
	// Só estende: um aviso antigo chegando fora de ordem não pode encurtar
	// uma janela que já vale mais tempo.
	if c.JanelaExpiraEm != nil && !expira.After(*c.JanelaExpiraEm) {
		return nil
	}
	return s.repo.db.WithContext(ctx).Model(&Conversa{}).
		Where("id = ?", c.ID).Update("janela_expira_em", expira).Error
}

func traduzirStatus(meta string) string {
	switch meta {
	case "sent":
		return StatusEnviada
	case "delivered":
		return StatusEntregue
	case "read":
		return StatusLida
	case "failed":
		return StatusFalhou
	default:
		return ""
	}
}

// RegistrarDisparoEnviado guarda na conversa a mensagem que a régua mandou,
// para a mesa mostrar o histórico completo e não só o que o cliente falou.
func (s *Service) RegistrarDisparoEnviado(ctx context.Context, telefone, wamid, texto string) error {
	agora := s.agora()

	conversa, err := s.repo.AbrirOuAtualizar(ctx, telefone, nil, agora)
	if err != nil {
		return err
	}

	enviada := StatusEnviada
	return s.repo.RegistrarSaida(ctx, &Mensagem{
		ID:         uuid.New(),
		ConversaID: conversa.ID,
		Direcao:    DirecaoSaida,
		Wamid:      &wamid,
		Tipo:       "template",
		Texto:      &texto,
		Status:     &enviada,
		OcorridaEm: agora,
		CriadoEm:   agora,
	})
}

// --- leitura pela mesa ---

type ConversaResposta struct {
	ID               uuid.UUID  `json:"id"`
	Telefone         string     `json:"telefone"`
	NomePerfil       *string    `json:"nomePerfil"`
	ClienteID        *uuid.UUID `json:"clienteId"`
	JanelaAberta     bool       `json:"janelaAberta"`
	JanelaExpiraEm   *time.Time `json:"janelaExpiraEm"`
	UltimaMensagemEm *time.Time `json:"ultimaMensagemEm"`
	NaoLidas         int        `json:"naoLidas"`

	// Nome é como o cliente é tratado na mesa: razão social sem o sufixo
	// jurídico para PJ, primeiro nome para PF (ver cobranca.NomeDeTratamento).
	// Sem cliente na carteira sobra o nome do perfil do WhatsApp, e sem ele o
	// telefone mascarado — a conversa nunca aparece sem identificação.
	Nome string `json:"nome"`

	// Previa é a última fala da conversa, o que permite triar a fila sem abrir
	// cada uma. Vazia quando a última mensagem não é texto (áudio, imagem).
	Previa *string `json:"previa"`

	// PreviaDoCliente diz de que lado veio a prévia, para a tela marcar o que
	// está esperando resposta.
	PreviaDoCliente bool `json:"previaDoCliente"`
}

type MensagemResposta struct {
	ID         uuid.UUID `json:"id"`
	Direcao    string    `json:"direcao"`
	Tipo       string    `json:"tipo"`
	Texto      *string   `json:"texto"`
	Status     *string   `json:"status"`
	OcorridaEm time.Time `json:"ocorridaEm"`
}

func (s *Service) Listar(ctx context.Context, u acesso.Usuario, pagina, porPagina int) ([]ConversaResposta, int64, error) {
	if pagina < 1 {
		pagina = 1
	}
	if porPagina < 1 || porPagina > 100 {
		porPagina = 20
	}

	conversas, total, err := s.repo.ListarConversas(ctx, u, pagina, porPagina)
	if err != nil {
		return nil, 0, err
	}

	ids := make([]uuid.UUID, 0, len(conversas))
	for _, c := range conversas {
		ids = append(ids, c.ID)
	}
	resumos, err := s.repo.ResumoDaFilaPor(ctx, ids)
	if err != nil {
		// A fila continua utilizável sem nome e prévia: some o que ajuda a
		// triar, não o acesso à conversa. Melhor lista pobre que tela vazia.
		s.log.WarnContext(ctx, "não consegui carregar nome e prévia da fila", "erro", err)
		resumos = map[uuid.UUID]ResumoDaFila{}
	}

	agora := s.agora()
	itens := make([]ConversaResposta, 0, len(conversas))
	for _, c := range conversas {
		mascarado := MascararTelefone(c.Telefone)
		r := resumos[c.ID]

		item := ConversaResposta{
			ID: c.ID, Telefone: mascarado, NomePerfil: c.NomePerfil,
			ClienteID: c.ClienteID, JanelaAberta: c.JanelaAberta(agora),
			JanelaExpiraEm: c.JanelaExpiraEm, UltimaMensagemEm: c.UltimaMensagemEm,
			NaoLidas: c.NaoLidas,
			Nome:     nomeDaMesa(r, c.NomePerfil, mascarado),
			Previa:   r.UltimoTexto,
		}
		if r.UltimaDirecao != nil {
			item.PreviaDoCliente = *r.UltimaDirecao == DirecaoEntrada
		}
		itens = append(itens, item)
	}
	return itens, total, nil
}

// nomeDaMesa escolhe como chamar o cliente, na ordem de confiança: o cadastro
// da carteira, o nome que a pessoa pôs no perfil do WhatsApp, e por último o
// telefone mascarado. Nunca devolve vazio — conversa sem identificação na tela
// é conversa que o analista não consegue atender.
func nomeDaMesa(r ResumoDaFila, nomePerfil *string, telefoneMascarado string) string {
	if r.NomeCliente != nil && *r.NomeCliente != "" {
		doc := ""
		if r.Documento != nil {
			doc = *r.Documento
		}
		return cobranca.NomeDeTratamento(*r.NomeCliente, doc)
	}
	if nomePerfil != nil && *nomePerfil != "" {
		return *nomePerfil
	}
	return telefoneMascarado
}

var ErrForaDoEscopo = errors.New("conversa fora da carteira do usuário")

func (s *Service) Mensagens(ctx context.Context, u acesso.Usuario, id uuid.UUID) ([]MensagemResposta, error) {
	// A checagem de escopo passa pela mesma listagem: se a conversa não
	// aparece para este usuário, ele também não abre o histórico dela.
	if err := s.confirmarEscopo(ctx, u, id); err != nil {
		return nil, err
	}

	ms, err := s.repo.MensagensDaConversa(ctx, id, 200)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ZerarNaoLidas(ctx, id); err != nil {
		s.log.WarnContext(ctx, "não consegui zerar as não lidas", "erro", err)
	}

	itens := make([]MensagemResposta, 0, len(ms))
	for _, m := range ms {
		itens = append(itens, MensagemResposta{
			ID: m.ID, Direcao: m.Direcao, Tipo: m.Tipo,
			Texto: m.Texto, Status: m.Status, OcorridaEm: m.OcorridaEm,
		})
	}
	return itens, nil
}

func (s *Service) confirmarEscopo(ctx context.Context, u acesso.Usuario, id uuid.UUID) error {
	if u.Papel.VeCarteiraInteira() {
		if _, err := s.repo.PorID(ctx, id); err != nil {
			return err
		}
		return nil
	}

	// Para quem tem carteira própria, a consulta com escopo é a autorização.
	conversas, _, err := s.repo.ListarConversas(ctx, u, 1, 1000)
	if err != nil {
		return err
	}
	for _, c := range conversas {
		if c.ID == id {
			return nil
		}
	}
	return ErrForaDoEscopo
}

var (
	// ErrJanelaFechada é o caso comum e esperado, não uma falha: passadas as
	// 24 horas desde a última mensagem do cliente, a Meta recusa texto livre.
	// Daí em diante só um template aprovado alcança essa pessoa — o que é
	// disparo da régua, não resposta de mesa.
	ErrJanelaFechada = errors.New("janela de atendimento fechada")

	// ErrSemSaida é a mesa sem canal configurado: lê, não responde.
	ErrSemSaida = errors.New("nenhum canal de envio configurado")

	// ErrTextoVazio e ErrTextoLongo são validação de entrada. Ficam também no
	// serviço, e não só no DTO do handler, porque a regra vale para qualquer
	// chamador — validação no handler é a primeira barreira, não a única.
	ErrTextoVazio = errors.New("resposta sem texto")
	ErrTextoLongo = errors.New("resposta acima do limite da Meta")
)

// LimiteTextoResposta é o teto da Meta para o corpo de uma mensagem de texto.
// Cortar aqui evita gastar a chamada para ela recusar por tamanho.
const LimiteTextoResposta = 4096

// Responder entrega a mensagem do analista ao cliente e a guarda na conversa.
//
// A ordem importa: envia primeiro, registra depois. Registrar antes deixaria
// na tela uma resposta que o cliente nunca recebeu — numa cobrança, isso é
// pior do que não ter registro, porque o analista pararia de insistir
// acreditando ter falado.
func (s *Service) Responder(ctx context.Context, u acesso.Usuario, id uuid.UUID, texto string) (MensagemResposta, error) {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return MensagemResposta{}, ErrTextoVazio
	}
	if len([]rune(texto)) > LimiteTextoResposta {
		return MensagemResposta{}, ErrTextoLongo
	}

	// Escopo antes de tudo: quem não vê a conversa também não escreve nela.
	if err := s.confirmarEscopo(ctx, u, id); err != nil {
		return MensagemResposta{}, err
	}

	c, err := s.repo.PorID(ctx, id)
	if err != nil {
		return MensagemResposta{}, err
	}

	agora := s.agora()
	if !c.JanelaAberta(agora) {
		return MensagemResposta{}, ErrJanelaFechada
	}
	if s.saida == nil {
		return MensagemResposta{}, ErrSemSaida
	}

	wamid, err := s.saida.EnviarTextoLivre(ctx, c.Telefone, texto)
	if err != nil {
		return MensagemResposta{}, err
	}

	m := Mensagem{
		ID:         uuid.New(),
		ConversaID: c.ID,
		Direcao:    DirecaoSaida,
		Tipo:       "text",
		Texto:      &texto,
		Status:     ptr(StatusEnviada),
		OcorridaEm: agora,
		CriadoEm:   agora,
	}
	if wamid != "" {
		m.Wamid = &wamid
	}

	if err := s.repo.RegistrarSaida(ctx, &m); err != nil {
		// A mensagem já saiu; perder o registro não a traz de volta. Devolver
		// erro aqui faria o analista mandar de novo e o cliente receber duas
		// vezes, então o certo é avisar alto e considerar entregue.
		s.log.ErrorContext(ctx, "resposta entregue mas não registrada na conversa",
			"conversa", c, "erro", err)
	}

	s.log.InfoContext(ctx, "resposta da mesa entregue",
		"conversa", c, "usuario", u.ID, "tamanho_texto", len(texto))

	return MensagemResposta{
		ID: m.ID, Direcao: m.Direcao, Tipo: m.Tipo,
		Texto: m.Texto, Status: m.Status, OcorridaEm: m.OcorridaEm,
	}, nil
}

func ptr[T any](v T) *T { return &v }
