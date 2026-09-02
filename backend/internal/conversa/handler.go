package conversa

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"facilitaarcom/internal/acesso"
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

func escreverJSON(w http.ResponseWriter, status int, corpo any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(corpo)
}
