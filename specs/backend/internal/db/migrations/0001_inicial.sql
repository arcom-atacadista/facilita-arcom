-- Primeira migration do projeto. Não faz nada no banco — só marca a versão
-- inicial pro goose (ver internal/db/migrations.go e cmd/migrate). Adicione
-- as migrations do seu projeto em arquivos novos, sempre com Up e Down:
--   goose create internal/db/migrations nome_da_migration sql
--
-- +goose Up
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
