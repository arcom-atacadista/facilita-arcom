# Versão do Padrão ARCOM

## v3.4 — atual

`docker-compose.prod.yml`: o `ports:` do serviço `frontend` ganha a tag
`!override`.

- **Motivo:** achado operando o padrão num projeto legado de verdade
  (`relatorio-unimed`) — o frontend nunca subia em produção com
  `docker compose -f docker-compose.yml -f docker-compose.prod.yml up`,
  sempre com `failed to bind host port 127.0.0.1:8080/tcp: address already
  in use`, mesmo sem nenhum processo externo na porta. Causa: o Compose
  **funde** a lista de `ports` do override com a do base em vez de
  substituir — o `"8080:8080"` (0.0.0.0, todas as interfaces) do
  `docker-compose.yml` continuava valendo, e a entrada `127.0.0.1:${PORT_FRONTEND}`
  do override tentava publicar a mesma porta 8080 de novo por cima. A
  primeira (0.0.0.0) sempre ganhava a corrida e a segunda sempre falhava —
  bug determinístico, presente em todo projeto nascido do template que use
  o override de produção, não peculiaridade de um projeto só.
- **Fix:** `ports: !override` no serviço `frontend` de `docker-compose.prod.yml`
  força o Compose a substituir a lista em vez de fundir (suportado desde
  Compose 2.24). `docker-compose.yml` base não muda.
- **`04-docker-deploy.md`:** novo item explicando a regra, pra quem escrever
  um override do zero (ou adicionar outro serviço com `ports:` em produção)
  saber que precisa da tag.

## v3.3

Headers de segurança do frontend saem do `nginx.conf.template` e vão pra um
arquivo próprio, `frontend/nginx-headers-seguranca.conf`.

- **Motivo:** achado operando o padrão num projeto legado de verdade
  (`relatorio-unimed`) — o `nginx.conf.template` da v3.2 põe os `add_header`
  direto no bloco `server`, mas isso é frágil: uma regra traiçoeira do nginx
  faz um `add_header` (ou `expires`, que usa o mesmo mecanismo) dentro de um
  `location` **descartar** todos os `add_header` herdados do `server`.
  Bastaria um `location` novo com regra de cache pra os headers de segurança
  sumirem daquele caminho, silenciosamente, sem erro nenhum.
- **`nginx-headers-seguranca.conf`:** novo arquivo com os mesmos headers de
  antes (CSP, `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`,
  `Permissions-Policy`), incluído explicitamente (`include`) em cada
  `location` de conteúdo estático do `nginx.conf.template` — nunca em
  `/api/`, onde quem manda os headers é o próprio backend. Guard
  `proteger-arquivo.js` atualizado pra proteger esse arquivo do mesmo jeito
  que já protegia o `nginx.conf.template`.
- **`frontend/Dockerfile`:** novo `COPY` do arquivo pra
  `/etc/nginx/headers-seguranca.conf` no stage do nginx.

## v3.2

Troca do servidor de produção do frontend: `vite preview` → **nginx**.

- **Motivo:** `vite preview` é servidor de dev, não pensado pra produção —
  o padrão já precisava endurecê-lo na mão (headers de segurança, allowlist
  de Host contra DNS rebinding, tudo no `vite.config.ts`). nginx é servidor
  HTTP de verdade: mais rápido pra estático (sendfile/gzip nativos), maduro,
  e a peça que a própria `04-docker-deploy.md` já recomendava pro reverse
  proxy externo — mesma ferramenta dentro e fora do compose.
- **`frontend/Dockerfile` vira multi-stage:** stage 1 continua `node:22-alpine`
  (`pnpm install` + `pnpm run build`); stage 2 é
  `nginxinc/nginx-unprivileged:1.27-alpine` (non-root de fábrica, uid 101,
  escuta em 8080). Novo allowlist de imagem em `01-stack-permitida.md`.
- **Novo arquivo `frontend/nginx.conf.template`:** serve os estáticos (com
  cache longo pros assets com hash e `no-cache` no `index.html`), faz o proxy
  de `/api` pro backend (envsubst da variável `API_TARGET` em runtime — mesmo
  nome de env que o `vite.config.ts` já usava) e replica os headers de
  segurança que antes viviam em `preview.headers`. Guard `proteger-arquivo.js`
  atualizado pra proteger esse arquivo do mesmo jeito que já protegia o
  `Dockerfile`.
- **`vite.config.ts` mais simples:** o bloco `preview` (headers,
  `allowedHosts`) foi removido — só existia pra compensar o `vite preview`
  em produção. `pnpm dev` continua igual (proxy de `/api` pro backend local).
- **`ALLOWED_HOSTS` removida:** a proteção contra DNS rebinding do
  `vite preview` não tem equivalente necessário no nginx — quem já decide
  qual domínio é aceito é o reverse proxy externo (Caddy/Nginx/LB da infra)
  descrito em `04-docker-deploy.md`. Variável removida do `.env.example`,
  `docker-compose.prod.yml` e da skill `deploy-producao`.
