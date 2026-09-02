package cobranca

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

// RotasCarteira são as dívidas em atraso. Todas privadas — o grupo que as
// monta já passou por ExigirSessao.
func (h *Handler) RotasCarteira(r chi.Router) {
	r.Get("/", h.Wrapper(h.listarDividas))
	r.Get("/{id}", h.Wrapper(h.verDivida))
	r.Post("/{id}/link", h.Wrapper(h.gerarLink))
}

func (h *Handler) RotasPoliticas(r chi.Router) {
	r.Get("/", h.Wrapper(h.listarPoliticas))
	// Mexer em desconto é decisão de régua: só coordenação pra cima.
	r.With(acesso.ExigirPapel(acesso.PapelCoordenacao)).Patch("/{id}", h.Wrapper(h.atualizarPolitica))
}

func (h *Handler) RotasAcordos(r chi.Router) {
	r.Get("/", h.Wrapper(h.listarAcordos))
	r.Post("/", h.Wrapper(h.fecharAcordo))
}

func (h *Handler) listarDividas(w http.ResponseWriter, r *http.Request) error {
	u, _ := acesso.DoContexto(r.Context())

	pagina, porPagina, err := paginacaoDaQuery(r)
	if err != nil {
		return err
	}

	f := FiltroDividas{
		Faixa:     Faixa(r.URL.Query().Get("faixa")),
		Status:    r.URL.Query().Get("status"),
		Busca:     r.URL.Query().Get("busca"),
		Pagina:    pagina,
		PorPagina: porPagina,
	}
	if f.Status != "" && !statusDeDividaValido(f.Status) {
		return problema.UmCampo("status", "Status inválido.")
	}

	itens, total, err := h.svc.ListarDividas(r.Context(), u, f)
	if err != nil {
		return err
	}
	return escreverLista(w, itens, pagina, porPagina, total)
}

func statusDeDividaValido(s string) bool {
	switch s {
	case StatusDividaAberta, StatusDividaNegociada, StatusDividaQuitada, StatusDividaCancelada:
		return true
	}
	return false
}

func (h *Handler) verDivida(w http.ResponseWriter, r *http.Request) error {
	u, _ := acesso.DoContexto(r.Context())
	id, err := idDaRota(r)
	if err != nil {
		return err
	}

	d, ofertas, err := h.svc.OfertasDaDivida(r.Context(), u, id)
	if err != nil {
		return traduzir(err)
	}

	return escreverJSON(w, http.StatusOK, map[string]any{
		"divida":  RespostaDeDivida(d, h.svc.agora()),
		"ofertas": ofertas,
	})
}

func (h *Handler) gerarLink(w http.ResponseWriter, r *http.Request) error {
	u, _ := acesso.DoContexto(r.Context())
	id, err := idDaRota(r)
	if err != nil {
		return err
	}

	link, err := h.svc.GerarLink(r.Context(), u, id)
	if err != nil {
		return traduzir(err)
	}
	return escreverJSON(w, http.StatusCreated, map[string]any{"link": link})
}

func (h *Handler) listarPoliticas(w http.ResponseWriter, r *http.Request) error {
	itens, err := h.svc.ListarPoliticas(r.Context())
	if err != nil {
		return err
	}
	return escreverJSON(w, http.StatusOK, map[string]any{"itens": itens})
}

func (h *Handler) atualizarPolitica(w http.ResponseWriter, r *http.Request) error {
	id, err := idDaRota(r)
	if err != nil {
		return err
	}

	var dto EntradaAtualizarPolitica
	if err := entrada.Decodificar(r, &dto); err != nil {
		return err
	}

	p, err := h.svc.AtualizarPolitica(r.Context(), id, dto)
	if err != nil {
		return traduzir(err)
	}
	return escreverJSON(w, http.StatusOK, RespostaDePolitica(p))
}

func (h *Handler) listarAcordos(w http.ResponseWriter, r *http.Request) error {
	u, _ := acesso.DoContexto(r.Context())
	pagina, porPagina, err := paginacaoDaQuery(r)
	if err != nil {
		return err
	}

	itens, total, err := h.svc.ListarAcordos(r.Context(), u, pagina, porPagina)
	if err != nil {
		return err
	}
	return escreverLista(w, itens, pagina, porPagina, total)
}

func (h *Handler) fecharAcordo(w http.ResponseWriter, r *http.Request) error {
	u, _ := acesso.DoContexto(r.Context())

	var dto EntradaFecharAcordo
	if err := entrada.Decodificar(r, &dto); err != nil {
		return err
	}

	a, err := h.svc.FecharAcordoOperador(r.Context(), u, dto)
	if err != nil {
		return traduzir(err)
	}
	return escreverJSON(w, http.StatusCreated, RespostaDeAcordo(a))
}

// --- auxiliares ---

func idDaRota(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		// Id malformado e id inexistente respondem igual: não revela quais
		// ids existem.
		return uuid.Nil, problema.NaoEncontrado("Registro não encontrado.")
	}
	return id, nil
}

// paginacaoDaQuery lê pagina/porPagina do contrato. porPagina acima do máximo
// é recusado, não truncado em silêncio (09-contrato-api.md).
func paginacaoDaQuery(r *http.Request) (int, int, error) {
	pagina := 1
	if v := r.URL.Query().Get("pagina"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return 0, 0, problema.UmCampo("pagina", "Deve ser um número inteiro maior que zero.")
		}
		pagina = n
	}

	porPagina := PorPaginaPadrao
	if v := r.URL.Query().Get("porPagina"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return 0, 0, problema.UmCampo("porPagina", "Deve ser um número inteiro maior que zero.")
		}
		if n > PorPaginaMaximo {
			return 0, 0, problema.UmCampo("porPagina",
				"Máximo de "+strconv.Itoa(PorPaginaMaximo)+" registros por página.")
		}
		porPagina = n
	}
	return pagina, porPagina, nil
}

func traduzir(err error) error {
	var campo *ErroDeCampoCobranca
	var alcada *ErroAlcada
	switch {
	case errors.As(err, &campo):
		return problema.UmCampo(campo.Campo, campo.Mensagem)
	case errors.As(err, &alcada):
		return &problema.ErroDominio{
			Status: http.StatusForbidden,
			Titulo: "Acima da alçada",
			Detail: alcada.Error() + ". Peça aprovação à coordenação.",
			Codigo: "acima_da_alcada",
		}
	case errors.Is(err, ErrAcordoJaExiste):
		return problema.Conflito("Já existe um acordo ativo para este contrato.")
	case errors.Is(err, ErrDividaNaoNegociavel):
		return problema.Conflito("Esta dívida não está aberta para negociação.")
	case errors.Is(err, ErrNaoEncontrado):
		// Fora do escopo do usuário cai aqui também — e responder 404 em vez
		// de 403 é proposital: não revela que a dívida existe na carteira de
		// outra pessoa (09-contrato-api.md).
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

func escreverLista[T any](w http.ResponseWriter, itens []T, pagina, porPagina int, total int64) error {
	totalPaginas := int((total + int64(porPagina) - 1) / int64(porPagina))
	return escreverJSON(w, http.StatusOK, map[string]any{
		"itens": itens,
		"paginacao": map[string]any{
			"pagina":       pagina,
			"porPagina":    porPagina,
			"total":        total,
			"totalPaginas": totalPaginas,
		},
	})
}
