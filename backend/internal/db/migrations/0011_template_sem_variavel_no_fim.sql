-- Os três templates terminavam no link, e a Meta recusa isso.
--
-- Regra da plataforma: o corpo de um template não pode COMEÇAR nem TERMINAR
-- num marcador, e dois marcadores não podem ficar colados. Um corpo que acaba
-- em {{5}} é reprovado na submissão com "dangling parameter" — ou seja, os
-- três templates que a régua espera não passariam da aprovação, e a régua
-- inteira ficaria sem canal fora da janela de atendimento.
--
-- O fecho escolhido não é enchimento: "ou responda esta mensagem" convida a
-- resposta que ABRE a janela de 24 horas. É dentro dela que a mesa de
-- atendimento conversa com o cliente em texto livre, sem tarifa por mensagem.
--
-- A ordem dos marcadores continua a mesma — nome, contrato, valor, dias,
-- link — porque é ela que campanhas.parametros mapeia para {{1}}..{{5}}.
-- Mexer no texto sem mexer na ordem é seguro; trocar a ordem exigiria
-- reaprovar o template na Meta.
--
-- +goose Up
-- +goose StatementBegin
UPDATE campanhas SET template =
  'Olá {nome}. Identificamos o contrato {contrato}, no valor de {valor}, com {dias} dias de atraso. Para consultar e regularizar, acesse {link} ou responda esta mensagem.'
WHERE nome = 'Lembrete amigável';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE campanhas SET template =
  'Olá {nome}. O contrato {contrato}, no valor de {valor}, permanece em aberto há {dias} dias. Para ver as formas de pagamento e regularizar, acesse {link} ou responda esta mensagem.'
WHERE nome = 'Cobrança ativa';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE campanhas SET template =
  'Olá {nome}. O contrato {contrato}, no valor de {valor}, está com {dias} dias de atraso. Para regularizar diretamente com a ARCOM, acesse {link} ou responda esta mensagem.'
WHERE nome = 'Aviso de atraso prolongado';
-- +goose StatementEnd

-- +goose Down
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

-- +goose StatementBegin
UPDATE campanhas SET template =
  'Olá {nome}. O contrato {contrato}, no valor de {valor}, está com {dias} dias de atraso. Para regularizar diretamente com a ARCOM, acesse: {link}'
WHERE nome = 'Aviso de atraso prolongado';
-- +goose StatementEnd
