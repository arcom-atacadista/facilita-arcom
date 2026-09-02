package acesso

import (
	"net/http"

	"facilitaarcom/internal/problema"
)

// ExigirSessao recusa qualquer requisição sem cookie de sessão válido e
// coloca o usuário no contexto. Aplique sempre num grupo de rotas
// (chi.Router.Group), nunca rota por rota — é fácil esquecer uma.
func (s *Service) ExigirSessao(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(NomeCookie)
		if err != nil {
			_ = problema.EscreverErro(w, problema.NaoAutenticado("Faça login para continuar."))
			return
		}

		usuario, err := s.UsuarioDaSessao(r.Context(), cookie.Value)
		if err != nil {
			// Sessão inexistente, expirada ou de usuário desativado caem no
			// mesmo 401: o cliente só precisa saber que tem que logar de novo.
			_ = problema.EscreverErro(w, problema.SessaoExpirada())
			return
		}

		next.ServeHTTP(w, r.WithContext(ComUsuario(r.Context(), usuario)))
	})
}

// ExigirPapel barra quem está autenticado mas não tem nível suficiente.
// Sempre depois de ExigirSessao na cadeia — sem usuário no contexto, recusa.
func ExigirPapel(minimo Papel) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			usuario, ok := DoContexto(r.Context())
			if !ok {
				_ = problema.EscreverErro(w, problema.NaoAutenticado("Faça login para continuar."))
				return
			}
			if !usuario.Papel.PeloMenos(minimo) {
				_ = problema.EscreverErro(w, problema.SemPermissao())
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
