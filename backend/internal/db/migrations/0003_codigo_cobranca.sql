-- Liga o operador à carteira dele: casa com dividas.responsavel_cobranca,
-- que vem do Gateway ARCOM (dataset debitos).
--
-- Vazio significa "não é dono de carteira nenhuma", e é assim que um analista
-- recém-cadastrado não vê dívida alguma em vez de ver todas — falha fechado,
-- porque o que está em jogo é dado pessoal de devedor. Quem enxerga a
-- carteira inteira é coordenação pra cima, por papel, não por este campo.
--
-- Vai numa migration própria e não dentro da 0002 porque migration já
-- aplicada não roda de novo: mexer na 0002 deixaria qualquer banco que já a
-- rodou sem a coluna, sem nenhum aviso.
--
-- +goose Up
-- +goose StatementBegin
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS codigo_cobranca TEXT;
-- +goose StatementEnd

-- Índice para o filtro de carteira do analista, que roda em toda listagem.
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS usuarios_codigo_cobranca_idx ON usuarios (codigo_cobranca)
  WHERE codigo_cobranca IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS usuarios_codigo_cobranca_idx;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE usuarios DROP COLUMN IF EXISTS codigo_cobranca;
-- +goose StatementEnd
