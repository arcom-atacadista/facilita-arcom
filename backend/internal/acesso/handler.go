package acesso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"

	"facilitaarcom/internal/entrada"
	"facilitaarcom/internal/problema"
)

// NomeCookie é o cookie de sessão. Path /api porque o frontend é servido
// estático pelo nginx — o cookie não precisa acompanhar cada .js e .css.
const NomeCookie = "sessao"

type chaveContexto struct{}

// ComUsuario guarda o usuário autenticado no contexto da requisição.
func ComUsuario(ctx context.Context, u Usuario) context.Context {
	return context.WithValue(ctx, chaveContexto{}, u)
}

// DoContexto devolve o usuário autenticado. O segundo retorno é falso em rota
// pública — nenhum handler deve assumir que há alguém logado.
func DoContexto(ctx context.Context) (Usuario, bool) {
	u, ok := ctx.Value(chaveContexto{}).(Usuario)
	return u, ok
}

// Handler expõe as rotas de sessão e de usuários.
type Handler struct {
	svc     *Service
	seguro  bool // Secure no cookie: ligado em produção (HTTPS de verdade)
	Wrapper func(func(http.ResponseWriter, *http.Request) error) http.HandlerFunc
}

func NovoHandler(svc *Service, producao bool, wrapper func(func(http.ResponseWriter, *http.Request) error) http.HandlerFunc) *Handler {
	return &Handler{svc: svc, seguro: producao, Wrapper: wrapper}
}

// RotasSessao são públicas: login e primeiro acesso. Rate limit próprio e
// bem mais apertado que o global — é a rota que sofre força bruta.
func (h *Handler) RotasSessao(r chi.Router) {
	r.Group(func(g chi.Router) {
		g.Use(httprate.LimitByIP(10, time.Minute))
		g.Post("/", h.Wrapper(h.login))
		g.Post("/primeiro-acesso", h.Wrapper(h.primeiroAcesso))
	})
	r.Post("/logout", h.Wrapper(h.logout))
	r.Get("/", h.Wrapper(h.quemSouEu))
}

// RotasUsuarios são privadas — montadas dentro do grupo com ExigirSessao.
func (h *Handler) RotasUsuarios(r chi.Router) {
	r.Get("/", h.Wrapper(h.listar))
	r.Group(func(g chi.Router) {
		g.Use(ExigirPapel(PapelCoordenacao))
		g.Post("/", h.Wrapper(h.criar))
		g.Patch("/{id}", h.Wrapper(h.atualizar))
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) error {
	var dto EntradaLogin
	if err := entrada.Decodificar(r, &dto); err != nil {
		return err
	}

	usuario, token, err := h.svc.Autenticar(r.Context(), dto)
	if err != nil {
		if errors.Is(err, ErrCredenciaisInvalidas) {
			// Mensagem única: não revela se o e-mail existe nem se a conta
			// está desativada.
			return &problema.ErroDominio{
				Status: http.StatusUnauthorized,
				Titulo: "Não autenticado",
				Detail: "E-mail ou senha inválidos.",
				Codigo: "credenciais_invalidas",
			}
		}
		return err
	}

	h.definirCookie(w, token, DuracaoSessao)
	return escreverJSON(w, http.StatusOK, Responder(usuario))
}

func (h *Handler) primeiroAcesso(w http.ResponseWriter, r *http.Request) error {
	var dto EntradaCriarUsuario
	if err := entrada.Decodificar(r, &dto); err != nil {
		return err
	}

	usuario, err := h.svc.PrimeiroAcesso(r.Context(), dto)
	if err != nil {
		return traduzir(err)
	}
	return escreverJSON(w, http.StatusCreated, Responder(usuario))
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) error {
	if c, err := r.Cookie(NomeCookie); err == nil {
		if err := h.svc.Encerrar(r.Context(), c.Value); err != nil {
			return err
		}
	}
	// Expira o cookie mesmo se não havia sessão: logout é idempotente.
	h.definirCookie(w, "", -time.Second)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *Handler) quemSouEu(w http.ResponseWriter, r *http.Request) error {
	c, err := r.Cookie(NomeCookie)
	if err != nil {
		return problema.NaoAutenticado("Faça login para continuar.")
	}
	usuario, err := h.svc.UsuarioDaSessao(r.Context(), c.Value)
	if err != nil {
		return problema.SessaoExpirada()
	}
	return escreverJSON(w, http.StatusOK, Responder(usuario))
}

func (h *Handler) listar(w http.ResponseWriter, r *http.Request) error {
	usuarios, err := h.svc.Listar(r.Context())
	if err != nil {
		return err
	}
	itens := make([]UsuarioResposta, 0, len(usuarios))
	for _, u := range usuarios {
		itens = append(itens, Responder(u))
	}
	return escreverJSON(w, http.StatusOK, map[string]any{"itens": itens})
}

func (h *Handler) criar(w http.ResponseWriter, r *http.Request) error {
	var dto EntradaCriarUsuario
	if err := entrada.Decodificar(r, &dto); err != nil {
		return err
	}

	quemPede, _ := DoContexto(r.Context())
	// Coordenação não cria gerência: ninguém cria alguém acima de si.
	if dto.Papel.Nivel() > quemPede.Papel.Nivel() {
		return problema.UmCampo("papel", "Você não pode criar um usuário com papel acima do seu.")
	}

	usuario, err := h.svc.Criar(r.Context(), dto)
	if err != nil {
		return traduzir(err)
	}
	return escreverJSON(w, http.StatusCreated, Responder(usuario))
}

func (h *Handler) atualizar(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return problema.NaoEncontrado("Usuário não encontrado.")
	}

	var dto EntradaAtualizarUsuario
	if err := entrada.Decodificar(r, &dto); err != nil {
		return err
	}

	quemPede, _ := DoContexto(r.Context())
	usuario, err := h.svc.Atualizar(r.Context(), quemPede, id, dto)
	if err != nil {
		return traduzir(err)
	}
	return escreverJSON(w, http.StatusOK, Responder(usuario))
}

func (h *Handler) definirCookie(w http.ResponseWriter, valor string, duracao time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     NomeCookie,
		Value:    valor,
		Path:     "/api",
		HttpOnly: true,
		// Em dev o front roda em http://localhost — Secure impediria o
		// navegador de guardar o cookie e o login nunca funcionaria.
		Secure:   h.seguro,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(duracao.Seconds()),
	})
}

// traduzir converte os erros de domínio do service no formato do contrato.
func traduzir(err error) error {
	var campo *ErroDeCampo
	switch {
	case errors.As(err, &campo):
		return problema.UmCampo(campo.Campo, campo.Mensagem)
	case errors.Is(err, ErrEmailEmUso):
		return problema.Conflito("Já existe um usuário com este e-mail.")
	case errors.Is(err, ErrJaInicializado):
		return problema.Conflito("O sistema já tem usuários cadastrados. Peça seu acesso à coordenação.")
	case errors.Is(err, ErrAutoPrivilegio):
		return problema.SemPermissao()
	case errors.Is(err, ErrPapelAcimaDoSeu):
		return problema.UmCampo("papel", "Você não pode conceder um papel acima do seu.")
	case errors.Is(err, ErrNaoEncontrado):
		return problema.NaoEncontrado("Usuário não encontrado.")
	default:
		return err
	}
}

func escreverJSON(w http.ResponseWriter, status int, corpo any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(corpo)
}
