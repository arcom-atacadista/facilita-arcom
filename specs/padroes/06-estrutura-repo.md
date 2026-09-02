# 06 — Estrutura de pastas (repositório no GitHub)

Todo projeto segue este layout. O chat deve criar/editar arquivos respeitando-o.

```
meu-projeto/
├── docker-compose.yml          # sobe tudo com "docker compose up --build"
├── docker-compose.prod.yml     # override de produção (secrets sem fallback etc.)
├── .env.example                # variáveis de exemplo (copiar pra .env)
├── .gitignore
├── README.md                   # como rodar (curto)
├── CLAUDE.md                   # lido sozinho pelo Claude Code
│
├── .claude/
│   ├── settings.json           # hooks dos guards (ver padroes/11-guards.md)
│   └── skills/                 # nova-tela, novo-endpoint, seguranca, autenticacao,
│                                # revisar-seguranca, deploy-producao
│
├── system-design/               # o Padrão ARCOM inteiro — travado, não é código do projeto
│   ├── padroes/                  # fonte da verdade (este guia e os outros 00-12)
│   │   └── gateway-arcom/        # schema do Gateway de dados (api.md, openapi.yaml) — ver 12-gateway-arcom.md
│   ├── design/                   # identidade visual ARCOM (logo, cores, tipografia)
│   ├── privacy/                  # termos de uso e boas práticas de IA
│   ├── scripts/
│   │   ├── verificar.sh          # varredura completa (lint/vuln/allowlist)
│   │   ├── guards/*.js           # os scripts que os hooks do .claude/ chamam
│   │   └── setup-storage-prod.sh # teto de disco em produção
│   ├── VERSAO.md
│   ├── LEIA-PRIMEIRO.md
│   └── INSTRUCOES-DO-PROJETO.md
│
├── frontend/                   # React + Vite
│   ├── .npmrc                  # NÃO alterar
│   ├── Dockerfile              # build (node) + runtime (nginx não-root)
│   ├── nginx.conf.template     # estático + proxy de /api em produção (envsubst)
│   ├── nginx-headers-seguranca.conf  # headers de segurança, incluído por location (evita gotcha do add_header)
│   ├── vite.config.ts          # proxy de /api em dev (não mexer no proxy)
│   ├── tailwind.config.js
│   ├── postcss.config.js
│   ├── eslint.config.js
│   ├── package.json
│   ├── pnpm-lock.yaml          # obrigatório (build usa --frozen-lockfile)
│   ├── pnpm-workspace.yaml     # só o campo allowBuilds
│   ├── index.html
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── rotas.tsx
│       ├── core/{api,erro,query,toast,carregando,estadoUI,sessao,ErroDeTela,cn}.ts(x)
│       ├── components/ui/      # shadcn — gerado
│       ├── components/
│       ├── pages/
│       ├── hooks/
│       └── tipos/
│
└── backend/                    # Go
    ├── Dockerfile
    ├── .golangci.yml
    ├── go.mod
    ├── go.sum                  # obrigatório (integridade dos módulos)
    ├── cmd/
    │   ├── server/main.go      # entrada: http.Server com timeouts + graceful shutdown
    │   └── migrate/main.go     # aplica migrations (goose), roda como serviço separado
    └── internal/
        ├── config/             # lê env com fail-fast
        ├── servidor/           # router, middlewares, contrato de erro, health/ready
        ├── db/                 # conexão Postgres/Redis + migrations embarcadas
        └── <recurso>/          # um pacote por feature (handler/service/repo/model)
```

## Regras da estrutura

- Frontend sempre em `frontend/`, backend sempre em `backend/`.
- Um recurso por pasta: no front em `src/pages/` (tela) ou `src/core/`
  (infraestrutura, ver `02-frontend.md`); no back, um pacote por feature em
  `internal/`.
- Commite os lockfiles: `frontend/pnpm-lock.yaml` e `backend/go.sum` (senão o
  build Docker falha / não valida).
- Nada de código fora de `frontend/` e `backend/` (o `build.context` do
  Docker só olha pra essas duas pastas). `system-design/` e `.claude/` são
  material da plataforma, não do app — e `system-design/` é travado (guard
  `proteger-arquivo.js` recusa qualquer edição ali, ver `11-guards.md`).
- Não versionar `node_modules/`, `dist/`, o binário compilado, `.env`,
  `.DS_Store` nem artefato gerado pelo `tsc` (`vite.config.js`,
  `vite.config.d.ts`, `*.tsbuildinfo`) — tudo já coberto pelo `.gitignore`.
