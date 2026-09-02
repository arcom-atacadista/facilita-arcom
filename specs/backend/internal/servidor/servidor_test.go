package servidor_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"meu-projeto/internal/config"
	"meu-projeto/internal/servidor"
)

// novoServidorDeTeste monta o handler completo (router + middlewares) sem
// Postgres/Redis — testa o comportamento do servidor de ponta a ponta
// (httptest bate no http.Handler real, não numa função isolada).
func novoServidorDeTeste(t *testing.T) http.Handler {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{Env: "development", Port: "3000"}
	return servidor.Novo(cfg, log, nil, nil)
}

func TestRotasBasicas(t *testing.T) {
	casos := []struct {
		nome       string
		metodo     string
		caminho    string
		statusQuer int
	}{
		{"liveness sempre ok", http.MethodGet, "/api/health", http.StatusOK},
		{"readiness sem dependencias configuradas", http.MethodGet, "/api/ready", http.StatusOK},
		{"rota inexistente vira problem+json 404", http.MethodGet, "/api/v1/naoexiste", http.StatusNotFound},
		{"metodo errado vira problem+json 405", http.MethodPut, "/api/health", http.StatusMethodNotAllowed},
	}

	handler := novoServidorDeTeste(t)

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			req := httptest.NewRequest(c.metodo, c.caminho, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != c.statusQuer {
				t.Fatalf("status = %d, quer %d (corpo: %s)", rec.Code, c.statusQuer, rec.Body.String())
			}
		})
	}
}

func TestErroSaiNoFormatoDoContrato(t *testing.T) {
	handler := novoServidorDeTeste(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/naoexiste", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type = %q, quer application/problem+json", ct)
	}

	var problema struct {
		Status int    `json:"status"`
		Codigo string `json:"codigo"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &problema); err != nil {
		t.Fatalf("corpo não é o Problema esperado: %v (%s)", err, rec.Body.String())
	}
	if problema.Codigo != "nao_encontrado" {
		t.Fatalf("codigo = %q, quer nao_encontrado", problema.Codigo)
	}
	if problema.Detail == "" {
		t.Fatal("detail vazio — mensagem pro usuário nunca pode faltar")
	}
}

func TestOrigemInvalidaBloqueiaEscritaCrossOrigin(t *testing.T) {
	handler := novoServidorDeTeste(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/qualquer", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, quer %d (guard de CSRF deveria barrar origem estranha)", rec.Code, http.StatusForbidden)
	}
}

func TestHeadersDeSegurancaSempreParticipam(t *testing.T) {
	handler := novoServidorDeTeste(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	for header, esperado := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	} {
		if got := rec.Header().Get(header); got != esperado {
			t.Errorf("header %s = %q, quer %q", header, got, esperado)
		}
	}
}
