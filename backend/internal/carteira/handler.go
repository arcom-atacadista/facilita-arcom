package carteira

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

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

func (h *Handler) Rotas(r chi.Router) {
	// Qualquer operador pode ver se a carteira de hoje entrou — é o que
	// explica uma lista vazia sem precisar abrir log de servidor.
	r.Get("/", h.Wrapper(h.ultima))
	// Disparar fora de hora mexe na carteira inteira: só coordenação pra cima.
	r.With(acesso.ExigirPapel(acesso.PapelCoordenacao)).Post("/", h.Wrapper(h.rodarAgora))
}

func (h *Handler) ultima(w http.ResponseWriter, r *http.Request) error {
	u, err := h.svc.Ultima(r.Context())
	if err != nil {
		return err
	}
	return escreverJSON(w, http.StatusOK, map[string]any{
		"ultima":             u,
		"gatewayConfigurado": h.svc.gateway != nil && h.svc.gateway.TemCredencial(),
	})
}

func (h *Handler) rodarAgora(w http.ResponseWriter, r *http.Request) error {
	res, err := h.svc.Sincronizar(r.Context())
	if err != nil {
		if errors.Is(err, ErrSemGateway) {
			return problema.Conflito(
				"A chave do Gateway ARCOM ainda não foi configurada, então não há de onde trazer a carteira.")
		}
		return err
	}
	return escreverJSON(w, http.StatusOK, res)
}

func escreverJSON(w http.ResponseWriter, status int, corpo any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(corpo)
}
