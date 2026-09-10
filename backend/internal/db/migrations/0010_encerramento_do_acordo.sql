-- Registra quem encerrou o acordo e quando.
--
-- Romper um acordo devolve os títulos para a régua e o cliente volta a ser
-- cobrado; cancelar apaga uma negociação que a operação tinha como fechada.
-- São decisões de gente, com consequência para o cliente, e a tabela guardava
-- só o status novo — dava para ver que mudou, nunca por ordem de quem.
--
-- criado_por e aprovado_por já existiam pela mesma razão, do outro lado do
-- ciclo. Isto fecha a ponta que faltava.
--
-- +goose Up
-- +goose StatementBegin
ALTER TABLE acordos
  ADD COLUMN IF NOT EXISTS encerrado_por UUID REFERENCES usuarios (id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS encerrado_em  TIMESTAMPTZ,
  -- Texto livre curto para o motivo, quando quem encerra quiser deixar
  -- registrado. Opcional: obrigar justificativa produz "asdf" em campo de
  -- formulário, não informação.
  ADD COLUMN IF NOT EXISTS motivo_encerramento TEXT;
-- +goose StatementEnd

-- Acordo encerrado pela sincronização (cliente pagou e o título sumiu do
-- Gateway) fica com encerrado_por nulo e encerrado_em preenchido — é assim que
-- se distingue "o sistema detectou" de "alguém decidiu".
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS acordos_encerrado_em_idx ON acordos (encerrado_em)
  WHERE encerrado_em IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS acordos_encerrado_em_idx;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE acordos
  DROP COLUMN IF EXISTS encerrado_por,
  DROP COLUMN IF EXISTS encerrado_em,
  DROP COLUMN IF EXISTS motivo_encerramento;
-- +goose StatementEnd
