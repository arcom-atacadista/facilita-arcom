-- Domínio de crédito e cobrança do Facilita ARCOM.
--
-- Porte do schema que rodava no Supabase. Três diferenças de propósito:
--
-- 1. Identidade é própria (usuarios + sessoes), não auth.users — a sessão é
--    cookie HttpOnly com token opaco, ver .claude/skills/autenticacao.
-- 2. Não há RLS. No Supabase o navegador falava direto com o banco, então a
--    policy era a única defesa; aqui quem decide quem vê o quê é o backend,
--    que é o único a ter a credencial do Postgres.
-- 3. O link de negociação é guardado como hash e tem validade. No modelo
--    antigo o token ficava em claro e valia pra sempre.
--
-- +goose Up
-- +goose StatementBegin
CREATE TYPE papel_usuario AS ENUM ('aprendiz', 'analista', 'coordenacao', 'gerencia');
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TYPE equipe_usuario AS ENUM ('interno', 'filiais');
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE usuarios (
  id             UUID PRIMARY KEY,
  nome           TEXT NOT NULL,
  email          TEXT NOT NULL,
  senha_hash     TEXT NOT NULL,
  papel          papel_usuario NOT NULL DEFAULT 'aprendiz',
  equipe         equipe_usuario NOT NULL DEFAULT 'interno',
  ativo          BOOLEAN NOT NULL DEFAULT TRUE,
  -- Teto de desconto que o usuário pode conceder sozinho, em pontos
  -- percentuais. Vem do papel na criação, mas pode ser ajustado caso a caso
  -- pela coordenação — por isso é coluna, não função do papel.
  alcada_maxima  NUMERIC(5,2) NOT NULL DEFAULT 20 CHECK (alcada_maxima BETWEEN 0 AND 100),
  criado_em      TIMESTAMPTZ NOT NULL DEFAULT now(),
  atualizado_em  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- E-mail corporativo é único sem diferenciar maiúscula/minúscula: sem isso
-- "Ana@arcom" e "ana@arcom" viram duas contas para a mesma pessoa.
-- +goose StatementBegin
CREATE UNIQUE INDEX usuarios_email_unico ON usuarios (lower(email));
-- +goose StatementEnd

-- Sessão opaca: o banco guarda só o SHA-256 do token que foi pro cookie.
-- Vazamento desta tabela não devolve nenhuma sessão utilizável.
-- +goose StatementBegin
CREATE TABLE sessoes (
  id            UUID PRIMARY KEY,
  usuario_id    UUID NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
  token_hash    BYTEA NOT NULL UNIQUE,
  criada_em     TIMESTAMPTZ NOT NULL DEFAULT now(),
  expira_em     TIMESTAMPTZ NOT NULL,
  ultimo_uso_em TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX sessoes_usuario_idx ON sessoes (usuario_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX sessoes_expira_idx ON sessoes (expira_em);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE clientes (
  id            UUID PRIMARY KEY,
  nome          TEXT NOT NULL,
  documento     TEXT NOT NULL UNIQUE,
  telefone      TEXT,
  email         TEXT,
  criado_em     TIMESTAMPTZ NOT NULL DEFAULT now(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE dividas (
  id             UUID PRIMARY KEY,
  cliente_id     UUID NOT NULL REFERENCES clientes (id) ON DELETE CASCADE,
  contrato       TEXT NOT NULL,
  valor_original NUMERIC(14,2) NOT NULL CHECK (valor_original > 0),
  vencimento     DATE NOT NULL,
  status         TEXT NOT NULL DEFAULT 'aberto'
                 CHECK (status IN ('aberto', 'negociado', 'quitado', 'cancelado')),

  -- Recorte da carteira. Vêm do Gateway ARCOM (dataset debitos, campos
  -- responsavel_cobranca e filial) e são o que permite mostrar a um analista
  -- só a carteira dele em vez da base inteira de devedores.
  responsavel_cobranca TEXT,
  filial               TEXT,

  -- Link público de negociação: guardamos o hash, nunca o token. O texto em
  -- claro existe uma vez só, no instante em que a mensagem é montada.
  token_hash       BYTEA UNIQUE,
  token_expira_em  TIMESTAMPTZ,

  criado_em     TIMESTAMPTZ NOT NULL DEFAULT now(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (cliente_id, contrato)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX dividas_vencimento_idx ON dividas (vencimento);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX dividas_status_idx ON dividas (status);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX dividas_responsavel_idx ON dividas (responsavel_cobranca);
-- +goose StatementEnd

-- Política comercial por faixa de atraso: quanto de desconto e em quantas
-- parcelas o cliente pode fechar sozinho, sem passar por um analista.
-- +goose StatementBegin
CREATE TABLE politicas (
  id                 UUID PRIMARY KEY,
  nome               TEXT NOT NULL,
  faixa_min          INT NOT NULL CHECK (faixa_min >= 0),
  faixa_max          INT NOT NULL,
  desconto_avista    NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (desconto_avista BETWEEN 0 AND 100),
  desconto_parcelado NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (desconto_parcelado BETWEEN 0 AND 100),
  max_parcelas       INT NOT NULL DEFAULT 6 CHECK (max_parcelas BETWEEN 1 AND 24),
  entrada_minima_pct NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (entrada_minima_pct BETWEEN 0 AND 100),
  criado_em          TIMESTAMPTZ NOT NULL DEFAULT now(),
  atualizado_em      TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (faixa_max >= faixa_min)
);
-- +goose StatementEnd

-- Duas políticas não podem cobrir o mesmo dia de atraso: com faixas
-- sobrepostas, qual desconto o cliente recebe passaria a depender da ordem de
-- leitura no banco. int4range com && no exclusion constraint resolve isso no
-- próprio banco, e não numa validação de aplicação que alguém esquece.
-- +goose StatementBegin
ALTER TABLE politicas ADD CONSTRAINT politicas_faixa_sem_sobreposicao
  EXCLUDE USING gist (int4range(faixa_min, faixa_max, '[]') WITH &&);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE campanhas (
  id            UUID PRIMARY KEY,
  nome          TEXT NOT NULL,
  faixa_min     INT NOT NULL CHECK (faixa_min >= 0),
  faixa_max     INT NOT NULL,
  canal         TEXT NOT NULL DEFAULT 'whatsapp',
  template      TEXT NOT NULL,
  ativo         BOOLEAN NOT NULL DEFAULT TRUE,
  criado_em     TIMESTAMPTZ NOT NULL DEFAULT now(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (faixa_max >= faixa_min)
);
-- +goose StatementEnd

-- Fila de mensagens a enviar. É uma tabela e não uma fila em Redis de
-- propósito: disparo de cobrança é registro que precisa sobreviver a
-- restart e ser auditável depois (ver 05-regras-para-a-ia.md sobre Redis).
-- +goose StatementBegin
CREATE TABLE disparos (
  id             UUID PRIMARY KEY,
  divida_id      UUID NOT NULL REFERENCES dividas (id) ON DELETE CASCADE,
  campanha_id    UUID REFERENCES campanhas (id) ON DELETE SET NULL,
  telefone       TEXT NOT NULL,
  mensagem       TEXT NOT NULL,
  canal          TEXT NOT NULL DEFAULT 'whatsapp',
  status         TEXT NOT NULL DEFAULT 'na_fila'
                 CHECK (status IN ('na_fila', 'enviado', 'erro', 'cancelado')),
  tentativas     INT NOT NULL DEFAULT 0,
  erro_detalhe   TEXT,
  -- Identificador devolvido pelo canal de envio, pra conciliar depois com o
  -- histórico do Gateway (dataset disparo-mensagem-nines, campo idReferencia).
  referencia_externa TEXT,
  agendado_para  TIMESTAMPTZ NOT NULL DEFAULT now(),
  enviado_em     TIMESTAMPTZ,
  criado_por     UUID REFERENCES usuarios (id) ON DELETE SET NULL,
  criado_em      TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- Índice do worker: ele procura sempre "na_fila e já venceu o agendamento".
-- +goose StatementBegin
CREATE INDEX disparos_fila_idx ON disparos (status, agendado_para) WHERE status = 'na_fila';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX disparos_divida_idx ON disparos (divida_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE acordos (
  id             UUID PRIMARY KEY,
  divida_id      UUID NOT NULL REFERENCES dividas (id) ON DELETE CASCADE,
  tipo_pagamento TEXT NOT NULL DEFAULT 'pix' CHECK (tipo_pagamento IN ('pix', 'boleto')),
  desconto_pct   NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (desconto_pct BETWEEN 0 AND 100),
  entrada        NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (entrada >= 0),
  parcelas       INT NOT NULL DEFAULT 1 CHECK (parcelas BETWEEN 1 AND 24),
  valor_total    NUMERIC(14,2) NOT NULL CHECK (valor_total > 0),
  status         TEXT NOT NULL DEFAULT 'ativo'
                 CHECK (status IN ('ativo', 'quitado', 'rompido', 'cancelado')),
  origem         TEXT NOT NULL DEFAULT 'cliente' CHECK (origem IN ('cliente', 'operador')),
  -- Quem fechou (nulo quando o próprio cliente fechou pelo link) e quem
  -- autorizou, quando o desconto passou da alçada de quem fechou.
  criado_por     UUID REFERENCES usuarios (id) ON DELETE SET NULL,
  aprovado_por   UUID REFERENCES usuarios (id) ON DELETE SET NULL,
  criado_em      TIMESTAMPTZ NOT NULL DEFAULT now(),
  atualizado_em  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- Uma dívida só pode ter um acordo vivo por vez. No modelo antigo isso era
-- checado lendo antes de gravar, o que deixa a corrida aberta: dois aceites
-- simultâneos do mesmo link criavam dois acordos.
-- +goose StatementBegin
CREATE UNIQUE INDEX acordos_um_ativo_por_divida ON acordos (divida_id) WHERE status = 'ativo';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE parcelas (
  id         UUID PRIMARY KEY,
  acordo_id  UUID NOT NULL REFERENCES acordos (id) ON DELETE CASCADE,
  numero     INT NOT NULL CHECK (numero > 0),
  valor      NUMERIC(14,2) NOT NULL CHECK (valor > 0),
  vencimento DATE NOT NULL,
  pago       BOOLEAN NOT NULL DEFAULT FALSE,
  pago_em    TIMESTAMPTZ,
  criado_em  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (acordo_id, numero)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX parcelas_vencimento_idx ON parcelas (vencimento) WHERE pago = FALSE;
-- +goose StatementEnd

-- Políticas e campanhas iniciais: as mesmas três faixas que a operação já
-- usa hoje (3-30, 31-60, 61-90).
-- +goose StatementBegin
INSERT INTO politicas (id, nome, faixa_min, faixa_max, desconto_avista, desconto_parcelado, max_parcelas, entrada_minima_pct) VALUES
  (gen_random_uuid(), 'Atraso recente',  3,  30,  5,  0,  3,  0),
  (gen_random_uuid(), 'Atraso médio',   31,  60, 10,  5,  6, 10),
  (gen_random_uuid(), 'Atraso avançado', 61, 90, 20, 10, 10, 15);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO campanhas (id, nome, faixa_min, faixa_max, template) VALUES
  (gen_random_uuid(), 'Lembrete amigável',   3, 30, 'Olá {nome}, identificamos o contrato {contrato} com {dias} dias de atraso, no valor de {valor}. Regularize com desconto pelo link: {link}'),
  (gen_random_uuid(), 'Cobrança ativa',     31, 60, 'Olá {nome}, seu contrato {contrato} está com {dias} dias de atraso ({valor}). Temos condições especiais para você: {link}'),
  (gen_random_uuid(), 'Última oportunidade', 61, 90, '{nome}, o contrato {contrato} está com {dias} dias de atraso ({valor}). Negocie agora com o melhor desconto: {link}');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS parcelas;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS acordos;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS disparos;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS campanhas;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS politicas;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS dividas;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS clientes;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS sessoes;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS usuarios;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TYPE IF EXISTS equipe_usuario;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TYPE IF EXISTS papel_usuario;
-- +goose StatementEnd