- **Portas:** frontend passa a escutar em 8080 dentro do container (era 4173).
  `docker-compose.yml`/`.prod.yml` atualizados.

## v3.1

Ajustes vindos de operar o padrão num projeto legado de verdade
(`relatorio-unimed`) e de um incidente real de produção — cada item fecha uma
lacuna encontrada na prática, não só na teoria.

- **Redis vira opt-in de verdade**: `docker-compose.yml`/`.prod.yml` não vêm
  mais com o serviço por padrão (só o comentário de como adicionar). Nova
  seção em `05-regras-para-a-ia.md` com checklist de auto-checagem — a IA se
  pergunta sozinha se o pedido justifica Redis antes de adicionar, não espera
  ser pedida. Quando adicionado, vem **sem persistência em disco** por padrão
  (`--save ""`, `--appendonly no`) — evita a causa mais comum de incidente de
  Redis em produção (disco cheio + `stop-writes-on-bgsave-error` travando
  toda escrita). `scripts/setup-storage-prod.sh` trocou `SKIP_REDIS` (padrão
  "cria") por `COM_REDIS` (padrão "não cria").
- **E-mail/SMTP vira padrão oficial**: `github.com/wneessen/go-mail` liberado
  em `01-stack-permitida.md`, mesma lógica opt-in do Redis. Sincronizado no
  guard `verificar-dependencia.js`.
- **Migrations toleram a janela de boot do Postgres**: novo
  `db.EsperarPronto` (retry com backoff) antes de `goose.Up` e antes do
  backend considerar a conexão pronta — corrige um incidente real (a imagem
  oficial do Postgres sobe/desliga/sobe de novo na primeira inicialização;
  sem retry, `cmd/migrate` caía numa janela de menos de 1s).
- **Toolchain Go pinada**: `go.mod` do esqueleto ganhou `toolchain go1.26.6`
  (antes só `go 1.25.0`, sem diretiva de toolchain) — fecha vulnerabilidades
  alcançáveis de stdlib que só apareciam por causa do patch da toolchain, não
  da versão de linguagem.
- **Guards corrigidos**: `11-guards.md` referenciava os hooks como `.sh`
  (são `.js` de verdade) e listava 5 checagens que `verificar-codigo.js` não
  implementa — texto agora reflete só o que roda de verdade.

## v3.0

Arquitetura de aplicação (não só infra) + imposição automática. Motivado por
uma revisão completa do padrão: o v2.2 definia bem o que podia/não podia usar,
mas era fino em como o frontend e o backend deveriam ser montados por dentro
— e não tinha nada que impedisse desvio automaticamente. Cada item abaixo
fecha uma lacuna real encontrada na revisão.

- **Contrato de API novo** (`09-contrato-api.md`): formato único de
  sucesso/erro (`application/problem+json` com `codigo` estável), prefixo
  `/api/v1` pras rotas de negócio (liveness/readiness ficam fora de versão em
  `/api/health` e `/api/ready`).
- **Frontend — camada `core/`**: `api.ts` (axios + interceptor + contrato de
  erro), `erro.ts`, `query.ts` (TanStack Query), `toast.ts` (sonner),
  `carregando.tsx` (loading global em barra, nunca overlay), `sessao.tsx`
  (guarda de rota), `ErroDeTela.tsx` (ErrorBoundary). Novas libs no allowlist:
  `@tanstack/react-query`, `zod`, `@hookform/resolvers`, `sonner`,
  `react-error-boundary`, `zustand`. ESLint endurecida (`import/no-restricted-paths`,
  `react/no-danger`, bloqueio de `localStorage`/`sessionStorage`).
- **Sessão vira cookie `HttpOnly`**: chega de token no frontend — o backend
  define o cookie no login, o front nunca toca nele. Guard de CSRF por
  checagem de `Origin` no backend (`03-backend.md`, skill `autenticacao`).
- **Backend — arquitetura real**: `http.Server` com timeouts + graceful
  shutdown (era `http.ListenAndServe` cru), stack de middleware completa
  (`RequestID`, IP do cliente, log estruturado, `Recoverer`, `Compress`,
  `Timeout`, rate limit, CSRF, headers), contrato de erro único
  (`internal/servidor/problema.go`), config com fail-fast, `/api/ready`
  checando Postgres/Redis, migrations `goose` embarcadas rodando via serviço
  `migrate` separado (nunca no boot), pacote por feature (proibido `pkg/`,
  `utils/`, `common/`, `helpers/` e interface com implementação única).
  `.golangci.yml` + `govulncheck` + `go test -race` no fluxo de qualidade.
- **Imagens base atualizadas**: `node:22-alpine` (pnpm 11+ exige Node ≥22.13)
  e `golang:1.25-alpine`.
