package conversa

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"facilitaarcom/internal/acesso"
	"facilitaarcom/internal/entrada"
	"facilitaarcom/internal/problema"
)

type Handler struct {
	svc     *Service
	Wrapper func(func(http.ResponseWriter, *http.Request) error) http.HandlerFunc
}

func NovoHandler(svc *Service, wrapper func(func(http.ResponseWriter, *http.Request) error) http.HandlerFunc) *Handler {
	return &Handler{svc: svc, Wrapper: wrapper}
}

// Rotas são privadas — o grupo que as monta já passou por ExigirSessao.
func (h *Handler) Rotas(r chi.Router) {
	r.Get("/", h.Wrapper(h.listar))
	r.Get("/{id}/mensagens", h.Wrapper(h.mensagens))
	r.Post("/{id}/mensagens", h.Wrapper(h.responder))
}

// EntradaResponder é o corpo aceito. Só o texto: a conversa vem da URL e o
// autor vem da sessão, nunca do corpo — deixar o cliente informar quem
// respondeu seria assinar mensagem no nome de outro analista.
type EntradaResponder struct {
	Texto string `json:"texto" validate:"required,min=1,max=4096"`
}

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) error {
	u, _ := acesso.DoContexto(r.Context())

	pagina, _ := strconv.Atoi(r.URL.Query().Get("pagina"))
	porPagina, _ := strconv.Atoi(r.URL.Query().Get("porPagina"))

	itens, total, err := h.svc.Listar(r.Context(), u, pagina, porPagina)
	if err != nil {
		return err
	}

	if pagina < 1 {
		pagina = 1
	}
	if porPagina < 1 {
		porPagina = 20
	}
	return escreverJSON(w, http.StatusOK, map[string]any{
		"itens": itens,
		"paginacao": map[string]any{
			"pagina": pagina, "porPagina": porPagina, "total": total,
			"totalPaginas": (total + int64(porPagina) - 1) / int64(porPagina),
		},
	})
}

func (h *Handler) mensagens(w http.ResponseWriter, r *http.Request) error {
	u, _ := acesso.DoContexto(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return problema.NaoEncontrado("Conversa não encontrada.")
	}

	itens, err := h.svc.Mensagens(r.Context(), u, id)
	if err != nil {
		// Fora do escopo e inexistente respondem igual: não revela que a
		// conversa existe na carteira de outra pessoa.
		if errors.Is(err, ErrForaDoEscopo) || EhNaoEncontrado(err) {
			return problema.NaoEncontrado("Conversa não encontrada.")
		}
		return err
	}
	return escreverJSON(w, http.StatusOK, map[string]any{"itens": itens})
}

func (h *Handler) responder(w http.ResponseWriter, r *http.Request) error {
	u, _ := acesso.DoContexto(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return problema.NaoEncontrado("Conversa não encontrada.")
	}

	var dto EntradaResponder
	if err := entrada.Decodificar(r, &dto); err != nil {
		return err
	}

	m, err := h.svc.Responder(r.Context(), u, id, dto.Texto)
	if err != nil {
		return traduzirResposta(err)
	}
	return escreverJSON(w, http.StatusCreated, map[string]any{"mensagem": m})
}

// traduzirResposta mapeia o erro de domínio para o contrato de API.
func traduzirResposta(err error) error {
	switch {
	// Fora do escopo e inexistente respondem igual, pelo mesmo motivo da
	// leitura: 403 confirmaria que a conversa existe na carteira de outro.
	case errors.Is(err, ErrForaDoEscopo), EhNaoEncontrado(err):
		return problema.NaoEncontrado("Conversa não encontrada.")

	// 409: o pedido está correto, o estado da conversa é que não permite.
	case errors.Is(err, ErrJanelaFechada):
		return problema.ConflitoComCodigo(
			"A janela de atendimento desta conversa expirou. Passadas 24 horas da última "+
				"mensagem do cliente, o WhatsApp só aceita uma mensagem de cobrança da régua.",
			"janela_fechada")

	case errors.Is(err, ErrSemSaida):
		return problema.ConflitoComCodigo(
			"O envio pelo WhatsApp ainda não está configurado, então a mesa está em modo leitura.",
			"sem_canal")

	case errors.Is(err, ErrTextoVazio):
		return problema.Validacao(map[string]string{"texto": "Escreva a mensagem antes de enviar."})

	case errors.Is(err, ErrTextoLongo):
		return problema.Validacao(map[string]string{
			"texto": "A mensagem passa do limite de 4096 caracteres do WhatsApp.",
		})
	}
	return err
}

func escreverJSON(w http.ResponseWriter, status int, corpo any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(corpo)
}
