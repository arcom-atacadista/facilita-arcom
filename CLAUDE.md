# Facilita ARCOM — instruções para o Claude Code

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


---

## Este projeto: Facilita ARCOM

Plataforma de crédito e cobrança: carteira em atraso de 3 a 90 dias, régua de
cobrança por faixa, e negociação self-service pelo próprio cliente via link.

### Onde está o quê

- `backend/internal/acesso` — usuários, papéis, alçada e sessão (cookie
  HttpOnly com token opaco; não usamos JWT).
- `backend/internal/cobranca` — clientes, dívidas, políticas, acordos e
  parcelas. Cliente/dívida/acordo ficam no mesmo pacote porque são um
  agregado só.
- `backend/internal/negociacao` — a tela pública por token, sem sessão.
- `backend/internal/disparo` — régua de cobrança: fila, montagem da mensagem
  e worker.
- `backend/internal/gatewayarcom` — leitura dos dados reais da empresa.
- `backend/internal/problema` — contrato de erro compartilhado. Mora fora de
  `servidor` porque `rotas.go` importa as features e as features precisam dos
  construtores de erro — seria ciclo de import. `servidor.NaoEncontrado`
  continua valendo por alias.

### Regras deste domínio

- **Quem vê o quê é o backend que decide.** Analista enxerga apenas a
  carteira dele (`usuarios.codigo_cobranca` = `dividas.responsavel_cobranca`);
  coordenação pra cima vê tudo; quem não tem código não vê nada. Dado fora do
  escopo responde 404, nunca 403 — 403 confirmaria que o registro existe.
  Toda consulta nova a dívida, acordo ou disparo passa por esse recorte.
- **Alçada é do servidor.** Desconto acima do teto do usuário é 403
  `acima_da_alcada`. A tela avisa antes por conveniência; ela não protege.
- **Dinheiro em centavo inteiro.** Arredonde com `math.Round(v*100)` numa
  etapa só. Arredondar para duas casas e depois multiplicar perde centavo
  (isso já apareceu em produção de mentira: R$ 1.234.567,89 virava ,88).
- **Nome do cliente** sai de `cobranca.NomeDeTratamento`: a carteira é quase
  toda PJ, e cortar na primeira palavra transformava "Mercado do João LTDA"
  em "Mercado".

### Duas coisas dependem de resposta da ARCOM

Estão marcadas no código e não devem ser resolvidas por adivinhação:

1. **Canal de envio de mensagem** (`internal/disparo/canal.go`). Não existe.
   O Gateway é somente leitura, não há provedor de mensagem no allowlist, e
   nome de fila do RabbitMQ vem da infra. A fila funciona e nada é marcado
   como enviado sem ter sido.
2. **Semântica de dois campos do Gateway** (`internal/gatewayarcom/debitos.go`):
   qual data é o vencimento e qual valor é o saldo. O `api.md` marca essa
   semântica como "a revisar".

### Rodar local sem Docker

```bash
# backend
DATABASE_URL=postgres://app@localhost:5432/app APP_URL=http://localhost:8080 \
  go run ./cmd/migrate && go run ./cmd/server

# frontend (proxy de /api já configurado)
cd frontend && pnpm dev
```

Os testes de integração do backend pulam sozinhos sem `DATABASE_URL_TESTE`.
Com um Postgres à mão:

```bash
cd backend && DATABASE_URL_TESTE=postgres://app@localhost:5432/teste go test -race ./...
```
