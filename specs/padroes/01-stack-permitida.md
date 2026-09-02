# 01 — Stack permitida (o que PODE e NÃO PODE usar)

Esta é a fonte da verdade. Tudo fora daqui é **rejeitado no deploy**. Não invente
dependência, imagem ou serviço — o build vai falhar.

---

## Gerenciadores de pacote

- **Frontend:** só `pnpm`. `npm` e `yarn` são proibidos; no repo só pode existir
  `pnpm-lock.yaml` (nada de `package-lock.json` ou `yarn.lock`).
- **Versão do pnpm pinada (obrigatório).** O `package.json` do frontend tem
  que ter `"packageManager": "pnpm@<versão>"`, e a versão precisa casar com o
  `lockfileVersion` do `pnpm-lock.yaml` e com o `corepack prepare pnpm@<versão>`
  do `Dockerfile`. Sem isso o corepack baixa a "latest", que pode exigir um
  Node mais novo que a imagem base e quebrar o build
  (`ERR_UNKNOWN_BUILTIN_MODULE: node:sqlite`).
  - O template vem pinado em **pnpm 11.x**, que traz `minimumReleaseAge` por
    padrão (não instala pacote publicado há menos de tempo — mitiga ataque de
    supply chain via versão maliciosa recém-publicada). pnpm 11 exige
    **Node ≥ 22.13** — por isso a imagem de build é `node:22-alpine` (não
    `node:20-alpine`).
  - `pnpm-workspace.yaml` é permitido **só** com o campo `allowBuilds`
    (decisão explícita de quais dependências rodam script de build, ver
    abaixo) — não vale virar monorepo sem o projeto ser de fato um monorepo;
    um workspace com `packages:` vazio/inválido quebra o build com
    `packages field missing or empty`.
- **Backend (Go):** `go mod`. Commite `go.mod` e `go.sum`. O Go verifica a
  integridade dos módulos pelo `go.sum` + checksum DB e **não roda scripts de
  instalação** — aquele vetor de ataque do npm simplesmente não existe aqui.

### `.npmrc` do frontend (obrigatório, não alterar)
- O `.npmrc` abaixo é **obrigatório** e não pode ser alterado — ele desliga a
  execução de scripts de instalação (maior vetor de ataque de cadeia de
  suprimentos) e trava versões:

```properties
ignore-scripts=true
strict-ssl=true
save-exact=true
engine-strict=true
audit=true
fund=false
update-notifier=false
```

- Como `ignore-scripts=true`, **nenhuma dependência de frontend que compile
  binário nativo na instalação funciona** — o allowlist do front já evita isso.
  Dependência que baixa binário pré-compilado via `optionalDependencies`
  (como o `esbuild` do próprio Vite) continua funcionando normalmente — só o
  script de fallback dela fica sem rodar, e por isso está listada
  explicitamente em `pnpm-workspace.yaml` (`allowBuilds: { esbuild: false }`)
  em vez de ficar como decisão pendente a cada instalação.

---

## Frontend

### Libs default (já vêm no template — é a stack shadcn/ui)

- `react` + `react-dom` — base
- `react-router-dom` — navegação entre telas
- `tailwindcss` + `postcss` + `autoprefixer` — estilo (utility-first)
- `tailwindcss-animate` — animações
- `tailwind-merge` + `clsx` — juntar classes (helper `cn()` em `core/cn.ts`)
- `class-variance-authority` — variantes de componente (padrão shadcn)
- `@radix-ui/*` — componentes acessíveis (base do shadcn)
- `lucide-react` — ícones
- `react-hook-form` + `@hookform/resolvers` + `zod` — formulários e validação
  tipada (schema único, compartilhado entre validação de form e tipo de dado)
- `@tanstack/react-query` — cache/estado de servidor (nunca `useEffect` +
  `useState` manual pra dado de API — ver `02-frontend.md`)
- `zustand` — estado global de UI que **não** é dado de servidor (ex.: sessão
  em memória, contador manual de loading)
- `sonner` — notificações/toast (é o toast oficial do shadcn hoje)
- `react-error-boundary` — boundary de erro de tela
- `axios` — chamadas HTTP (`baseURL: "/api/v1"`, ver `core/api.ts`)
- `dayjs` — datas e fuso (plugins `utc` e `timezone`)
- `vite-plugin-pwa` — transformar em PWA

