-- Entrada de mensagem: conversas e o que o cliente responde.
--
-- Até aqui a comunicação era de mão única. Com o webhook da Meta, a
-- plataforma passa a saber o que o cliente respondeu — o que habilita a mesa
-- de atendimento e, de quebra, derruba custo: dentro da janela de 24 horas
-- aberta pela resposta, as mensagens deixam de ser tarifadas.
--
-- +goose Up
-- +goose StatementBegin
CREATE TABLE conversas (
  id         UUID PRIMARY KEY,
  -- O telefone com DDI é a chave natural: é por ele que a Meta identifica a
  -- pessoa, e ele existe mesmo quando o número não está na nossa carteira.
  telefone   TEXT NOT NULL UNIQUE,
  -- Casado com o cliente quando reconhecemos o número. Fica nulo quando
  -- alguém escreve de um número que não está cadastrado — e isso não pode
  -- fazer a mensagem se perder.
  cliente_id UUID REFERENCES clientes (id) ON DELETE SET NULL,
  -- Nome que a pessoa usa no WhatsApp, que nem sempre é a razão social.
  nome_perfil TEXT,

  -- Fim da janela de atendimento de 24h, como a própria Meta informa em
  -- conversation.expiration_timestamp. Dentro dela dá para responder com
  -- texto livre e sem custo; fora dela, só template tarifado.
  janela_expira_em   TIMESTAMPTZ,

  ultima_mensagem_em TIMESTAMPTZ,
  nao_lidas          INT NOT NULL DEFAULT 0 CHECK (nao_lidas >= 0),
  criado_em          TIMESTAMPTZ NOT NULL DEFAULT now(),
  atualizado_em      TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- A mesa ordena por quem falou por último.
-- +goose StatementBegin
CREATE INDEX conversas_ultima_mensagem_idx ON conversas (ultima_mensagem_em DESC NULLS LAST);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX conversas_cliente_idx ON conversas (cliente_id) WHERE cliente_id IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE mensagens (
  id          UUID PRIMARY KEY,
  conversa_id UUID NOT NULL REFERENCES conversas (id) ON DELETE CASCADE,
  direcao     TEXT NOT NULL CHECK (direcao IN ('entrada', 'saida')),

  -- Id da mensagem na Meta. UNIQUE porque o webhook é reentregue quando a
  -- nossa resposta demora ou falha: sem isso, uma reentrega viraria mensagem
  -- duplicada na tela do operador.
  wamid TEXT UNIQUE,

  tipo  TEXT NOT NULL DEFAULT 'text',
  texto TEXT,

  -- Preenchido só na saída, conforme os avisos de status chegam.
  status      TEXT CHECK (status IS NULL OR status IN ('enviada', 'entregue', 'lida', 'falhou')),
  erro_codigo INT,

  ocorrida_em TIMESTAMPTZ NOT NULL,
  criado_em   TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX mensagens_conversa_idx ON mensagens (conversa_id, ocorrida_em);
-- +goose StatementEnd

-- O disparo ganha o rastro de entrega que só o webhook consegue informar.
-- Ficam como colunas próprias em vez de novos valores em disparos.status
-- para não misturar o estado da FILA (na_fila, enviado, erro) com o estado da
-- ENTREGA, que é outra coisa e continua evoluindo depois de a fila terminar.
-- +goose StatementBegin
ALTER TABLE disparos
  ADD COLUMN IF NOT EXISTS entregue_em TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS lido_em     TIMESTAMPTZ;
-- +goose StatementEnd

-- O status chega pelo wamid, então a busca por ele precisa ser indexada.
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS disparos_referencia_idx ON disparos (referencia_externa)
  WHERE referencia_externa IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS disparos_referencia_idx;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE disparos DROP COLUMN IF EXISTS entregue_em, DROP COLUMN IF EXISTS lido_em;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS mensagens;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS conversas;
-- +goose StatementEnd
