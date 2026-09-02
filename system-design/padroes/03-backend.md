# 03 — Backend (Go)

O backend é sempre em **Go**. Libs permitidas em `01-stack-permitida.md`.
Leia `09-contrato-api.md` antes deste — o formato de erro/dados vem de lá.

## Regras

- Rotas de negócio sob `/api/v1`; `GET /api/health` (liveness) e
  `GET /api/ready` (readiness) ficam **fora** da versão — a infra depende
  deles.
- Escutar em `0.0.0.0:3000` (porta vem de `PORT`, default 3000), com
  `http.Server` configurado com timeouts (`ReadHeaderTimeout`, `ReadTimeout`,
  `WriteTimeout`, `IdleTimeout`) e **graceful shutdown** — nunca
  `http.ListenAndServe(...)` cru (sem timeout = exposto a slowloris; sem
  shutdown = deploy corta requisição no meio).
- Ler `DATABASE_URL`/`REDIS_URL`/segredos do ambiente, nunca hardcode.
  `internal/config.Load()` falha rápido (retorna erro, o processo sai) se
  algo obrigatório estiver mal configurado — nunca sobe quebrado em silêncio.
- Validar tudo que vem do cliente (`validator`). Nunca confiar no frontend.
- Todo erro sai no formato de `09-contrato-api.md` — nunca
  `err.Error()` cru no corpo de uma resposta 5xx.

## Estrutura

Pacote por feature — as camadas (handler/service/repo) são **arquivos dentro
do pacote**, não pastas paralelas globais. Evita abrir três pastas
(`handlers/`, `services/`, `repositories/`) pra mudar uma coisa só.

```
backend/
├── Dockerfile
├── .golangci.yml
├── go.mod
├── go.sum
├── cmd/
│   ├── server/main.go        # entrada: monta deps, sobe http.Server com shutdown
│   └── migrate/main.go       # aplica migrations (goose) — roda separado, nunca no boot
└── internal/
    ├── config/
    │   └── config.go         # lê env com fail-fast (ver abaixo)
    ├── servidor/              # router, middlewares, contrato de erro
    │   ├── servidor.go        # Novo(cfg, log, db, redis) http.Handler
    │   ├── rotas.go           # TODAS as rotas num arquivo
    │   ├── middleware.go      # headers, CSRF, limite de corpo
    │   ├── problema.go        # erro -> HTTP (único ponto de conversão)
    │   └── saude.go           # /api/health e /api/ready
    ├── db/
    │   ├── db.go              # conexão Postgres (gorm) e Redis (go-redis)
    │   ├── migrations.go      # //go:embed migrations/*.sql
    │   └── migrations/*.sql   # goose
    └── <recurso>/             # um pacote por feature
        ├── handler.go         # HTTP: decodifica, valida, chama service, responde
        ├── service.go         # regra de negócio
        ├── repo.go             # acesso a dados (gorm/pgx)
        ├── model.go            # entidade + DTOs
        └── <recurso>_test.go
```

**Proibido:** `pkg/`, `utils/`, `common/`, `helpers/` (nome sem semântica —
todo pacote diz o que faz) e **interface com uma única implementação**
(interface só quando existem 2+ implementações reais ou o teste exige fake —
Clean Architecture "de livro" em Go pra um CRUD é 15 camadas de abstração pra
nada).

## Ponto de entrada — `http.Server` com timeouts e graceful shutdown

```go
// cmd/server/main.go (resumo — ver o arquivo completo no template)
srv := &http.Server{
    Addr:              "0.0.0.0:" + cfg.Port,
    Handler:           servidor.Novo(cfg, log, gdb, rdb),
    ReadHeaderTimeout: 5 * time.Second, // anti-slowloris
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      15 * time.Second,
    IdleTimeout:       60 * time.Second,
}

ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer parar()
go srv.ListenAndServe()
<-ctx.Done()

desligarCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
srv.Shutdown(desligarCtx) // drena requisição em andamento antes de sair
```

## Rotas e middlewares — `internal/servidor`

