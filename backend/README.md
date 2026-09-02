# backend/ (Go)

Servidor em Go. Entrada em `cmd/server/main.go`, expõe `GET /api/health`
(liveness) e `GET /api/ready` (readiness) fora de versão, e as rotas de
negócio sob `/api/v1`. Escuta na porta 3000. `cmd/migrate/main.go` aplica as
migrations do Postgres (goose) — roda como serviço separado no compose, nunca
no boot do servidor.

- Dependências: `go mod` (commite `go.mod` e `go.sum`).
- Build: binário estático (`CGO_ENABLED=0`).
- Qualidade: `go vet ./...`, `golangci-lint run`, `govulncheck ./...`,
  `go test -race ./...`.
- Convenções, contrato de erro e libs liberadas: veja `padroes/03-backend.md`,
  `padroes/09-contrato-api.md` e `padroes/01-stack-permitida.md`.
