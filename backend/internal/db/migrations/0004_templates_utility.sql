-- Reescreve os templates da régua para se classificarem como UTILITY no
-- WhatsApp, em vez de MARKETING.
--
-- POR QUE ISSO É DINHEIRO
--
-- A Meta cobra por mensagem entregue, com preço por categoria. No Brasil a
-- diferença entre as duas é de cerca de 9x (utility ~R$ 0,04-0,05 contra
-- marketing ~R$ 0,31-0,38). Quem decide a categoria é a Meta, lendo o texto
-- do template — e desde abril de 2025 a reclassificação é automática: se
-- você pede UTILITY e o texto parece promoção, ele é aprovado como MARKETING
-- e passa a custar 9x.
--
-- A regra da Meta é explícita: template utility "não pode conter desconto,
-- oferta ou qualquer linguagem promocional". Um lembrete de pagamento que
-- oferece desconto é marketing.
--
-- Os textos anteriores diziam "Regularize com desconto", "Temos condições
-- especiais para você" e "Negocie agora com o melhor desconto" — os três
-- seriam marketing.
--
-- O QUE MUDA NA PRÁTICA: nada para o cliente. A mensagem passa a ser o aviso
-- factual da dívida (a que a Meta chama de utility), e o desconto continua
-- existindo, revelado na página de negociação que o link abre. A página é
-- nossa e não custa por acesso.
--
-- Ao editar qualquer template daqui pra frente, evite: desconto, oferta,
-- promoção, condição especial, imperdível, aproveite, últimas vagas,
-- exclusivo. Ver o teste em internal/cobranca/templates_test.go.
--
-- +goose Up
-- +goose StatementBegin
UPDATE campanhas SET template =
  'Olá {nome}. Identificamos o contrato {contrato}, no valor de {valor}, com {dias} dias de atraso. Para consultar e regularizar, acesse: {link}'
WHERE nome = 'Lembrete amigável';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE campanhas SET template =
  'Olá {nome}. O contrato {contrato}, no valor de {valor}, permanece em aberto há {dias} dias. Para consultar as formas de pagamento e regularizar, acesse: {link}'
WHERE nome = 'Cobrança ativa';
-- +goose StatementEnd

-- "Última oportunidade" era nome de campanha promocional; o texto novo é um
-- aviso factual de situação da conta, e o nome acompanha.
-- +goose StatementBegin
UPDATE campanhas SET nome = 'Aviso de atraso prolongado', template =
  'Olá {nome}. O contrato {contrato}, no valor de {valor}, está com {dias} dias de atraso. Para regularizar diretamente com a ARCOM, acesse: {link}'
WHERE nome = 'Última oportunidade';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE campanhas SET template =
  'Olá {nome}, identificamos o contrato {contrato} com {dias} dias de atraso, no valor de {valor}. Regularize com desconto pelo link: {link}'
WHERE nome = 'Lembrete amigável';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE campanhas SET template =
  'Olá {nome}, seu contrato {contrato} está com {dias} dias de atraso ({valor}). Temos condições especiais para você: {link}'
WHERE nome = 'Cobrança ativa';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE campanhas SET nome = 'Última oportunidade', template =
  '{nome}, o contrato {contrato} está com {dias} dias de atraso ({valor}). Negocie agora com o melhor desconto: {link}'
WHERE nome = 'Aviso de atraso prolongado';
-- +goose StatementEnd
