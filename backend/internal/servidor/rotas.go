package servidor

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// rotas monta toda a árvore de rotas num lugar só — mais fácil de auditar do
// que rota espalhada em vários arquivos. /api/health e /api/ready ficam fora
// da versão porque a infra depende deles; rotas de negócio entram sob
// /api/v1 (ver padroes/09-contrato-api.md).
func (s *Servidor) rotas() {
	s.router.Route("/api", func(api chi.Router) {
		api.Get("/health", s.saude)
		api.Get("/ready", s.H(s.pronto))

		api.Route("/v1", func(v1 chi.Router) {
			// Rotas de negócio do projeto entram aqui — uma feature por
			// v1.Route("/recurso", recurso.Rotas) (ver padroes/03-backend.md).
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