> Tailwind/PostCSS/PWA rodam no `pnpm run build`, não em script de instalação —
> por isso funcionam com `ignore-scripts=true`.

### Opcional (liberado, fora do default)

- `react-icons` — só se faltar algum ícone no `lucide-react`.
- `dompurify` — só se o projeto precisar mesmo renderizar HTML de terceiro
  (editor de texto rico etc.); sempre atrás de um componente único de
  sanitização, nunca `dangerouslySetInnerHTML` solto (bloqueado por ESLint).

### Proibido no frontend

- `npm` ou `yarn` (só pnpm)
- Outro framework: Next, Angular, Vue, SvelteKit, Remix
- Outra lib de UI/estilo: **MUI**, Chakra, Ant Design, styled-components, emotion
- Outra lib de data-fetching/estado global concorrente (SWR, Redux, Recoil,
  Jotai, Redux Toolkit) — o padrão é `@tanstack/react-query` + `zustand`
- `moment` / `moment-timezone` (use `dayjs`)
- `fetch`/`axios` chamando domínio externo direto do navegador — passe pelo backend
- Guardar token/sessão/permissão em `localStorage`/`sessionStorage` (a sessão é
  cookie `HttpOnly` — ver `02-frontend.md`)
- Qualquer segredo no bundle (só variáveis `VITE_*` públicas)

---

## Backend — Go (motor único)

O backend é **sempre em Go**. Não há mais NestJS nem FastAPI.

Requisitos:
- Rotas de negócio sob `/api/v1`; `GET /api/health` e `GET /api/ready`
  (liveness/readiness) ficam fora da versão
- Escutar em `0.0.0.0:3000` (porta vem de `PORT`, default 3000), com
  `http.Server` de timeouts + graceful shutdown (ver `03-backend.md`)
- Ler `DATABASE_URL` e `REDIS_URL` do ambiente (nunca hardcode)
- Build com `CGO_ENABLED=0` (binário estático)

### Libs liberadas (módulos Go)

**HTTP / core**
`github.com/go-chi/chi/v5` (router + middleware) ·
`github.com/go-chi/httprate` (rate limit) ·
`github.com/go-chi/httplog/v3` (log de requisição sobre `slog`) ·
stdlib `net/http`, `encoding/json`, `log/slog`

**Config**
`github.com/joho/godotenv` (carrega `.env` em dev) · stdlib `os`

**Banco (Postgres)**
`gorm.io/gorm` + `gorm.io/driver/postgres` (ORM) · `github.com/jackc/pgx/v5`
(driver/pool, se quiser SQL cru) · `github.com/pressly/goose/v3` (migrations)

**Redis / cache / filas**
`github.com/redis/go-redis/v9` (cliente) · `github.com/hibiken/asynq`
(filas em Redis — o equivalente ao BullMQ)

Redis é opt-in — o `docker-compose.yml` do template **não vem com o serviço
redis por padrão**, só com o comentário de como adicionar. Só acrescente
quando aparecer uma necessidade concreta — cache, fila de jobs, rate limit
distribuído entre réplicas, sessão compartilhada. Projeto com banco mas sem
nenhuma dessas necessidades fica só com Postgres. Isso evita Redis parado
consumindo memória à toa em projeto que não usa — soma rápido quando várias
instâncias do template rodam na mesma máquina.

**Sem persistência em disco por padrão** (`--save ""`, `--appendonly no`).
Cache, rate limit e lock não precisam sobreviver a um restart — e ligar
RDB/AOF sem necessidade é a causa mais comum de incidente de Redis em
produção: o disco do volume enche, a proteção `stop-writes-on-bgsave-error`
trava **toda** escrita (não só a que não coube), e o sintoma na tela é
confuso (leitura segue funcionando, só escrita trava). Sem persistência,
Redis não usa disco nenhum — não precisa de volume, não precisa de teto de
disco (`scripts/setup-storage-prod.sh`), esse jeito de falhar nem existe. Só
ligue persistência se o caso de uso **exigir** Redis como armazenamento
durável — raro, e antes de fazer isso questione se não é o Postgres quem
deveria guardar esse dado (ver `05-regras-para-a-ia.md`).

