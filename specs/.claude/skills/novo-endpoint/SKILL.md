---
name: novo-endpoint
description: Cria um novo endpoint/recurso no backend Go sob /api/v1, seguindo o Padrão ARCOM. Use quando o usuário pedir uma rota, API, endpoint ou lógica de servidor nova.
---

# Novo endpoint (backend Go)

Antes de escrever, leia `padroes/03-backend.md`, `padroes/09-contrato-api.md`
e `padroes/01-stack-permitida.md`. Se o endpoint recebe dado do usuário,
autentica ou lida com dado sensível, leia também a skill `seguranca` (e
`autenticacao` se for rota protegida).

Só crie backend se o pedido exigir persistência, login, segredo ou API
externa (ver `07-frontend-primeiro.md`) — senão resolva no frontend.

## Passos

1. Um pacote por recurso em `backend/internal/<recurso>/`, com
   `handler.go` (HTTP), `service.go` (regra de negócio), `repo.go` (acesso a
   dados), `model.go` (entidade + DTOs). Não crie `handlers/`/`services/`/
   `repositories/` paralelos — a camada é o **arquivo**, não a pasta.
2. Registre a rota em `backend/internal/servidor/rotas.go`, sob `/api/v1`
   (`v1.Route("/recurso", recurso.Rotas)`). Se for privada, dentro de um
   grupo com o middleware de sessão (ver skill `autenticacao`).
3. Handler usa o wrapper `s.H(fn)` (`internal/servidor/problema.go`) e
   devolve `error`, nunca escreve JSON de erro na mão. Erro de domínio vem
   dos construtores (`servidor.NaoEncontrado(...)`, `servidor.Conflito(...)`,
   `servidor.Validacao(campos)`) — formato de resposta em
   `09-contrato-api.md`.
4. Valide toda entrada do cliente com `go-playground/validator` nas structs
   de DTO — nunca confie no frontend. Se o handler recebe um id de recurso,
   confira que o usuário autenticado é o **dono** dele (ownership).
5. Config (DB, Redis, segredos) sempre via `internal/config` — nunca
   hardcode host, senha ou chave. Segredo novo que a feature precisa:
   `config.Obrigatorio("NOME_DA_CHAVE")`.

## Regras

- Só módulos Go do allowlist (`01-stack-permitida.md`). Nada de outra
  linguagem. Proibido `pkg/`, `utils/`, `common/`, `helpers/` e interface com
  uma única implementação.
- Escutar na porta do `PORT` (default 3000); toda rota de negócio sob
  `/api/v1`.
- Build estático (`CGO_ENABLED=0`) — não quebre isso.
- SQL sempre parametrizado (`?`, nunca `fmt.Sprintf`/concatenação — ver skill
  `seguranca`). Update de recurso nunca aceita o corpo inteiro do request sem
  allowlist de campo (mass assignment).
- Precisa de banco? Postgres via `DATABASE_URL`; cache/fila via Redis.
  Migration com `goose`, arquivo em `internal/db/migrations/`, sempre com
  `Down` preenchido — peça confirmação antes de migration que altere/apague
  dados. Migration roda via `cmd/migrate` (serviço `migrate` no compose),
  nunca no boot do servidor.
- Depois de pronto, rode (ou peça pra rodar) `go vet ./...`,
  `golangci-lint run`, `go test -race ./...` antes de considerar terminado.
