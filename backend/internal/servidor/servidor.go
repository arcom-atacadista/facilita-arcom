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
	"facilitaarcom/internal/carteira"
	"facilitaarcom/internal/cobranca"
	"facilitaarcom/internal/config"
	"facilitaarcom/internal/conversa"
	"facilitaarcom/internal/disparo"
	"facilitaarcom/internal/gatewayarcom"
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
	disparo    *disparo.Service
	conversa   *conversa.Service
	webhook    *conversa.Webhook
	gateway    *gatewayarcom.Cliente

	carteira *carteira.Service

	// Os workers ficam expostos para cmd/server iniciar e parar junto com o
	// http.Server — o servidor monta, quem controla o ciclo de vida é o main.
	Worker         *disparo.Worker
	WorkerCarteira *carteira.Worker
}

// Novo monta o router e devolve http.Handler — é o que os testes usam quando
// só interessa bater nas rotas.
func Novo(cfg *config.Config, log *slog.Logger, gdb *gorm.DB, rdb *redis.Client) http.Handler {
	return Montar(cfg, log, gdb, rdb)
}

// Montar devolve o *Servidor concreto. cmd/server usa esta versão porque
// precisa do Worker de disparo para iniciar e parar junto com o http.Server —
// o servidor monta as dependências, quem controla ciclo de vida é o main.
func Montar(cfg *config.Config, log *slog.Logger, gdb *gorm.DB, rdb *redis.Client) *Servidor {
	s := &Servidor{cfg: cfg, log: log, db: gdb, redis: rdb, router: chi.NewRouter()}

	producao := cfg.Env == "production"

	if gdb != nil {
		s.acesso = acesso.NovoService(acesso.NovoRepo(gdb))

		repoCobranca := cobranca.NovoRepo(gdb)
		s.cobranca = cobranca.NovoService(repoCobranca, cfg.AppURL)
		s.negociacao = negociacao.NovoService(repoCobranca, s.cobranca)

		// O Gateway é opcional: sem GATEWAY_ARCOM_API_KEY o client existe e
		// cada chamada devolve ErrSemCredencial, que é o esperado em dev.
		s.gateway = gatewayarcom.NovoCliente(cfg.GatewayArcomAPIKey)
		if cfg.GatewayBaseURL != "" {
			s.gateway = s.gateway.ComBaseURL(cfg.GatewayBaseURL)
		}

		mapeamento := gatewayarcom.MapeamentoPadrao()
		if cfg.GatewayCampoVencimento != "" {
			mapeamento.Vencimento = gatewayarcom.CampoVencimento(cfg.GatewayCampoVencimento)
		}
		if cfg.GatewayCampoValor != "" {
			mapeamento.Valor = gatewayarcom.CampoValor(cfg.GatewayCampoValor)
		}
		if err := mapeamento.Valido(); err != nil {
			// Campo com erro de digitação traria carteira vazia em silêncio;
			// melhor voltar ao padrão e avisar alto.
			log.Error("mapeamento do Gateway inválido — usando o padrão", "erro", err)
			mapeamento = gatewayarcom.MapeamentoPadrao()
		}

		s.carteira = carteira.NovoService(gdb, s.gateway, mapeamento, log)
		s.WorkerCarteira = carteira.NovoWorker(s.carteira, cfg.SincronizacaoHorario, log)

		// O canal só existe quando as credenciais da Meta estão configuradas.
		// Sem elas o serviço roda com canal nulo: a fila funciona, nada é
		// entregue e — importante — nada é marcado como entregue.
		var canal disparo.Canal
		if cfg.WhatsAppIDNumero != "" && cfg.WhatsAppToken != "" {
			c, err := disparo.NovoCanalMeta(disparo.ConfigMeta{
				IDNumero:  cfg.WhatsAppIDNumero,
				Token:     cfg.WhatsAppToken,
				VersaoAPI: cfg.WhatsAppVersaoAPI,
				BaseURL:   cfg.WhatsAppBaseURL,
			})
			if err != nil {
				// Config.Load já garante que as duas variáveis vêm juntas, então
				// chegar aqui com erro é bug de programação, não de ambiente.
				log.Error("canal do WhatsApp mal configurado — a fila vai encher sem enviar", "erro", err)
			} else {
				canal = c
			}
		}

		// A mesa responde pelo mesmo canal da régua. Sem credencial da Meta ela
		// fica em modo leitura: mostra o que o cliente escreveu e recusa a
		// resposta com erro claro. O if evita guardar uma interface não-nula
		// com ponteiro nulo dentro, que faria a checagem de "tem canal?" mentir.
		var saida conversa.Saida
		if canal != nil {
			saida = disparo.NovoTextoLivre(canal)
		}
		s.conversa = conversa.NovoService(conversa.NovoRepo(gdb), saida, log)

		s.disparo = disparo.NovoService(disparo.NovoRepo(gdb), repoCobranca, s.cobranca, canal, s.conversa, log)
		s.Worker = disparo.NovoWorker(s.disparo, log)

		s.webhook = conversa.NovoWebhook(s.conversa, cfg.WhatsAppAppSecret, cfg.WhatsAppVerifyToken)
		if !s.webhook.Configurado() {
			// Sem segredo de assinatura a rota não é montada: um webhook
			// público sem conferir assinatura aceitaria "mensagem de cliente"
			// forjada por qualquer um.
			log.Warn("webhook do WhatsApp desligado — defina WHATSAPP_APP_SECRET e WHATSAPP_VERIFY_TOKEN para receber resposta de cliente")
		}
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
	// O limite global protege as rotas de uso humano. O webhook da Meta é
	// exceção: ela agrupa eventos e pode passar de 100 por minuto num pico, e
	// evento descartado é resposta de cliente perdida. A rota tem limite
	// próprio, bem mais folgado, dentro do grupo dela.
	s.router.Use(exceto(caminhoDoWebhook, httprate.LimitBy(100, time.Minute, chaveDeCliente)))
	s.router.Use(s.origemPermitida)
	s.router.Use(headersSeguros(producao))

	s.rotas()

	return s
}

// ServeHTTP faz do Servidor um http.Handler — o router é detalhe interno.
func (s *Servidor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// chaveDeCliente é a chave usada pelo rate limit — o IP resolvido por
// middleware.ClientIPFromXFFTrustedProxies acima, canonicalizado (IPv6 vira
// prefixo /64, pra um cliente não escapar do limite trocando de endereço
// dentro do próprio /64).
func chaveDeCliente(r *http.Request) (string, error) {
	return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
}

// caminhoDoWebhook é o caminho completo da rota que a Meta chama.
const caminhoDoWebhook = "/api/v1" + conversa.CaminhoWebhook

// exceto aplica um middleware em tudo, menos no caminho dado. Existe porque o
// chi aplica middleware de raiz em toda a árvore, e o webhook precisa ficar
// fora do limite de taxa pensado para navegador.
func exceto(caminho string, mw func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		comMiddleware := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == caminho {
				next.ServeHTTP(w, r)
				return
			}
			comMiddleware.ServeHTTP(w, r)
		})
	}
}
