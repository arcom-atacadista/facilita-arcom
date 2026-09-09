-- O acordo passa a ser do CNPJ, não de um título isolado.
--
-- POR QUE
--
-- É regra de negócio da ARCOM: não se parcela um título sozinho — o
-- parcelamento sempre engloba todos os títulos em atraso do CNPJ. Um cliente
-- com seis documentos vencidos fecha um acordo, não seis.
--
-- O modelo antigo (um acordo por dívida) não conseguia representar isso, e a
-- inconsistência era pior que o modelo errado: a tela calcularia a condição
-- sobre a posição consolidada, o cliente aceitaria esse valor, e apenas um dos
-- títulos ficaria marcado como negociado. Os outros cinco continuariam na
-- régua, cobrando alguém que já fechou acordo.
--
-- O QUE MUDA
--
-- acordos.divida_id sai; entra acordos.cliente_id, e a tabela acordo_dividas
-- registra quais títulos aquele acordo cobre. O índice de "um acordo ativo"
-- passa a valer por cliente, que é o novo escopo.
--
-- NADA É PERDIDO: cada acordo existente vira um acordo do cliente daquela
-- dívida, com uma linha em acordo_dividas apontando para o título original.
-- A migration copia antes de derrubar a coluna, e o Down reconstrói o
-- divida_id a partir da tabela de ligação.
--
-- +goose Up
-- +goose StatementBegin
CREATE TABLE acordo_dividas (
  acordo_id UUID NOT NULL REFERENCES acordos (id) ON DELETE CASCADE,
  divida_id UUID NOT NULL REFERENCES dividas (id) ON DELETE CASCADE,
  -- Saldo e encargos do título no momento do acordo. Guardados porque a
  -- sincronização diária atualiza a dívida: sem a foto, um acordo fechado hoje
  -- deixaria de bater com a soma dos títulos amanhã.
  saldo_no_acordo    NUMERIC(14,2) NOT NULL CHECK (saldo_no_acordo > 0),
  encargos_no_acordo NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (encargos_no_acordo >= 0),
  PRIMARY KEY (acordo_id, divida_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX acordo_dividas_divida_idx ON acordo_dividas (divida_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE acordos ADD COLUMN cliente_id UUID REFERENCES clientes (id) ON DELETE CASCADE;
-- +goose StatementEnd

-- Migra o que existe: o cliente vem da dívida, e a ligação preserva o título.
-- +goose StatementBegin
UPDATE acordos a
   SET cliente_id = d.cliente_id
  FROM dividas d
 WHERE d.id = a.divida_id;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO acordo_dividas (acordo_id, divida_id, saldo_no_acordo, encargos_no_acordo)
SELECT a.id, a.divida_id, d.valor_original, d.valor_encargos
  FROM acordos a
  JOIN dividas d ON d.id = a.divida_id;
-- +goose StatementEnd

-- Só agora o cliente_id pode ser obrigatório: acordo órfão de cliente não
-- existe, e deixar a coluna anulável abriria espaço para um aparecer.
-- +goose StatementBegin
ALTER TABLE acordos ALTER COLUMN cliente_id SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS acordos_um_ativo_por_divida;
-- +goose StatementEnd

-- Um acordo ativo por cliente. É o que impede duas negociações concorrentes
-- sobre a mesma posição — antes valia por título, o que deixava seis acordos
-- ativos conviverem para o mesmo CNPJ.
-- +goose StatementBegin
CREATE UNIQUE INDEX acordos_um_ativo_por_cliente ON acordos (cliente_id) WHERE status = 'ativo';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE acordos DROP COLUMN divida_id;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX acordos_cliente_idx ON acordos (cliente_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE acordos ADD COLUMN divida_id UUID REFERENCES dividas (id) ON DELETE CASCADE;
-- +goose StatementEnd

-- Reconstrói o vínculo antigo. Acordo que cobre mais de um título só cabe no
-- modelo antigo pelo primeiro deles — o Down volta a estrutura, não a
-- informação que o modelo antigo nunca soube guardar.
-- +goose StatementBegin
UPDATE acordos a
   SET divida_id = (
     SELECT ad.divida_id FROM acordo_dividas ad
      WHERE ad.acordo_id = a.id
      ORDER BY ad.divida_id
      LIMIT 1
   );
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM acordos WHERE divida_id IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE acordos ALTER COLUMN divida_id SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS acordos_um_ativo_por_cliente;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX IF EXISTS acordos_cliente_idx;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX acordos_um_ativo_por_divida ON acordos (divida_id) WHERE status = 'ativo';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE acordos DROP COLUMN cliente_id;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS acordo_dividas;
-- +goose StatementEnd
