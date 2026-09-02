package servidor

import (
	"net/http"
	"net/url"
	"strings"
)

// headersSeguros aplica um piso de defesas em toda resposta da API. Não
// substitui as regras da skill `seguranca`, mas garante que nenhuma rota saia
// sem o básico.
func headersSeguros(producao bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff") // não "adivinhe" o content-type
			h.Set("X-Frame-Options", "DENY")           // não deixa embutir em iframe (clickjacking)
			h.Set("Referrer-Policy", "no-referrer")    // não vaza a URL interna no Referer
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			if producao {
				// só faz sentido atrás de HTTPS de verdade (reverse proxy da infra)
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// limiteDeCorpo recusa corpo de requisição maior que limite bytes. Sem isso,
// um cliente mal-intencionado pode mandar um corpo gigante e esgotar memória
// antes mesmo da validação rodar.
func limiteDeCorpo(limite int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limite)
			next.ServeHTTP(w, r)
		})
	}
}

var metodosQueMudamEstado = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// origemPermitida é o guard de CSRF. A sessão vive em cookie
// (HttpOnly; SameSite=Lax), então o navegador manda o cookie automaticamente
// em qualquer requisição de qualquer origem — SameSite=Lax barra a maioria,
// mas isto aqui é a segunda camada, em todo método que muda estado.
//
// A checagem principal é o Sec-Fetch-Site, que o navegador preenche e nenhum
// JavaScript de página consegue forjar (é header proibido). "same-origin" é
// a nossa requisição; "none" é navegação digitada na barra de endereço.
//
// O fallback por Origin existe para cliente sem Sec-Fetch-Site, e compara
// contra o host que o NAVEGADOR usou — que nem sempre é r.Host:
//
//   - produção: o nginx faz proxy_set_header Host $host, então r.Host já é o
//     host do navegador e a comparação direta funciona;
//   - desenvolvimento: o proxy do Vite usa changeOrigin: true e reescreve o
//     Host para localhost:3000, enquanto o Origin continua sendo o endereço
//     que a pessoa abriu. Comparar só com r.Host recusaria TODA escrita vinda
//     do navegador em dev (login inclusive). Com xfwd: true, esse mesmo proxy
//     preenche X-Forwarded-Host com o host original — é ele que vale aqui.
//
// Confiar no X-Forwarded-Host não abre brecha nova: o backend nunca é
// exposto direto (só o frontend publica porta), é a mesma premissa de um
// salto confiável que já vale para o X-Forwarded-For, e uma página de
// terceiro não consegue definir esse header numa requisição de navegador sem
// cair no preflight de CORS, que não passa.
func (s *Servidor) origemPermitida(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !metodosQueMudamEstado[r.Method] {
			next.ServeHTTP(w, r)
			return
		}

		switch r.Header.Get("Sec-Fetch-Site") {
		case "same-origin", "none":
			next.ServeHTTP(w, r)
			return
		case "cross-site", "same-site":
			s.recusarOrigem(w, r)
			return
		}

		// Sem Sec-Fetch-Site: cai no Origin.
		origem := r.Header.Get("Origin")
		if origem == "" {
			// Requisição sem Origin não vem de página de navegador (curl, app
			// interno, health check). SameSite=Lax já barra o cookie no caso
			// que interessa.
			next.ServeHTTP(w, r)
			return
		}

		u, err := url.Parse(origem)
		if err != nil || !strings.EqualFold(u.Host, hostDoNavegador(r)) {
			s.recusarOrigem(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// hostDoNavegador devolve o host que o navegador realmente acessou. Ver o
// comentário de origemPermitida para o porquê do X-Forwarded-Host.
func hostDoNavegador(r *http.Request) string {
	if encaminhado := r.Header.Get("X-Forwarded-Host"); encaminhado != "" {
		// Uma cadeia de proxies acumula "a, b, c" — o primeiro é o original.
		if virgula := strings.IndexByte(encaminhado, ','); virgula >= 0 {
			encaminhado = encaminhado[:virgula]
		}
		return strings.TrimSpace(encaminhado)
	}
	return r.Host
}

func (s *Servidor) recusarOrigem(w http.ResponseWriter, r *http.Request) {
	s.escreverProblema(w, r, http.StatusForbidden,
		"Origem não permitida", "Requisição de origem não confiável.", "origem_invalida", nil)
}
