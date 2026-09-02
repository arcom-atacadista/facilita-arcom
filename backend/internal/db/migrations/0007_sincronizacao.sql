-- Rastro da sincronização com o Gateway ARCOM.
--
-- A carteira deixa de ser semeada à mão e passa a espelhar o dataset
-- `debitos`. Duas colunas tornam isso seguro:
--
--   origem          — separa a dívida que veio do Gateway da que alguém
--                     lançou na mão. Só a primeira pode ser fechada
--                     automaticamente por sumir da origem.
--   sincronizado_em — quando a linha foi vista pela última vez no Gateway.
--                     É o que permite descobrir o que saiu de lá.
--
-- Fechar o que sumiu importa: dívida que desaparece do `debitos` foi quase
-- certamente paga, e continuar cobrando quem pagou é pior do que deixar de
-- cobrar quem deve.
--
-- +goose Up
-- +goose StatementBegin
ALTER TABLE dividas
  ADD COLUMN IF NOT EXISTS origem TEXT NOT NULL DEFAULT 'manual'
    CHECK (origem IN ('manual', 'gateway')),
  ADD COLUMN IF NOT EXISTS sincronizado_em TIMESTAMPTZ;
-- +goose StatementEnd

-- O fechamento automático varre por (origem, status, sincronizado_em).
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS dividas_sincronizacao_idx
  ON dividas (origem, status, sincronizado_em)
  WHERE origem = 'gateway';
-- +goose StatementEnd

-- Registro de cada rodada, para a operação conseguir responder "a carteira de
-- hoje entrou?" sem depender de ler log de servidor.
-- +goose StatementBegin
CREATE TABLE sincronizacoes (
  id             UUID PRIMARY KEY,
  iniciada_em    TIMESTAMPTZ NOT NULL,
  terminada_em   TIMESTAMPTZ,
  status         TEXT NOT NULL DEFAULT 'rodando'
                 CHECK (status IN ('rodando', 'concluida', 'falhou')),
  lidas          INT NOT NULL DEFAULT 0,
  criadas        INT NOT NULL DEFAULT 0,
  atualizadas    INT NOT NULL DEFAULT 0,
  fechadas       INT NOT NULL DEFAULT 0,
  ignoradas      INT NOT NULL DEFAULT 0,
  erro_detalhe   TEXT
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX sincronizacoes_iniciada_idx ON sincronizacoes (iniciada_em DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sincronizacoes;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS dividas_sincronizacao_idx;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE dividas DROP COLUMN IF EXISTS origem, DROP COLUMN IF EXISTS sincronizado_em;
-- +goose StatementEnd
