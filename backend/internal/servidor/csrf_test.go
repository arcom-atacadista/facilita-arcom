package servidor_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"facilitaarcom/internal/acesso"
)

// O guard de CSRF precisa aceitar a requisição legítima nos dois ambientes e
// recusar a de terceiro. O caso do meio é o que já quebrou: em dev o proxy do
// Vite reescreve o Host, e comparar Origin só com r.Host recusava toda
// escrita vinda do navegador — inclusive o login.
func TestGuardDeCSRF(t *testing.T) {
	casos := []struct {
		nome       string
		headers    map[string]string
		host       string
		querRecusa bool
	}{
		{
			nome:    "produção: nginx preserva o Host, Origin bate",
			host:    "facilita.arcom.com.br",
			headers: map[string]string{"Origin": "https://facilita.arcom.com.br"},
		},
		{
			nome: "dev: proxy do Vite reescreve o Host e encaminha o original",
			host: "localhost:3000",
			headers: map[string]string{
				"Origin":           "http://127.0.0.1:8080",
				"X-Forwarded-Host": "127.0.0.1:8080",
			},
		},
		{
			nome:    "navegador moderno: Sec-Fetch-Site same-origin passa",
			host:    "localhost:3000",
			headers: map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "http://127.0.0.1:8080"},
		},
		{
			nome:       "site de terceiro: Sec-Fetch-Site cross-site é recusado",
			host:       "facilita.arcom.com.br",
			headers:    map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example.com"},
			querRecusa: true,
		},
		{
			nome:       "site de terceiro sem Sec-Fetch-Site: Origin não bate",
			host:       "facilita.arcom.com.br",
			headers:    map[string]string{"Origin": "https://evil.example.com"},
			querRecusa: true,
		},
		{
			// Um atacante que só controlasse o Origin não passa: o
			// X-Forwarded-Host teria que casar, e página de navegador não
			// consegue definir esse header sem cair no preflight de CORS.
			nome: "Origin forjado com X-Forwarded-Host legítimo é recusado",
			host: "localhost:3000",
			headers: map[string]string{
				"Origin":           "https://evil.example.com",
				"X-Forwarded-Host": "127.0.0.1:8080",
			},
			querRecusa: true,
		},
		{
			nome:    "cliente sem navegador (curl, health check) não tem Origin",
			host:    "localhost:3000",
			headers: map[string]string{},
		},
	}

	handler := novoServidorDeTeste(t)

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			// Rota inexistente de propósito: interessa só se o guard deixou
			// passar (404 do router) ou barrou antes (403).
			req := httptest.NewRequest(http.MethodPost, "/api/v1/qualquer", nil)
			req.Host = c.host
			for k, v := range c.headers {
				req.Header.Set(k, v)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			recusou := rec.Code == http.StatusForbidden
			if recusou != c.querRecusa {
				t.Fatalf("status = %d (recusou=%v), queria recusa=%v — corpo: %s",
					rec.Code, recusou, c.querRecusa, rec.Body.String())
			}
		})
	}
}

// GET nunca passa pelo guard: leitura não muda estado.
func TestGuardDeCSRFNaoAtrapalhaLeitura(t *testing.T) {
	handler := novoServidorDeTeste(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, quer 200", rec.Code)
	}
}

// O custo reduzido é ferramenta de teste; o valor que vai para produção não
// pode mudar por acidente.
func TestCustoDeHashDeProducao(t *testing.T) {
	if acesso.CustoBcryptPadrao != 12 {
		t.Fatalf("CustoBcryptPadrao = %d, quer 12 (exigência de 03-backend.md)",
			acesso.CustoBcryptPadrao)
	}
}