Stack de middleware (ordem importa): `RequestID` → IP do cliente
(`middleware.ClientIPFromXFFTrustedProxies` — **não** `RealIP`/`LimitByRealIP`,
depreciados por serem spoofáveis) → log estruturado (`httplog` sobre `slog`)
→ `Recoverer` → `Compress` → `Timeout` → limite de tamanho de corpo →
rate limit (`httprate`) → guard de CSRF (`origemPermitida`) → headers de
segurança.

```go
// internal/servidor/rotas.go
func (s *Servidor) rotas() {
    s.router.Route("/api", func(api chi.Router) {
        api.Get("/health", s.saude)      // liveness — 200 fixo, nunca toca I/O
        api.Get("/ready", s.H(s.pronto)) // readiness — 503 se Postgres/Redis falharem

        api.Route("/v1", func(v1 chi.Router) {
            // v1.Route("/produtos", produto.Rotas) — uma feature por linha
        })
    })
    s.router.NotFound(...)         // também sai no formato do contrato
    s.router.MethodNotAllowed(...)
}
```

**Rate limit:** o backend nunca é exposto direto à internet — só o frontend
publica porta (ver `04-docker-deploy.md`). O proxy do frontend escreve
`X-Forwarded-For` — nginx em produção (`nginx.conf.template`), Vite em dev
(`xfwd: true` no `vite.config.ts`); o backend confia nesse único salto
(`ClientIPFromXFFTrustedProxies(1)`). Se um dia o backend for exposto direto,
isso vira spoofável — reavalie.

**CSRF:** a sessão é cookie (ver abaixo), então todo `POST/PUT/PATCH/DELETE`
passa por um guard que exige o header `Origin` (quando presente) bater com o
host da requisição — barra requisição forjada de outro site.

## Contrato de erro — `problema.go`

Handler devolve erro; um único ponto converte pra HTTP. Nunca escreva JSON de
erro na mão num handler.

```go
type apiHandler func(w http.ResponseWriter, r *http.Request) error

func (s *Servidor) H(fn apiHandler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if err := fn(w, r); err != nil {
            s.tratarErro(w, r, err) // ErroDominio -> Problema; resto -> 500 genérico + log
        }
    }
}

// num handler de feature:
func buscar(w http.ResponseWriter, r *http.Request) error {
    produto, err := service.Buscar(r.Context(), id)
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return servidor.NaoEncontrado("Produto não encontrado.")
    }
    if err != nil {
        return err // vira 500 genérico + log completo no servidor
    }
    return json.NewEncoder(w).Encode(produto)
}
```

Erro 5xx nunca devolve `err.Error()` no corpo (vaza nome de tabela, driver,
caminho) — só o genérico do contrato; o erro real vai pro `slog`.

## Config — fail-fast

`internal/config.Load()` retorna erro (não sobe com valor vazio) quando algo
obrigatório está ausente ou inválido — ex.: `JWT_SECRET` presente mas curto
demais pra HS256. `DATABASE_URL`/`REDIS_URL` são opcionais (string vazia = o
projeto não usa, ver `07-frontend-primeiro.md`); o que a feature realmente
precisa (chave de API de terceiro) usa `config.Obrigatorio("CHAVE")` e falha
no boot, nunca em request.

## Convenções

- **HTTP:** `chi` como router; ver stack de middleware acima.
- **Config:** `internal/config` lê `os.Getenv` com fail-fast; em dev,
  `godotenv` carrega `.env`.
- **Banco:** `gorm` + driver `postgres` (ou `pgx` puro pra SQL cru). Uma
  struct por tabela. **Nunca** monte query concatenando string com dado do
  usuário (`Raw("...".+id)` = SQLi) — sempre `?`/parâmetro. Coluna/ordenação
  dinâmica vinda do cliente passa por **allowlist**, nunca interpolada direto.