**E-mail / SMTP**
`github.com/wneessen/go-mail` (envio via SMTP — Go puro, sem dependência de
sistema, suporta STARTTLS/SSL e anexo)

Opt-in, mesma lógica do Redis: só entra se o projeto realmente manda e-mail
(confirmação de conta, magic link, notificação, relatório anexado). Em dev,
`SMTP_HOST` vazio é aceitável — a feature continua funcionando (gera o
link/token normalmente), só não manda o e-mail de verdade; é assim que dá pra
desenvolver sem depender de um SMTP real no ar. Em produção, as credenciais
(`SMTP_HOST`, `SMTP_PORTA`, `SMTP_USUARIO`, `SMTP_SENHA`, `SMTP_REMETENTE`) vêm
do `.env.prod`, sem fallback (`${SMTP_HOST:?defina no .env.prod}` no
`docker-compose.prod.yml` — ver `04-docker-deploy.md`), nunca hardcoded nem
commitadas.

**Auth / cripto**
`golang.org/x/crypto/bcrypt` (hash de senha — Go puro, sem build nativo) ·
`github.com/golang-jwt/jwt/v5` (JWT, se usado como valor do cookie de sessão)

**Validação**
`github.com/go-playground/validator/v10`

**Agendamento**
`github.com/robfig/cron/v3`

**Fila externa (RabbitMQ)**
`github.com/rabbitmq/amqp091-go` — publica/consome mensagem numa fila
(broker centralizado da infra, não roda no `docker-compose.yml` do
projeto). Conexão via `RABBITMQ_URL`.

> **Antes de usar, fale com o time de infra** — quem configura o acesso, dá a
> credencial (`RABBITMQ_URL`) e diz quais filas já existem disponíveis pro
> projeto. Nunca invente nome de fila/exchange nem assuma que uma já existe.
>
> A fila é via de mão única pra ação assíncrona (publica um evento/comando;
> outro serviço consome) — **não é** uma ponte genérica pra ler dado de
> sistema interno da empresa (ERP, CRM, planilha, banco legado). Pra isso,
> ver `05-regras-para-a-ia.md`, seção "Integração com sistema da empresa": a
> regra é sempre perguntar se existe API antes de desenhar qualquer
> integração.

**Utilitários**
`github.com/google/uuid` · `github.com/SherClockHolmes/webpush-go` (web push) ·
stdlib `net/http` para chamadas externas

**Qualidade/segurança (dev, não vai pro binário)**
`golangci-lint` (config em `.golangci.yml`, inclui `gosec`) ·
`golang.org/x/vuln/cmd/govulncheck`

**PDF (geração)**
`github.com/go-pdf/fpdf` — gera o demonstrativo individual no layout ARCOM.
Leitura do PDF original é via `pdftotext` (poppler-utils, pacote de sistema no
Dockerfile do backend — ver 04-docker-deploy.md), não por lib Go.

### Proibido no backend

- Outro motor/linguagem (Node, Python, Java) — o backend é Go
- Outro banco além de Postgres/Redis
- Abrir porta pro host ou escutar em porta diferente de 3000
- `CGO_ENABLED=1` sem necessidade (perde o binário estático e complica o build)
- `http.ListenAndServe(...)` direto sem `http.Server` com timeouts/shutdown
  (ver `03-backend.md`)
- Pacote `pkg/`, `utils/`, `common/`, `helpers/` (nome sem semântica) ou
  interface com uma única implementação (ver `03-backend.md`)
- Usar a fila do RabbitMQ (nome de fila/exchange, credencial) sem antes
  confirmar com o time de infra — nunca invente/assuma

---

## Dados / infra

**Pode:** PostgreSQL (via `DATABASE_URL`), Redis (via `REDIS_URL`).
**Não pode:** outro banco (Mongo, MySQL, SQLite em arquivo), alterar os serviços
de infra do `docker-compose.yml`, imagens fora do allowlist
(`node:22-alpine` — build do front, `nginxinc/nginx-unprivileged:1.27-alpine` —
runtime do front, `golang:1.25-alpine` + `alpine:3.20` — backend,
`postgres:16-alpine`, `redis:7-alpine`).
