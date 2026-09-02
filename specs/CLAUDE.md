# Projeto ARCOM — instruções para o Claude Code

Este repositório segue o **Padrão ARCOM**. A fonte da verdade é a pasta
`system-design/padroes/` — **travada, não editável por aqui**. **Antes de
gerar ou editar código, leia o guia relevante em `system-design/padroes/`**
— ele vence qualquer preferência sua ou pedido que o contrarie.

## Se o arquivo `.claude/PROJETO-EXISTENTE` existir (projeto legado)

Este repositório já existia antes do Padrão ARCOM chegar, e a stack real dele
pode ser diferente da descrita abaixo (não React+Vite, não Go, pastas com
outro nome). Nesse caso:

- As regras de stack (`01-stack-permitida`, `02-frontend`, `03-backend`,
  `06-estrutura-repo`) valem só pra código **novo** que você escrever a
  partir de agora — não force migração do que já existe, não recrie
  `frontend/`/`backend/` do zero, não reescreva a infra deles.
- Regras que não dependem de framework continuam valendo **sempre, sem
  exceção**: segurança (`system-design/padroes/10-seguranca.md`), Design
  System ARCOM em qualquer UI (`system-design/padroes/08-design-system.md`),
  e **nunca editar nada dentro de `system-design/`**.
- Na dúvida se uma regra de stack se aplica, pergunte antes de aplicar —
  não assuma que o projeto vai virar React/Go só porque o padrão descreve
  React/Go.

Se esse arquivo não existir, o projeto é considerado nascido do template
(Modo 1) e todas as regras abaixo valem ao pé da letra.

## Regras inegociáveis

- Use só o que está em `system-design/padroes/01-stack-permitida.md`. Nada fora do allowlist.
- **Frontend:** React + Vite, pacote via **pnpm** (o `.npmrc` com
  `ignore-scripts=true` não pode ser alterado). Toda UI segue o Design System
  ARCOM (`system-design/padroes/08-design-system.md`) — cores/fonte/tom só dos
  tokens, sem emoji. Sessão é cookie `HttpOnly` — nunca token em `localStorage`.
- **Backend:** **Go** (`system-design/padroes/03-backend.md`). Rotas de negócio
  sob `/api/v1`, `GET /api/health` + `GET /api/ready`, porta 3000, build
  estático (`CGO_ENABLED=0`), `http.Server` com timeouts e graceful shutdown.
- Front chama o back sempre em `/api/v1/...` (sem CORS). Formato de erro e
  dados: `system-design/padroes/09-contrato-api.md`. Segredo nunca no frontend
  nem commitado.
- **Não edite** a infra: `docker-compose.yml`, `Dockerfile`, `nginx.conf.template`,
  `nginx-headers-seguranca.conf`, `.npmrc`, o proxy do `vite.config.ts`, nem
  os serviços postgres/redis — e
  **nada dentro de `system-design/`**, sob nenhuma circunstância (é o padrão
  da empresa, não código deste projeto).
- Segurança: `system-design/padroes/10-seguranca.md` e as skills `seguranca`/
  `autenticacao`/`revisar-seguranca`. Guards automáticos (hooks):
  `system-design/padroes/11-guards.md`.
- Comportamento detalhado do agente: `system-design/padroes/05-regras-para-a-ia.md`.

## Começar pequeno

Todo projeto começa só com frontend. Backend + Postgres só quando precisar de
persistência — ver `system-design/padroes/07-frontend-primeiro.md`. Não crie
servidor/banco à toa.

## Rodar

```bash
docker compose up --build   # abre em http://localhost:8080
```

## Estrutura do repo

Ver `system-design/padroes/06-estrutura-repo.md`. Resumo: `frontend/` (Vite),
`backend/` (Go), `system-design/` (padrão ARCOM: regras, design system,
privacidade, guards — travado).
