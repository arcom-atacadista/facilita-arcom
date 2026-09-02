package disparo

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"facilitaarcom/internal/acesso"
	"facilitaarcom/internal/cobranca"
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

func (h *Handler) Rotas(r chi.Router) {
	r.Get("/", h.Wrapper(h.listar))
	r.Get("/resumo", h.Wrapper(h.resumo))
	r.Post("/", h.Wrapper(h.enfileirar))
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
		porPagina = cobranca.PorPaginaPadrao
	}
	return escreverJSON(w, http.StatusOK, map[string]any{
		"itens": itens,
		"paginacao": map[string]any{
			"pagina": pagina, "porPagina": porPagina, "total": total,
			"totalPaginas": (total + int64(porPagina) - 1) / int64(porPagina),
		},
	})
}

// resumo alimenta o painel: quantos estão na fila, quantos saíram, e — o que
// mais importa hoje — se existe canal de envio configurado.
func (h *Handler) resumo(w http.ResponseWriter, r *http.Request) error {
	resumo, err := h.svc.Resumo(r.Context())
	if err != nil {
		return err
	}
	return escreverJSON(w, http.StatusOK, resumo)
}

func (h *Handler) enfileirar(w http.ResponseWriter, r *http.Request) error {
	u, _ := acesso.DoContexto(r.Context())

	var dto EntradaEnfileirar
	if err := entrada.Decodificar(r, &dto); err != nil {
		return err
	}

	d, err := h.svc.Enfileirar(r.Context(), u, dto.DividaID)
	if err != nil {
		return traduzir(err)
	}

	resposta := map[string]any{"disparo": Responder(d)}
	// O operador precisa saber que a mensagem entrou na fila mas ainda não
	// tem por onde sair — senão vai achar que o cliente foi avisado.
	if !h.svc.TemCanal() {
		resposta["aviso"] = "A mensagem entrou na fila, mas nenhum canal de envio está configurado ainda. " +
			"Ela será entregue assim que o canal for ligado, e é descartada se ficar mais de 7 dias esperando."
	}
	return escreverJSON(w, http.StatusCreated, resposta)
}

func traduzir(err error) error {
	switch {
	case errors.Is(err, ErrDisparoRecente):
		return problema.Conflito("Já houve uma cobrança para este cliente nos últimos 3 dias.")
	case errors.Is(err, ErrSemTelefone):
		return problema.UmCampo("dividaId", "O cliente não tem um telefone válido cadastrado.")
	case errors.Is(err, ErrSemCampanha):
		return problema.Conflito("Não há campanha ativa para a faixa de atraso desta dívida.")
	case errors.Is(err, ErrForaDaRegua):
		return problema.Conflito("Esta dívida está fora da régua de cobrança (3 a 90 dias de atraso).")
	case errors.Is(err, cobranca.ErrDividaNaoNegociavel):
		return problema.Conflito("Esta dívida não está aberta.")
	case errors.Is(err, cobranca.ErrNaoEncontrado):
		return problema.NaoEncontrado("Registro não encontrado.")
	default:
		return err
	}
}

func escreverJSON(w http.ResponseWriter, status int, corpo any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(corpo)
}
