// Package config lê a configuração do processo a partir do ambiente.
//
// Regra: falha rápido. Se uma variável obrigatória faltar ou for inválida, o
// processo recusa subir (erro no boot) em vez de rodar quebrado ou com um
// valor de desenvolvimento por engano. Em dev fora do Docker, carregue um
// `.env` com godotenv (ver cmd/server/main.go) antes de chamar Load.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config é a configuração já validada do processo.
type Config struct {
	// Ambiente: "development" (default) ou "production". Nunca decide
	// segurança sozinho (isso é feature flag, não gate) — só ajusta log e
	// mensagens de erro mais verbosas em dev.
	Env string

	Port string

	// Opcionais: string vazia = recurso não configurado. Um projeto que não
	// usa Postgres/Redis (ver padroes/07-frontend-primeiro.md) simplesmente
	// não define essas variáveis — o resto do backend não deve assumir que
	// elas existem.
	DatabaseURL string
	RedisURL    string

	// JWTSecret só é obrigatório quando a feature de autenticação existe no
	// projeto (ver .claude/skills/autenticacao). Enquanto não houver login,
	// fica vazio e ninguém lê.
	JWTSecret string
}

// Load lê e valida o ambiente. Retorna erro (em vez de sair do processo) para
// que quem chama decida como reportar — cmd/server e cmd/migrate imprimem e
// saem com status 1.
func Load() (*Config, error) {
	cfg := &Config{
		Env:         env("APP_ENV", "development"),
		Port:        env("PORT", "3000"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}

	var faltando []string

	// JWT_SECRET: se existe, tem que ser forte o bastante para HS256. Um
	// segredo curto sobe silenciosamente até alguém forjar um token.
	if cfg.JWTSecret != "" && len(cfg.JWTSecret) < 32 {
		faltando = append(faltando, "JWT_SECRET precisa ter pelo menos 32 caracteres (gere com: openssl rand -base64 48)")
	}

	if len(faltando) > 0 {
		return nil, fmt.Errorf("configuração inválida:\n  - %s", strings.Join(faltando, "\n  - "))
	}

	return cfg, nil
}

// Obrigatorio é o helper que uma feature usa quando precisa de uma variável
// que o core não exige (ex.: chave de API de terceiro). Uso:
//
//	chave, err := config.Obrigatorio("PAGAMENTO_API_KEY")
//	if err != nil { return nil, err } // acumule e falhe no boot, não em request
func Obrigatorio(chave string) (string, error) {
	v := os.Getenv(chave)
	if v == "" {
		return "", fmt.Errorf("variável de ambiente obrigatória ausente: %s", chave)
	}
	return v, nil
}

func env(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}