- **Migrations:** `goose`, SQL em `internal/db/migrations/`, embarcado no
  binário (`//go:embed`). Rodam via `cmd/migrate`, como serviço próprio do
  compose — nunca dentro do boot do servidor (evita corrida entre réplicas).
  Sempre com `-- +goose Down` preenchido. Antes de rodar `goose.Up`, espere a
  conexão responder com retry (`db.EsperarPronto`, ver `internal/db/db.go`) —
  não conecte uma vez só e desista. Na primeira subida com volume vazio, a
  imagem oficial do Postgres sobe, roda os scripts de inicialização, desliga
  e sobe de novo pra valer, tudo em menos de 1 segundo; `migrate` conectando
  bem nessa janela pega um erro transitório ("the database system is
  starting up") que se resolveria sozinho no segundo seguinte — incidente
  real que já aconteceu com esse exato padrão. `AbrirPostgres` (usado pelo
  `cmd/server`) usa o mesmo helper, pela mesma razão.
- **Redis:** opt-in — só entra se o projeto precisar de cache, fila (`asynq`)
  ou rate limit distribuído (ver `01-stack-permitida.md`); não é serviço
  padrão só por ter Postgres. Se o backend do rate limit for Redis, trate
  falha de Redis como **fail-closed** (nega, nunca libera geral).
- **Sessão/auth:** token vive num **cookie `HttpOnly; Secure; SameSite=Lax`**
  que o backend define no login — o frontend nunca vê o token (ver
  `02-frontend.md`). Senha: `bcrypt` (cost 12, não o default). Token/código
  aleatório: `crypto/rand` (nunca `math/rand`), comparação com
  `crypto/subtle.ConstantTimeCompare`. Checagem de **posse** (o usuário é dono
  do recurso que pediu) em toda rota que recebe um id — ver
  `.claude/skills/autenticacao`.
- **Validação:** `go-playground/validator` nas structs de entrada (DTOs),
  com `DisallowUnknownFields()` no decoder JSON.
- **Agendamento:** `robfig/cron`.
- **Fila (RabbitMQ):** `amqp091-go`, conexão via `RABBITMQ_URL`, só se o
  projeto realmente precisar de mensageria. **Antes de implementar, confirme
  com o time de infra**: URL/credencial de acesso e quais filas/exchanges já
  existem disponíveis pro projeto — nunca invente nome de fila. Nunca monte
  a conexão com input do cliente (mesma regra de SSRF da skill `seguranca`).
- **E-mail (SMTP):** opt-in — só entra se o projeto precisar mandar e-mail
  (confirmação, magic link, notificação, relatório anexado); ver
  `01-stack-permitida.md`. `github.com/wneessen/go-mail`, credenciais
  (`SMTP_HOST`/`SMTP_PORTA`/`SMTP_USUARIO`/`SMTP_SENHA`/`SMTP_REMETENTE`) só
  do ambiente. Sem `SMTP_HOST` configurado, a feature continua funcionando
  (gera o link/token normalmente) e só deixa de mandar o e-mail — não trava o
  fluxo nem falha a requisição por causa disso; é assim que dá pra
  desenvolver sem SMTP real no ar. Em produção as credenciais vão no
  `docker-compose.prod.yml` sem fallback (ver `04-docker-deploy.md`).
- **Datas:** stdlib `time`, sempre em **UTC** (ver `09-contrato-api.md`) —
  `gorm.Config{NowFunc: func() time.Time { return time.Now().UTC() }}`.
- **IDs:** `google/uuid`.
- **Log:** `log/slog` estruturado, sempre com `request_id`. Tipo sensível
  (senha, token, e-mail, CPF) implementa `LogValue()` devolvendo valor
  mascarado/redigido — nunca logue o dado cru.

## Qualidade e segurança

```bash
go vet ./...
golangci-lint run   # config em .golangci.yml — inclui gosec
govulncheck ./...    # reachability analysis: só reporta vuln que o código alcança
go test -race ./...
```

Piso mínimo de teste: table-driven + `httptest` batendo no `http.Handler` de
`servidor.Novo(...)` (testa router + middleware + handler + validação de uma
vez — ver `internal/servidor/servidor_test.go` no template).

## Integridade / segurança

- Commite `go.sum` — o Go valida os módulos automaticamente (checksum DB).
- Go **não executa scripts na instalação** de dependências, então não existe o
  risco de "postinstall malicioso" do mundo npm.
- Segredos sempre via ambiente, nunca no código nem commitados.
- Ver `.claude/skills/seguranca` e `.claude/skills/autenticacao` antes de
  criar login, formulário ou endpoint que recebe dado do usuário.
