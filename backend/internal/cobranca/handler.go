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
	// A posição consolidada é do cliente, não de um título — é o que a mesa de
	// atendimento consulta para montar as condições na conversa.
	r.Get("/clientes/{id}/ofertas", h.Wrapper(h.ofertasDoCliente))
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

	d, posicao, ofertas, err := h.svc.OfertasDaDivida(r.Context(), u, id)
	if err != nil {
		return traduzir(err)
	}

	// A posição vai junto porque a condição oferecida é do CNPJ inteiro, não
	// deste título: sem ela a tela mostraria um desconto que não fecha com o
	// valor do documento aberto.
	return escreverJSON(w, http.StatusOK, map[string]any{
		"divida":  RespostaDeDivida(d, h.svc.agora()),
		"posicao": posicao,
		"ofertas": ofertas,
	})
}

// ofertasDoCliente devolve a posição em atraso do CNPJ e as condições
// calculadas para a alçada de quem consulta.
func (h *Handler) ofertasDoCliente(w http.ResponseWriter, r *http.Request) error {
	u, _ := acesso.DoContexto(r.Context())
	id, err := idDaRota(r)
	if err != nil {
		return err
	}

	posicao, ofertas, acordo, err := h.svc.OfertasDoCliente(r.Context(), u, id)
	if err != nil {
		return traduzir(err)
	}

	return escreverJSON(w, http.StatusOK, map[string]any{
		"posicao":     posicao,
		"ofertas":     ofertas,
		"acordoAtivo": acordo,
		// A alçada vai na resposta para a tela poder dizer "sua alçada" na
		// condição que ela cobre, sem precisar adivinhar pelo papel.
		"alcadaMaxima": u.AlcadaMaxima,
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
		// O acordo agora é do CNPJ, então "já existe" fala do cliente e não de
		// um contrato — a mensagem antiga mandaria o operador procurar no título
		// errado.
		return problema.Conflito("Já existe um acordo ativo para este cliente.")
	case errors.Is(err, ErrSemPosicaoAberta):
		// Cliente sem título aberto e cliente fora do escopo respondem igual,
		// pela mesma razão do ErrNaoEncontrado abaixo.
		return problema.NaoEncontrado("Nenhum título em atraso aberto para este cliente.")
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
