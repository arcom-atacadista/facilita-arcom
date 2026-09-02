package servidor

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"facilitaarcom/internal/acesso"
	"facilitaarcom/internal/cobranca"
	"facilitaarcom/internal/conversa"
	"facilitaarcom/internal/disparo"
	"facilitaarcom/internal/negociacao"
)

// rotas monta toda a árvore de rotas num lugar só — mais fácil de auditar do
// que rota espalhada em vários arquivos. /api/health e /api/ready ficam fora
// da versão porque a infra depende deles; rotas de negócio entram sob
// /api/v1 (ver system-design/padroes/09-contrato-api.md).
func (s *Servidor) rotas() {
	s.router.Route("/api", func(api chi.Router) {
		api.Get("/health", s.saude)
		api.Get("/ready", s.H(s.pronto))

		api.Route("/v1", func(v1 chi.Router) {
			// Sem Postgres não existe cobrança nem sessão — o projeto sobe
			// só com health/ready em vez de estourar 500 em cada rota.
			if s.db == nil {
				return
			}
			s.rotasDeNegocio(v1)
		})
	})

	// Rota inexistente/método errado também sai no formato do contrato — o
	// front nunca recebe um 404 de texto puro do chi.
	s.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		s.escreverProblema(w, r, http.StatusNotFound, "Não encontrado", "Rota não encontrada.", "nao_encontrado", nil)
	})
	s.router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		s.escreverProblema(w, r, http.StatusMethodNotAllowed, "Método não permitido", "Método não permitido para esta rota.", "metodo_nao_permitido", nil)
	})
}

func (s *Servidor) rotasDeNegocio(v1 chi.Router) {
	producao := s.cfg.Env == "production"
	hAcesso := acesso.NovoHandler(s.acesso, producao, s.H)
	hCobranca := cobranca.NovoHandler(s.cobranca, s.H)
	hNegociacao := negociacao.NovoHandler(s.negociacao, s.H)
	hDisparo := disparo.NovoHandler(s.disparo, s.H)
	hConversa := conversa.NovoHandler(s.conversa, s.H)

	// Público. /sessao é login/logout/quem-sou-eu; /negociar é a tela que o
	// cliente devedor abre pelo link do WhatsApp, sem conta — quem autoriza
	// ali é a posse do token.
	v1.Route("/sessao", hAcesso.RotasSessao)
	v1.Route("/negociar", hNegociacao.Rotas)

	// O webhook da Meta é público por natureza — quem autoriza é a assinatura
	// do corpo, não uma sessão. Só é montado quando há segredo configurado.
	if s.webhook.Configurado() {
		v1.Group(func(g chi.Router) {
			// Limite próprio e folgado: a Meta agrupa eventos, e descartar um
			// evento é perder a resposta de um cliente.
			g.Use(httprate.LimitByIP(1200, time.Minute))
			s.webhook.Rotas(g)
		})
	}

	// Privado: tudo daqui pra baixo exige cookie de sessão válido.
	v1.Group(func(g chi.Router) {
		g.Use(s.acesso.ExigirSessao)

		g.Route("/usuarios", hAcesso.RotasUsuarios)
		g.Route("/carteira", hCobranca.RotasCarteira)
		g.Route("/politicas", hCobranca.RotasPoliticas)
		g.Route("/acordos", hCobranca.RotasAcordos)
		g.Route("/disparos", hDisparo.Rotas)
		g.Route("/conversas", hConversa.Rotas)
	})
}
