// Package negociacao é a tela pública que o cliente devedor abre pelo link
// recebido no WhatsApp. É a única parte do sistema sem sessão: quem autoriza
// é a posse do token, não uma conta.
//
// Por isso tudo aqui é deliberadamente econômico com dado pessoal — devolve o
// primeiro nome, o contrato e o valor, e nada mais. Um link vazado não pode
// virar consulta à ficha do cliente.
package negociacao

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"facilitaarcom/internal/cobranca"
	"facilitaarcom/internal/entrada"
	"facilitaarcom/internal/problema"
)

// Tamanho do token gerado por cobranca.GerarLink (32 bytes em base64url sem
// padding = 43 caracteres). Conferir antes de tocar o banco descarta lixo de
// varredura sem gastar uma consulta.
const tamanhoToken = 43

type Service struct {
	repo     *cobranca.Repo
	cobranca *cobranca.Service
	agora    func() time.Time
}

func NovoService(repo *cobranca.Repo, svc *cobranca.Service) *Service {
	return &Service{repo: repo, cobranca: svc, agora: func() time.Time { return time.Now().UTC() }}
}

var ErrLinkInvalido = errors.New("link inválido ou expirado")

func hashToken(token string) []byte {
	soma := sha256.Sum256([]byte(token))
	return soma[:]
}

// Proposta é o que a tela pública mostra.
type Proposta struct {
	Cliente       string                   `json:"cliente"`
	Contrato      string                   `json:"contrato"`
	ValorOriginal float64                  `json:"valorOriginal"`
	Vencimento    string                   `json:"vencimento"`
	DiasAtraso    int                      `json:"diasAtraso"`
	Ofertas       cobranca.Ofertas         `json:"ofertas"`
	Acordo        *cobranca.AcordoResposta `json:"acordo"`
}

type EntradaAceite struct {
	Tipo     string `json:"tipo" validate:"required,oneof=avista parcelado"`
	Parcelas int    `json:"parcelas" validate:"required,gte=1,lte=24"`
}

func (s *Service) dividaDoToken(ctx context.Context, token string) (cobranca.Divida, error) {
	if len(token) != tamanhoToken {
		return cobranca.Divida{}, ErrLinkInvalido
	}
	d, err := s.repo.DividaPorTokenHash(ctx, hashToken(token), s.agora())
	if err != nil {
		if errors.Is(err, cobranca.ErrNaoEncontrado) {
			return cobranca.Divida{}, ErrLinkInvalido
		}
		return cobranca.Divida{}, err
	}
	return d, nil
}

func (s *Service) Consultar(ctx context.Context, token string) (Proposta, error) {
	d, err := s.dividaDoToken(ctx, token)
	if err != nil {
		return Proposta{}, err
	}

	dias := cobranca.DiasAtraso(d.Vencimento, s.agora())
	politica, err := s.repo.PoliticaDaFaixa(ctx, dias)
	if err != nil {
		return Proposta{}, err
	}

	acordo, err := s.repo.AcordoAtivoDaDivida(ctx, d.ID)
	if err != nil {
		return Proposta{}, err
	}

	p := Proposta{
		Cliente:       tratamento(d),
		Contrato:      d.Contrato,
		ValorOriginal: d.ValorOriginal,
		Vencimento:    d.Vencimento.Format(cobranca.FormatoData),
		DiasAtraso:    dias,
		Ofertas:       cobranca.CalcularOfertas(d.ValorOriginal, politica),
	}
	if acordo != nil {
		r := cobranca.RespostaDeAcordo(*acordo)
		p.Acordo = &r
	}
	return p, nil
}

// tratamento é como o cliente é chamado na tela. Pessoa física aparece só
// pelo primeiro nome — o bastante pra reconhecer que a proposta é dela, sem
// expor o nome completo a quem tiver o link. Empresa aparece pela razão
// social sem a forma jurídica, porque a carteira da ARCOM é quase toda PJ e
// cortar na primeira palavra não identificaria ninguém.
func tratamento(d cobranca.Divida) string {
	if d.Cliente == nil {
		return "Cliente"
	}
	return cobranca.NomeDeTratamento(d.Cliente.Nome, d.Cliente.Documento)
}

func (s *Service) Aceitar(ctx context.Context, token string, e EntradaAceite) (cobranca.AcordoResposta, error) {
	d, err := s.dividaDoToken(ctx, token)
	if err != nil {
		return cobranca.AcordoResposta{}, err
	}

	acordo, err := s.cobranca.FecharAcordoCliente(ctx, d, e.Tipo, e.Parcelas)
	if err != nil {
		return cobranca.AcordoResposta{}, err
	}
	return cobranca.RespostaDeAcordo(acordo), nil
}

// --- HTTP ---

type Handler struct {
	svc     *Service
	Wrapper func(func(http.ResponseWriter, *http.Request) error) http.HandlerFunc
}

func NovoHandler(svc *Service, wrapper func(func(http.ResponseWriter, *http.Request) error) http.HandlerFunc) *Handler {
	return &Handler{svc: svc, Wrapper: wrapper}
}

// Rotas é pública. Rate limit por IP bem mais apertado que o global: é a
// única superfície do sistema aberta na internet, e um token de 256 bits não
// é enumerável, mas a rota ainda pode ser usada pra sondar o serviço.
func (h *Handler) Rotas(r chi.Router) {
	r.Use(httprate.LimitByIP(30, time.Minute))
	r.Get("/{token}", h.Wrapper(h.consultar))
	r.Post("/{token}/aceitar", h.Wrapper(h.aceitar))
}

func (h *Handler) consultar(w http.ResponseWriter, r *http.Request) error {
	p, err := h.svc.Consultar(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		return traduzir(err)
	}
	return escreverJSON(w, http.StatusOK, p)
}

func (h *Handler) aceitar(w http.ResponseWriter, r *http.Request) error {
	var dto EntradaAceite
	if err := entrada.Decodificar(r, &dto); err != nil {
		return err
	}

	acordo, err := h.svc.Aceitar(r.Context(), chi.URLParam(r, "token"), dto)
	if err != nil {
		return traduzir(err)
	}
	return escreverJSON(w, http.StatusCreated, acordo)
}

func traduzir(err error) error {
	var campo *cobranca.ErroDeCampoCobranca
	switch {
	case errors.Is(err, ErrLinkInvalido):
		return &problema.ErroDominio{
			Status: http.StatusNotFound,
			Titulo: "Link inválido",
			Detail: "Este link de negociação não é válido ou já expirou. Fale com a nossa equipe para receber um novo.",
			Codigo: "link_invalido",
		}
	case errors.As(err, &campo):
		return problema.UmCampo(campo.Campo, campo.Mensagem)
	case errors.Is(err, cobranca.ErrAcordoJaExiste):
		return problema.Conflito("Já existe um acordo ativo para este contrato.")
	case errors.Is(err, cobranca.ErrDividaNaoNegociavel):
		return problema.Conflito("Este contrato não está mais aberto para negociação.")
	default:
		return err
	}
}

func escreverJSON(w http.ResponseWriter, status int, corpo any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(corpo)
}
