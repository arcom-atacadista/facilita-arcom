-- Prepara a régua para o envio real pela Cloud API da Meta.
--
-- Fora de uma janela de atendimento aberta, a Meta NÃO aceita texto livre:
-- só aceita um template previamente aprovado, identificado pelo nome, com os
-- valores passados em ordem para os marcadores {{1}}, {{2}}, ... Ou seja, o
-- texto que a gente monta serve para o operador ver e para o registro — não é
-- o que viaja na chamada.
--
-- Por isso a campanha passa a guardar três coisas novas: o nome do template
-- como ele foi aprovado na Meta, o idioma, e a ORDEM em que nossas variáveis
-- entram nos marcadores.
--
-- +goose Up
-- +goose StatementBegin
ALTER TABLE campanhas
  ADD COLUMN IF NOT EXISTS template_meta TEXT,
  ADD COLUMN IF NOT EXISTS idioma TEXT NOT NULL DEFAULT 'pt_BR',
  -- Nomes das nossas variáveis na ordem dos marcadores da Meta, como lista
  -- JSON (o driver do allowlist não traz conversão de array nativo):
  -- parametros[1] vira {{1}}, parametros[2] vira {{2}}, e assim por diante.
  ADD COLUMN IF NOT EXISTS parametros JSONB NOT NULL DEFAULT '[]'::jsonb;
-- +goose StatementEnd

-- Os nomes abaixo são os que precisam ser submetidos à Meta para aprovação.
-- Enquanto não forem aprovados, o envio falha com erro claro do provedor — e
-- é por isso que a submissão é a etapa 0 do plano.
-- +goose StatementBegin
UPDATE campanhas SET
  template_meta = 'arcom_cobranca_atraso_recente',
  parametros = '["nome","contrato","valor","dias","link"]'::jsonb
WHERE nome = 'Lembrete amigável';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE campanhas SET
  template_meta = 'arcom_cobranca_atraso_medio',
  parametros = '["nome","contrato","valor","dias","link"]'::jsonb
WHERE nome = 'Cobrança ativa';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE campanhas SET
  template_meta = 'arcom_cobranca_atraso_prolongado',
  parametros = '["nome","contrato","valor","dias","link"]'::jsonb
WHERE nome = 'Aviso de atraso prolongado';
-- +goose StatementEnd

-- A fila guarda o que foi montado no momento do enfileiramento, e não uma
-- referência à campanha: se alguém editar o template entre enfileirar e
-- enviar, o reenvio tem que mandar exatamente o que foi decidido antes, não o
-- texto novo.
-- +goose StatementBegin
ALTER TABLE disparos
  ADD COLUMN IF NOT EXISTS template_meta TEXT,
  ADD COLUMN IF NOT EXISTS idioma TEXT NOT NULL DEFAULT 'pt_BR',
  ADD COLUMN IF NOT EXISTS parametros JSONB NOT NULL DEFAULT '[]'::jsonb;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE disparos
  DROP COLUMN IF EXISTS template_meta,
  DROP COLUMN IF EXISTS idioma,
  DROP COLUMN IF EXISTS parametros;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE campanhas
  DROP COLUMN IF EXISTS template_meta,
  DROP COLUMN IF EXISTS idioma,
  DROP COLUMN IF EXISTS parametros;
-- +goose StatementEnd
