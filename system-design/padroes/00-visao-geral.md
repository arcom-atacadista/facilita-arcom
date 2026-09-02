# 00 — Visão geral

Como o projeto funciona por cima, pra você e pro chat terem o mesmo modelo mental.

## As peças

- **frontend** — a tela (React + Vite). Em produção, o **nginx** serve o build
  e faz **proxy de `/api`** pro backend (config já pronta no
  `nginx.conf.template`). Em dev (`pnpm dev`), quem faz esse mesmo proxy é o
  próprio Vite (`vite.config.ts`).
- **backend** — o servidor em **Go**, onde fica a lógica.
- **postgres** — banco de dados (só se o projeto precisar).
- **redis** — cache e filas (só se o projeto precisar).

## Como sobe

Tudo roda com **um comando**:

```bash
docker compose up --build
```

Depois é só abrir **http://localhost:8080** no navegador.

## Como eles conversam (sem Traefik, sem CORS)

- O navegador acessa só o **frontend** (porta 8080).
- Quando o front chama `/api/...`, o **nginx** repassa (proxy) pro backend.
- Por isso funciona numa **origem só**: sem CORS, sem URL de servidor na mão.

```
navegador ─▶ frontend (nginx :8080 → publicado em :8080)
                 │  /api/*  ──proxy──▶ backend (:3000) ─┬─▶ postgres
                 └  resto   = os arquivos estáticos      └─▶ redis
```

## O que você mexe vs. o que não mexe

| Você mexe                          | Não mexer (base pronta)                              |
|------------------------------------|-------------------------------------------------------|
| código em `frontend/` e `backend/` | `docker-compose.yml`                                   |
| suas dependências (do allowlist)   | `Dockerfile`, `nginx.conf.template`, `.npmrc`, `vite.config.ts`* |

*No `vite.config.ts` você só edita o bloco `manifest` do PWA (nome/ícone). O
proxy e o resto ficam como estão.

O backend é sempre em Go. O setup inicial do repositório é
montado **com a ajuda da equipe de infra** na criação do projeto.
