-- Separa, na dívida, o que é mercadoria do que é juros e multa.
--
-- POR QUE ISSO É A REGRA MAIS CARA DO SISTEMA
--
-- A política da ARCOM concede desconto SOMENTE sobre juros e encargos, nunca
-- sobre o principal. O cálculo que existia aplicava o percentual da política
-- sobre o valor inteiro da dívida — e a diferença não é de ajuste, é de ordem
-- de grandeza: num saldo de R$ 84.210,00 com R$ 4.370,00 de encargos, 20% de
-- desconto valem R$ 874,00 pela política, e valeriam R$ 16.842,00 pela conta
-- antiga. Dezenove vezes mais, numa tela em que o próprio cliente fecha o
-- acordo sozinho.
--
-- COMO O CAMPO É PREENCHIDO
--
-- O dataset `debitos` do Gateway traz dois campos de valor (`vlr_liquido_deb`
-- e `deb_vlr_docto`). A diferença entre eles é o encargo. Qual dos dois é o
-- saldo e qual é o valor de face ainda está pendente de confirmação do time
-- de TI — a mesma pendência que já existia para a data de vencimento — e por
-- isso a conta mora em gatewayarcom.Mapear, atrás da configuração.
--
-- Enquanto a confirmação não vem, e para toda dívida que não veio do Gateway,
-- valor_encargos fica em 0. Isso zera o desconto calculado, e é de propósito:
-- deixar de oferecer desconto é uma conversa com o analista; oferecer desconto
-- que a empresa não autorizou é prejuízo que já saiu pela porta.
--
-- +goose Up
-- +goose StatementBegin
ALTER TABLE dividas
  ADD COLUMN IF NOT EXISTS valor_encargos NUMERIC(14,2) NOT NULL DEFAULT 0
    CHECK (valor_encargos >= 0);
-- +goose StatementEnd

-- O encargo é parte do saldo, não um acréscimo a ele: valor_original continua
-- sendo o que o cliente deve, e valor_encargos diz quanto daquilo é juros. A
-- trava impede o encargo passar o próprio saldo, que tornaria o principal
-- negativo.
-- +goose StatementBegin
ALTER TABLE dividas
  ADD CONSTRAINT dividas_encargos_dentro_do_saldo
    CHECK (valor_encargos <= valor_original);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE dividas DROP CONSTRAINT IF EXISTS dividas_encargos_dentro_do_saldo;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE dividas DROP COLUMN IF EXISTS valor_encargos;
-- +goose StatementEnd
