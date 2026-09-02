// Package config lê a configuração do processo a partir do ambiente.
//
// Regra: falha rápido. Se uma variável obrigatória faltar ou for inválida, o
// processo recusa subir (erro no boot) em vez de rodar quebrado ou com um
// valor de desenvolvimento por engano. Em dev fora do Docker, carregue um
// `.env` com godotenv (ver cmd/server/main.go) antes de chamar Load.
package config

import (
	"fmt"
	"net/url"
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
	//
	// O Facilita ARCOM não usa JWT: a sessão é token opaco registrado no
	// Postgres (ver internal/acesso). A variável segue aqui porque é do
	// template e a validação de tamanho continua valendo se alguém definir.
	JWTSecret string

	// GatewayArcomAPIKey é a credencial do Gateway de dados da ARCOM. Vazia
	// em dev de propósito: a chave só existe em produção, provisionada pelo
	// time de TI (ver system-design/padroes/12-gateway-arcom.md). Sem ela a
	// consulta ao Gateway falha com uma mensagem clara, e o resto do sistema
	// sobe normalmente.
	GatewayArcomAPIKey string

	// WhatsApp: credenciais da Cloud API da Meta, provisionadas junto com a
	// conta WhatsApp Business da ARCOM. Vazias em dev de propósito — sem
	// elas o sistema sobe normalmente, a fila de disparo funciona e nada é
	// enviado nem marcado como enviado (ver internal/disparo/canal.go).
	WhatsAppIDNumero  string
	WhatsAppToken     string
	WhatsAppVersaoAPI string
	// Segredo da aplicação, usado para conferir a assinatura dos webhooks.
	WhatsAppAppSecret string
	// Token combinado com a Meta no cadastro do webhook, conferido no
	// handshake de verificação.
	WhatsAppVerifyToken string

	// Os dois campos do dataset `debitos` cuja semântica a documentação do
	// Gateway ainda marca como "a revisar". Configuráveis para que a resposta
	// do time de TI seja troca de variável de ambiente, e não novo deploy de
	// código (ver internal/gatewayarcom/debitos.go).
	GatewayCampoVencimento string
	GatewayCampoValor      string

	// Horário (cron, em UTC) da sincronização diária da carteira.
	SincronizacaoHorario string

	// AppURL é o endereço público da aplicação, usado para montar o link de
	// negociação que vai na mensagem ao cliente. Precisa ser absoluto: o link
	// é aberto do WhatsApp, fora do contexto do site.
	AppURL string
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
		AppURL:      env("APP_URL", "http://localhost:8080"),

		GatewayArcomAPIKey:     os.Getenv("GATEWAY_ARCOM_API_KEY"),
		GatewayCampoVencimento: os.Getenv("GATEWAY_CAMPO_VENCIMENTO"),
		GatewayCampoValor:      os.Getenv("GATEWAY_CAMPO_VALOR"),
		SincronizacaoHorario:   os.Getenv("SINCRONIZACAO_HORARIO"),

		WhatsAppIDNumero:    os.Getenv("WHATSAPP_PHONE_NUMBER_ID"),
		WhatsAppToken:       os.Getenv("WHATSAPP_ACCESS_TOKEN"),
		WhatsAppVersaoAPI:   os.Getenv("WHATSAPP_API_VERSION"),
		WhatsAppAppSecret:   os.Getenv("WHATSAPP_APP_SECRET"),
		WhatsAppVerifyToken: os.Getenv("WHATSAPP_VERIFY_TOKEN"),
	}

	var faltando []string

	// JWT_SECRET: se existe, tem que ser forte o bastante para HS256. Um
	// segredo curto sobe silenciosamente até alguém forjar um token.
	if cfg.JWTSecret != "" && len(cfg.JWTSecret) < 32 {
		faltando = append(faltando, "JWT_SECRET precisa ter pelo menos 32 caracteres (gere com: openssl rand -base64 48)")
	}

	// Link de negociação com endereço relativo ou malformado chega quebrado no
	// WhatsApp do cliente — melhor recusar subir do que descobrir depois.
	if u, err := url.Parse(cfg.AppURL); err != nil || u.Scheme == "" || u.Host == "" {
		faltando = append(faltando, "APP_URL precisa ser uma URL absoluta (ex.: https://facilita.arcom.com.br)")
	}

	// As credenciais do WhatsApp vêm em conjunto: com uma só delas, o envio
	// falharia em toda mensagem no meio da operação. Melhor recusar subir.
	temIDNumero, temToken := cfg.WhatsAppIDNumero != "", cfg.WhatsAppToken != ""
	if temIDNumero != temToken {
		faltando = append(faltando,
			"WHATSAPP_PHONE_NUMBER_ID e WHATSAPP_ACCESS_TOKEN andam juntos: defina as duas ou nenhuma")
	}
	// Webhook sem segredo de assinatura aceitaria evento forjado de qualquer
	// origem; sem verify token o cadastro na Meta nem se completa.
	if cfg.WhatsAppAppSecret != "" && cfg.WhatsAppVerifyToken == "" {
		faltando = append(faltando, "WHATSAPP_APP_SECRET definido exige WHATSAPP_VERIFY_TOKEN")
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
