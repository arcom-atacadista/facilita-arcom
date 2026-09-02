// Package servidor monta o roteador HTTP: middlewares, rotas, conversão de
// erro. É o único pacote que conhece chi — o resto do backend depende só do
// tipo Servidor e dos construtores de erro em problema.go.
package servidor

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	"github.com/go-chi/httprate"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"facilitaarcom/internal/acesso"
	"facilitaarcom/internal/cobranca"
	"facilitaarcom/internal/config"
	"facilitaarcom/internal/negociacao"
)

// Servidor guarda as dependências que os handlers precisam. Injeção manual
// por struct — sem framework de DI (ver padroes/03-backend.md).
type Servidor struct {
	cfg    *config.Config
	log    *slog.Logger
	db     *gorm.DB
	redis  *redis.Client
	router chi.Router

	// Serviços de feature. Ficam no Servidor porque rotas.go precisa deles
	// para montar a árvore, e o middleware de sessão precisa do de acesso.
	// Nulos quando o projeto sobe sem Postgres — rotas.go não monta as rotas
	// de negócio nesse caso.
	acesso     *acesso.Service
	cobranca   *cobranca.Service
	negociacao *negociacao.Service
}

// Novo monta o router com toda a stack de middleware e as rotas. Devolve
// http.Handler — cmd/server só chama Novo(...) e sobe um http.Server em cima.
func Novo(cfg *config.Config, log *slog.Logger, gdb *gorm.DB, rdb *redis.Client) http.Handler {
	s := &Servidor{cfg: cfg, log: log, db: gdb, redis: rdb, router: chi.NewRouter()}

	producao := cfg.Env == "production"

	if gdb != nil {
		s.acesso = acesso.NovoService(acesso.NovoRepo(gdb))

		repoCobranca := cobranca.NovoRepo(gdb)
		s.cobranca = cobranca.NovoService(repoCobranca, cfg.AppURL)
		s.negociacao = negociacao.NovoService(repoCobranca, s.cobranca)
	}

	// Ordem importa: request id primeiro (todo log downstream referencia
	// ele), recuperação de panic antes de qualquer coisa que possa
	// panicar, depois compressão/timeout/limites, headers por último antes
	// das rotas.
	s.router.Use(middleware.RequestID)
	// O backend nunca é alcançado direto da internet — o docker-compose só
	// publica a porta do frontend; a única forma de chegar aqui é pelo proxy
	// de /api do Vite, dentro da rede interna do compose (1 salto sempre
	// confiável). Por isso confiamos no X-Forwarded-For que esse proxy
	// escreve (ver frontend/vite.config.ts: proxy com `xfwd: true`) em vez do
	// IP da conexão TCP, que seria sempre o do container do frontend. Se o
	// backend um dia for exposto direto pra internet (sem esse proxy no
	// meio), isto vira spoofável — troque para middleware.ClientIPFromRemoteAddr.
	s.router.Use(middleware.ClientIPFromXFFTrustedProxies(1))
	s.router.Use(httplog.RequestLogger(log, &httplog.Options{
		Level:         slog.LevelInfo,
		RecoverPanics: false, // o Recoverer abaixo já cobre isso
	}))
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Compress(5))
	s.router.Use(middleware.Timeout(30 * time.Second))
	s.router.Use(limiteDeCorpo(1 << 20)) // 1 MB
	s.router.Use(httprate.LimitBy(100, time.Minute, chaveDeCliente))
	s.router.Use(s.origemPermitida)
	s.router.Use(headersSeguros(producao))

	s.rotas()

	return s.router
}

// chaveDeCliente é a chave usada pelo rate limit — o IP resolvido por
// middleware.ClientIPFromXFFTrustedProxies acima, canonicalizado (IPv6 vira
// prefixo /64, pra um cliente não escapar do limite trocando de endereço
// dentro do próprio /64).
func chaveDeCliente(r *http.Request) (string, error) {
	return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
}