- **Guards automáticos**: hooks do Claude Code (`.claude/settings.json`) que
  bloqueiam na hora edição de arquivo de infra, comando fora do padrão
  (`npm`, `curl | sh`, `--no-verify`), segredo em `VITE_*`, padrão de código
  inseguro (SQL concatenado, `localStorage`, `dangerouslySetInnerHTML`,
  mass assignment) e dependência fora do allowlist. Varredura completa sob
  demanda em `scripts/verificar.sh`. Detalhe em `11-guards.md`.
- **Segurança reorganizada**: `10-seguranca.md` (classificação
  INTERNA/EXTERNA/MISTA, severidade, LGPD), skill `autenticacao` (fluxo
  completo de login/sessão/ownership) e skill `revisar-seguranca` (auditoria
  do código já escrito, com calibração de severidade CVSS/CWE/OWASP e
  anti-falso-positivo).
- **Correções de referência**: caminho do design system
  (`design-system/...` → `design/...`), logos localizados na pasta real,
  `.DS_Store` fora do git, allowlist documentando `clsx`/`class-variance-authority`.

## v2.2
Endurecimento vindo do primeiro projeto real que rodou o padrão de ponta a ponta
(CDA — Consulta Dados Arcom). Cada item abaixo corrige um buraco que quebrou ou
expôs o projeto de verdade:
- **Build confiável:** `packageManager` (pnpm) pinado no `package.json` e
  `corepack prepare pnpm@<versão> --activate` no `Dockerfile`. Sem isso o
  corepack pegava a pnpm "latest" (Node 22+) e quebrava no `node:20-alpine`.
  Regra em `01-stack-permitida.md`.
- **Segurança de imagem:** `.dockerignore` obrigatório no `frontend/` e
  `backend/` (evita `.env`/segredo e `node_modules` entrarem na imagem) e
  containers rodando como usuário **non-root**. Regras em `04-docker-deploy.md`.
- **Produção:** nova seção em `04-docker-deploy.md` — override
  `docker-compose.prod.yml` (secrets sem fallback, `restart: always`, log
  rotacionado, limites de recurso), TLS via reverse proxy externo, e
  persistência dos dados em disco/array real via volume nomeado com
  `driver_opts` (bind). Teto de disco 100% via `scripts/setup-storage-prod.sh`
  (filesystem de tamanho fixo pra Postgres e Redis) + `--maxmemory` do Redis.
- **Base que já sobe:** o template deixou de ser só config e passou a trazer um
  hello-world funcional (frontend on-brand + backend `GET /api/health`), com
  `package.json`/`pnpm-lock.yaml`/`go.mod`/`go.sum` prontos. `docker compose up`
  fica verde de cara — validado ponta a ponta.
- **Sensible defaults:** Prettier + ESLint (config recomendada, leve) no front —
  qualidade sem a cerimônia dos guias de FAANG.
- **Skills do Claude Code** em `.claude/skills/`: `nova-tela`, `novo-endpoint`,
  `deploy-producao` — os fluxos ARCOM viram atalhos que todo projeto herda.

## v2.1
- Adicionado `CLAUDE.md` (raiz) + `.gitignore`: o mesmo padrão passa a funcionar
  no **Claude Code** (que lê o CLAUDE.md sozinho), além do Claude Project.
  Fonte única = `padroes/`; dois adaptadores (Instruções do Project + CLAUDE.md).

## v2.0
- **BREAKING:** backend agora é Go (motor único). Saíram NestJS e FastAPI.
  Novo allowlist de módulos Go (chi, gorm/pgx, go-redis, asynq, jwt, bcrypt,
  validator, goose, cron, go-mail, uuid). `backend/Dockerfile` multi-stage
  estático (golang:1.23-alpine -> alpine:3.20). Sem `.npmrc` no backend.
- pnpm/`.npmrc` passam a ser regra só do frontend.

## v1.2
- Integrado o Design System ARCOM: guia `08-design-system.md`, tokens da marca
  no `tailwind.config.js` (paleta, Red Hat Display, raios, sombras) e o
  arquivo completo em `design-system/ARCOM-Design-System.md`.
- Regra correspondente no `05-regras-para-a-ia.md`.

## v1.1
- Novo guia `07-frontend-primeiro.md`: começar só com frontend; backend +
  Postgres só quando precisar de persistência.
- Regra correspondente no `05-regras-para-a-ia.md`.

## v1.0
- Stack: Vite + React (Tailwind/shadcn/Radix/lucide), backend NestJS ou FastAPI,
  Postgres/Redis opcionais.
- pnpm obrigatório + `.npmrc` de segurança (`ignore-scripts=true`).
- Sobe com `docker compose up --build` (front serve via Vite + proxy de `/api`,
  sem Traefik/nginx).

---

## Como atualizar (quando sair uma versão nova)
1. Baixe o novo conjunto de `.md`.
2. No seu Projeto do Claude, vá em Conhecimento e **substitua** os arquivos
   antigos pelos novos.
3. Se o `INSTRUCOES-DO-PROJETO.md` mudou, recole o conteúdo no campo Instruções.
4. Confira a versão no topo do `LEIA-PRIMEIRO.md`.
