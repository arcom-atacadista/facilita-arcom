# Arcom Gateway API — Documentação dos Datasets

Auth: header `X-API-Key`. Base: `https://kabana-api.arcom.com.br`. Todas as rotas são `GET`.

Convenções de filtro: valor exato `campo=X` (múltiplos por vírgula = OR); intervalo `campo.gte=`, `campo.lt=`, `campo.gte=`/`campo.lte=`. Agregação: `/v1/{dataset}/agregado?por=&metrica=&top=&incluir_campos=`; nível temporal `campo:mes` (`:ano`,`:semana`,`:dia`).

Datasets documentados: 42

---

## `debitos`

- **Endpoint base:** `GET /v1/debitos`
- **Identificador (GET por id):** `REVISAR` → `GET /v1/debitos/{id}`
- **Busca textual (`q`):** `acerto`, `cidade_estado`, `cliente`, `cnpj`, `coordenador`, `documento`, `estado`, `fantasia`, `filial`, `gerente`, `inconsistencia`, `marca_rede`, `nome_consultor`, `nome_coordenador`, `nome_gerente`, `nome_resp_cob`, `origem_pedido`, `pagto_antecipado`, `ramo_atividade`, `razao_social`, `rede`, `representante`, `responsavel_cobranca`, `seq_debito`, `setor`

| Campo                  | Tipo  | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ---------------------- | ----- | :----: | :-----------: | :-----: | :------: |
| `acerto`               | text  |   ✓    |       ✓       |    ·    |    ·     |
| `cidade_estado`        | text  |   ✓    |       ✓       |    ·    |    ·     |
| `cliente`              | text  |   ✓    |       ✓       |    ·    |    ·     |
| `cnpj`                 | text  |   ✓    |       ✓       |    ·    |    ·     |
| `coordenador`          | text  |   ✓    |       ✓       |    ·    |    ·     |
| `data_emissao`         | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `data_vct`             | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `data_vencto`          | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `deb_vlr_docto`        | float |   ✓    |       ✓       |    ✓    |    ·     |
| `documento`            | text  |   ✓    |       ✓       |    ·    |    ·     |
| `estado`               | text  |   ✓    |       ✓       |    ·    |    ·     |
| `fantasia`             | text  |   ✓    |       ✓       |    ·    |    ·     |
| `filial`               | text  |   ✓    |       ✓       |    ·    |    ·     |
| `gerente`              | text  |   ✓    |       ✓       |    ·    |    ·     |
| `inconsistencia`       | text  |   ✓    |       ✓       |    ·    |    ·     |
| `marca_rede`           | text  |   ✓    |       ✓       |    ·    |    ·     |
| `nome_consultor`       | text  |   ✓    |       ✓       |    ·    |    ·     |
| `nome_coordenador`     | text  |   ✓    |       ✓       |    ·    |    ·     |
| `nome_gerente`         | text  |   ✓    |       ✓       |    ·    |    ·     |
| `nome_resp_cob`        | text  |   ✓    |       ✓       |    ·    |    ·     |
| `origem_pedido`        | text  |   ✓    |       ✓       |    ·    |    ·     |
| `pagto_antecipado`     | text  |   ✓    |       ✓       |    ·    |    ·     |
| `ramo_atividade`       | text  |   ✓    |       ✓       |    ·    |    ·     |
| `razao_social`         | text  |   ✓    |       ✓       |    ·    |    ·     |
| `rede`                 | text  |   ✓    |       ✓       |    ·    |    ·     |
| `representante`        | text  |   ✓    |       ✓       |    ·    |    ·     |
| `responsavel_cobranca` | text  |   ✓    |       ✓       |    ·    |    ·     |
| `seq_debito`           | text  |   ✓    |       ✓       |    ·    |    ·     |
| `setor`                | text  |   ✓    |       ✓       |    ·    |    ·     |
| `vlr_liquido_deb`      | float |   ✓    |       ✓       |    ✓    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/debitos?acerto.keyword=VALOR&limite=20

# agregacao: soma de deb_vlr_docto por acerto.keyword
GET /v1/debitos/agregado?por=acerto.keyword&metrica=sum:deb_vlr_docto&top=10

# serie temporal: soma de deb_vlr_docto por mes
GET /v1/debitos/agregado?por=data_emissao:mes&metrica=sum:deb_vlr_docto&data_emissao.gte=2025-01-01&data_emissao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `representante-info`

- **Endpoint base:** `GET /v1/representante-info`
- **Identificador (GET por id):** `REVISAR` → `GET /v1/representante-info/{id}`
- **Busca textual (`q`):** `bairro`, `cep`, `cnpj`, `cpf`, `ddd_telefone`, `ddd_telefone_aux`, `desc_cidade`, `empresa_funcionario`, `endereco`, `estado`, `funcionario`, `latitude`, `logradouro`, `longitude`, `nome`, `nro_endereco`, `nro_telefone`, `nro_telefone_aux`, `representante`, `setor_atual`, `sexo`, `tipo`, `ultimo_setor`

| Campo                   | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `bairro`                | text |   ✓    |       ✓       |    ·    |    ·     |
| `cep`                   | text |   ✓    |       ✓       |    ·    |    ·     |
| `cnpj`                  | text |   ✓    |       ✓       |    ·    |    ·     |
| `cpf`                   | text |   ✓    |       ✓       |    ·    |    ·     |
| `data_entrada`          | date |   ✓    |       ✓       |    ·    |    ✓     |
| `data_entrada_original` | date |   ✓    |       ✓       |    ·    |    ✓     |
| `data_nascimento`       | date |   ✓    |       ✓       |    ·    |    ✓     |
| `data_saida`            | date |   ✓    |       ✓       |    ·    |    ✓     |
| `ddd_telefone`          | text |   ✓    |       ✓       |    ·    |    ·     |
| `ddd_telefone_aux`      | text |   ✓    |       ✓       |    ·    |    ·     |
| `desc_cidade`           | text |   ✓    |       ✓       |    ·    |    ·     |
| `empresa_funcionario`   | text |   ✓    |       ✓       |    ·    |    ·     |
| `endereco`              | text |   ✓    |       ✓       |    ·    |    ·     |
| `estado`                | text |   ✓    |       ✓       |    ·    |    ·     |
| `funcionario`           | text |   ✓    |       ✓       |    ·    |    ·     |
| `latitude`              | text |   ✓    |       ✓       |    ·    |    ·     |
| `logradouro`            | text |   ✓    |       ✓       |    ·    |    ·     |
| `longitude`             | text |   ✓    |       ✓       |    ·    |    ·     |
| `nome`                  | text |   ✓    |       ✓       |    ·    |    ·     |
| `nro_endereco`          | text |   ✓    |       ✓       |    ·    |    ·     |
| `nro_telefone`          | text |   ✓    |       ✓       |    ·    |    ·     |
| `nro_telefone_aux`      | text |   ✓    |       ✓       |    ·    |    ·     |
| `representante`         | text |   ✓    |       ✓       |    ·    |    ·     |
| `setor_atual`           | text |   ✓    |       ✓       |    ·    |    ·     |
| `sexo`                  | text |   ✓    |       ✓       |    ·    |    ·     |
| `tipo`                  | text |   ✓    |       ✓       |    ·    |    ·     |
| `ultimo_setor`          | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/representante-info?bairro.keyword=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `cliente-monitorado`

- **Endpoint base:** `GET /v1/cliente-monitorado`
- **Identificador (GET por id):** `id` → `GET /v1/cliente-monitorado/{id}`
- **Busca textual (`q`):** `tipoMonitoramento`

| Campo               | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ------------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `dataInclusao`      | date |   ✓    |       ✓       |    ·    |    ✓     |
| `id`                | long |   ✓    |       ✓       |    ·    |    ·     |
| `idCliente`         | long |   ✓    |       ✓       |    ·    |    ·     |
| `rede`              | long |   ✓    |       ✓       |    ·    |    ·     |
| `setores`           | long |   ✓    |       ✓       |    ·    |    ·     |
| `tipoMonitoramento` | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/cliente-monitorado?dataInclusao=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `cidades`

- **Endpoint base:** `GET /v1/cidades`
- **Identificador (GET por id):** `REVISAR` → `GET /v1/cidades/{id}`
- **Busca textual (`q`):** `descricao`, `estado`

| Campo       | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------- | ---- | :----: | :-----------: | :-----: | :------: |
| `cidade`    | long |   ✓    |       ✓       |    ·    |    ·     |
| `descricao` | text |   ✓    |       ✓       |    ·    |    ·     |
| `estado`    | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/cidades?cidade=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `disparo-mensagem-nines`

- **Endpoint base:** `GET /v1/disparo-mensagem-nines`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/disparo-mensagem-nines/{id}`
- **Busca textual (`q`):** `id`, `idReferencia`, `nroIncricao`, `payload.callback`, `payload.callback_url`, `payload.contato`, `payload.document`, `payload.linhaDigitavel`, `payload.name`, `payload.nome`, `payload.nomeArquivo`, `payload.nroInscricao`, `payload.params.body`, `payload.params.carousel.cards.body`, `payload.params.carousel.cards.buttons.sub_type`, `payload.params.carousel.cards.buttons.value`, `payload.params.carousel.cards.header.type`, `payload.params.carousel.cards.header.value`, `payload.params.header.type`, `payload.params.header.value`, `payload.phone_number`, `payload.telefone`, `payload.tipo`, `payload.urlBoleto`, `status`, `tipo`

| Campo                                            | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ------------------------------------------------ | ---- | :----: | :-----------: | :-----: | :------: |
| `dataDisparo`                                    | date |   ✓    |       ✓       |    ·    |    ✓     |
| `id`                                             | text |   ✓    |       ✓       |    ·    |    ·     |
| `idReferencia`                                   | text |   ✓    |       ✓       |    ·    |    ·     |
| `nroIncricao`                                    | text |   ✓    |       ✓       |    ·    |    ·     |
| `numerosUtilizados`                              | long |   ✓    |       ✓       |    ·    |    ·     |
| `payload.callback`                               | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.callback_url`                           | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.contato`                                | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.document`                               | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.linhaDigitavel`                         | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.name`                                   | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.nome`                                   | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.nomeArquivo`                            | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.nroInscricao`                           | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.params.body`                            | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.params.carousel.cards.body`             | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.params.carousel.cards.buttons.sub_type` | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.params.carousel.cards.buttons.value`    | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.params.carousel.cards.header.type`      | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.params.carousel.cards.header.value`     | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.params.header.type`                     | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.params.header.value`                    | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.phone_number`                           | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.telefone`                               | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.tipo`                                   | text |   ✓    |       ✓       |    ·    |    ·     |
| `payload.urlBoleto`                              | text |   ✓    |       ✓       |    ·    |    ·     |
| `status`                                         | text |   ✓    |       ✓       |    ·    |    ·     |
| `tipo`                                           | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/disparo-mensagem-nines?dataDisparo=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `ponto-digital-s3a`

- **Endpoint base:** `GET /v1/ponto-digital-s3a`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/ponto-digital-s3a/{id}`
- **Busca textual (`q`):** `coordenada`, `id`, `pausas.observacao`, `resumoDiario.observacoesDeMercado`, `resumoDiario.observacoesGerais`, `resumoDiario.solucoes`

| Campo                               | Tipo    | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------------------------------- | ------- | :----: | :-----------: | :-----: | :------: |
| `coordenada`                        | text    |   ✓    |       ✓       |    ·    |    ·     |
| `dataEnvioServidor`                 | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataFimAlmocoMobile`               | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataFimAlmocoServidor`             | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataFimJornadaMobile`              | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataFimJornadaServidor`            | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataHoraRegistro`                  | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataInicioAlmocoMobile`            | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataInicioAlmocoServidor`          | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataInicioJornadaMobile`           | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataInicioJornadaServidor`         | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataSincronizacao`                 | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `entrada`                           | boolean |   ✓    |       ·       |    ·    |    ·     |
| `id`                                | text    |   ✓    |       ✓       |    ·    |    ·     |
| `idSetor`                           | long    |   ✓    |       ✓       |    ·    |    ·     |
| `pausas.dataFimMobile`              | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `pausas.dataFimServidor`            | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `pausas.dataInicioMobile`           | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `pausas.dataInicioServidor`         | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `pausas.observacao`                 | text    |   ✓    |       ✓       |    ·    |    ·     |
| `resumoDiario.observacoesDeMercado` | text    |   ✓    |       ✓       |    ·    |    ·     |
| `resumoDiario.observacoesGerais`    | text    |   ✓    |       ✓       |    ·    |    ·     |
| `resumoDiario.solucoes`             | text    |   ✓    |       ✓       |    ·    |    ·     |
| `tipoPontoS3a`                      | long    |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/ponto-digital-s3a?coordenada.keyword=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `erro-aceite-transportadora`

- **Endpoint base:** `GET /v1/erro-aceite-transportadora`
- **Identificador (GET por id):** `REVISAR` → `GET /v1/erro-aceite-transportadora/{id}`
- **Busca textual (`q`):** `foto`, `qrCode`

| Campo                  | Tipo    | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ---------------------- | ------- | :----: | :-----------: | :-----: | :------: |
| `assinatura_detectada` | boolean |   ✓    |       ·       |    ·    |    ·     |
| `dataErro`             | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `foto`                 | text    |   ✓    |       ✓       |    ·    |    ·     |
| `qrCode`               | text    |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/erro-aceite-transportadora?assinatura_detectada=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `ramo-atividades`

- **Endpoint base:** `GET /v1/ramo-atividades`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/ramo-atividades/{id}`
- **Busca textual (`q`):** `descricao`, `id`

| Campo       | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------- | ---- | :----: | :-----------: | :-----: | :------: |
| `descricao` | text |   ✓    |       ✓       |    ·    |    ·     |
| `id`        | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/ramo-atividades?descricao.keyword=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `solicitacao-recolhimento-ivendas`

- **Endpoint base:** `GET /v1/solicitacao-recolhimento-ivendas`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/solicitacao-recolhimento-ivendas/{id}`
- **Busca textual (`q`):** `descMerc`, `fotos`, `id`, `idItemNfOrigem`, `loteProcessamento`, `solucao`, `tipoRecolhimento`

| Campo                   | Tipo    | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------------------- | ------- | :----: | :-----------: | :-----: | :------: |
| `creditoFinalNegociado` | float   |   ✓    |       ✓       |    ·    |    ·     |
| `dataCriacao`           | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataEmissaoOrigem`     | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataProcessamento`     | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataSolucao`           | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `descMerc`              | text    |   ✓    |       ✓       |    ·    |    ·     |
| `filialOrigem`          | long    |   ✓    |       ✓       |    ·    |    ·     |
| `fotos`                 | text    |   ✓    |       ✓       |    ·    |    ·     |
| `id`                    | text    |   ✓    |       ✓       |    ·    |    ·     |
| `idCliente`             | long    |   ✓    |       ✓       |    ·    |    ·     |
| `idItemNfOrigem`        | text    |   ✓    |       ✓       |    ·    |    ·     |
| `idMercadoria`          | long    |   ✓    |       ✓       |    ·    |    ·     |
| `idSetor`               | long    |   ✓    |       ✓       |    ·    |    ·     |
| `itemExclusivo`         | boolean |   ✓    |       ·       |    ·    |    ·     |
| `loteProcessamento`     | text    |   ✓    |       ✓       |    ·    |    ·     |
| `motivo`                | long    |   ✓    |       ✓       |    ·    |    ·     |
| `nroNotaOrigem`         | long    |   ✓    |       ✓       |    ·    |    ·     |
| `previsaoCredito`       | float   |   ✓    |       ✓       |    ·    |    ·     |
| `qtdFinalNegociada`     | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `qtdSolicitada`         | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `solucao`               | text    |   ✓    |       ✓       |    ·    |    ·     |
| `status`                | long    |   ✓    |       ✓       |    ·    |    ·     |
| `tipoRecolhimento`      | text    |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/solicitacao-recolhimento-ivendas?creditoFinalNegociado=VALOR&limite=20

# agregacao: soma de qtdFinalNegociada por descMerc.keyword
GET /v1/solicitacao-recolhimento-ivendas/agregado?por=descMerc.keyword&metrica=sum:qtdFinalNegociada&top=10

# serie temporal: soma de qtdFinalNegociada por mes
GET /v1/solicitacao-recolhimento-ivendas/agregado?por=dataCriacao:mes&metrica=sum:qtdFinalNegociada&dataCriacao.gte=2025-01-01&dataCriacao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `extrato-representante-ivendas`

- **Endpoint base:** `GET /v1/extrato-representante-ivendas`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/extrato-representante-ivendas/{id}`
- **Busca textual (`q`):** `descricaoLancto`, `id`, `validoElasticsearch`

| Campo                 | Tipo  | Filtro | Ordena/Agrupa | Métrica | Temporal |
| --------------------- | ----- | :----: | :-----------: | :-----: | :------: |
| `dataAlteracao`       | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `dataExtrato`         | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `dataReferencia`      | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `dataValidade`        | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `descricaoLancto`     | text  |   ✓    |       ✓       |    ·    |    ·     |
| `id`                  | text  |   ✓    |       ✓       |    ·    |    ·     |
| `idEmpresa`           | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idRepresentante`     | long  |   ✓    |       ✓       |    ·    |    ·     |
| `seqExtrato`          | long  |   ✓    |       ✓       |    ·    |    ·     |
| `validoElasticsearch` | text  |   ✓    |       ✓       |    ·    |    ·     |
| `vlrLancto`           | float |   ✓    |       ✓       |    ✓    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/extrato-representante-ivendas?dataAlteracao=VALOR&limite=20

# agregacao: soma de vlrLancto por descricaoLancto.keyword
GET /v1/extrato-representante-ivendas/agregado?por=descricaoLancto.keyword&metrica=sum:vlrLancto&top=10

# serie temporal: soma de vlrLancto por mes
GET /v1/extrato-representante-ivendas/agregado?por=dataAlteracao:mes&metrica=sum:vlrLancto&dataAlteracao.gte=2025-01-01&dataAlteracao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `debitos-cobranca-v2`

- **Endpoint base:** `GET /v1/debitos-cobranca-v2`
- **Identificador (GET por id):** `id` → `GET /v1/debitos-cobranca-v2/{id}`

| Campo                       | Tipo         | Filtro | Ordena/Agrupa | Métrica | Temporal |
| --------------------------- | ------------ | :----: | :-----------: | :-----: | :------: |
| `cnpj`                      | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `dataAlteracao`             | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `dataVencto`                | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `exibeChatBot`              | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `id`                        | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `idCliente`                 | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idEmpresaContabil`         | short        |   ✓    |       ✓       |    ·    |    ·     |
| `notasFiscais.dataEmissao`  | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `notasFiscais.idFilial`     | short        |   ✓    |       ✓       |    ·    |    ·     |
| `notasFiscais.idNotaFiscal` | long         |   ✓    |       ✓       |    ·    |    ·     |
| `notasFiscais.idSeqDebito`  | long         |   ✓    |       ✓       |    ·    |    ·     |
| `notasFiscais.vlrDebito`    | scaled_float |   ✓    |       ✓       |    ✓    |    ·     |
| `notificaCliente`           | boolean      |   ✓    |       ·       |    ·    |    ·     |
| `vlrDebito`                 | scaled_float |   ✓    |       ✓       |    ✓    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/debitos-cobranca-v2?cnpj=VALOR&limite=20

# agregacao: soma de notasFiscais.vlrDebito por cnpj
GET /v1/debitos-cobranca-v2/agregado?por=cnpj&metrica=sum:notasFiscais.vlrDebito&top=10

# serie temporal: soma de notasFiscais.vlrDebito por mes
GET /v1/debitos-cobranca-v2/agregado?por=dataAlteracao:mes&metrica=sum:notasFiscais.vlrDebito&dataAlteracao.gte=2025-01-01&dataAlteracao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `mercadorias`

- **Endpoint base:** `GET /v1/mercadorias`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/mercadorias/{id}`
- **Busca textual (`q`):** `descricao`, `id`, `unidadeVenda`

| Campo          | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| -------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `codigoFoto`   | long |   ✓    |       ✓       |    ·    |    ·     |
| `descricao`    | text |   ✓    |       ✓       |    ·    |    ·     |
| `id`           | text |   ✓    |       ✓       |    ·    |    ·     |
| `situacao`     | long |   ✓    |       ✓       |    ·    |    ·     |
| `unidadeVenda` | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/mercadorias?codigoFoto=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `saldo-vendedor-promox`

- **Endpoint base:** `GET /v1/saldo-vendedor-promox`
- **Identificador (GET por id):** `id` → `GET /v1/saldo-vendedor-promox/{id}`

| Campo                   | Tipo  | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------------------- | ----- | :----: | :-----------: | :-----: | :------: |
| `dataConsulta`          | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `dataPagamento`         | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `dataPagtoBrinde`       | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `id`                    | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idPromocao`            | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idPromocaoRestricao`   | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idRepresentante`       | long  |   ✓    |       ✓       |    ·    |    ·     |
| `qtdPontos`             | long  |   ✓    |       ✓       |    ✓    |    ·     |
| `qtdTotalPromocao`      | long  |   ✓    |       ✓       |    ✓    |    ·     |
| `qtdUnidadePromocao`    | short |   ✓    |       ✓       |    ✓    |    ·     |
| `valorAtingido`         | float |   ✓    |       ✓       |    ✓    |    ·     |
| `valorBonusExtra`       | float |   ✓    |       ✓       |    ✓    |    ·     |
| `valorDesafioVenda`     | float |   ✓    |       ✓       |    ✓    |    ·     |
| `valorFaixa`            | float |   ✓    |       ✓       |    ✓    |    ·     |
| `valorPremioDinheiro`   | float |   ✓    |       ✓       |    ✓    |    ·     |
| `valorVendaGeral`       | float |   ✓    |       ✓       |    ✓    |    ·     |
| `valorVendaPeriodoBase` | float |   ✓    |       ✓       |    ✓    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/saldo-vendedor-promox?dataConsulta=VALOR&limite=20

# serie temporal: soma de qtdPontos por mes
GET /v1/saldo-vendedor-promox/agregado?por=dataConsulta:mes&metrica=sum:qtdPontos&dataConsulta.gte=2025-01-01&dataConsulta.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `ocor-clientes`

- **Endpoint base:** `GET /v1/ocor-clientes`
- **Identificador (GET por id):** `REVISAR` → `GET /v1/ocor-clientes/{id}`
- **Busca textual (`q`):** `ativa`, `automatico`, `bloqueia_pedido`, `bloqueio_rede`, `desbloq_automatico`, `descricao`, `desmarca_antecipado`, `exclui_cliente`, `libera_uso_palm`, `lista_cliente`, `marca_antecipado`, `permite_antecipado`, `relacionar_debitos`, `situacao_pedido`, `situacao_reimplantacao`, `visualiza_cba`, `visualiza_mobile`

| Campo                    | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ------------------------ | ---- | :----: | :-----------: | :-----: | :------: |
| `ativa`                  | text |   ✓    |       ✓       |    ·    |    ·     |
| `automatico`             | text |   ✓    |       ✓       |    ·    |    ·     |
| `bloqueia_pedido`        | text |   ✓    |       ✓       |    ·    |    ·     |
| `bloqueio_rede`          | text |   ✓    |       ✓       |    ·    |    ·     |
| `desbloq_automatico`     | text |   ✓    |       ✓       |    ·    |    ·     |
| `descricao`              | text |   ✓    |       ✓       |    ·    |    ·     |
| `desmarca_antecipado`    | text |   ✓    |       ✓       |    ·    |    ·     |
| `exclui_cliente`         | text |   ✓    |       ✓       |    ·    |    ·     |
| `libera_uso_palm`        | text |   ✓    |       ✓       |    ·    |    ·     |
| `lista_cliente`          | text |   ✓    |       ✓       |    ·    |    ·     |
| `marca_antecipado`       | text |   ✓    |       ✓       |    ·    |    ·     |
| `nivel_hierarquico`      | long |   ✓    |       ✓       |    ·    |    ·     |
| `ocor_bloqueio`          | long |   ✓    |       ✓       |    ·    |    ·     |
| `ocorrencia`             | long |   ✓    |       ✓       |    ·    |    ·     |
| `permite_antecipado`     | text |   ✓    |       ✓       |    ·    |    ·     |
| `relacionar_debitos`     | text |   ✓    |       ✓       |    ·    |    ·     |
| `situacao_pedido`        | text |   ✓    |       ✓       |    ·    |    ·     |
| `situacao_reimplantacao` | text |   ✓    |       ✓       |    ·    |    ·     |
| `soluciona_solicitacao`  | long |   ✓    |       ✓       |    ·    |    ·     |
| `tempo_exclusao`         | long |   ✓    |       ✓       |    ·    |    ·     |
| `tipo`                   | long |   ✓    |       ✓       |    ·    |    ·     |
| `visualiza_cba`          | text |   ✓    |       ✓       |    ·    |    ·     |
| `visualiza_mobile`       | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/ocor-clientes?ativa.keyword=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `despesa-s3a`

- **Endpoint base:** `GET /v1/despesa-s3a`
- **Identificador (GET por id):** `idSetor` → `GET /v1/despesa-s3a/{id}`
- **Busca textual (`q`):** `descricao`, `foto`, `numeroNota`

| Campo         | Tipo  | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ------------- | ----- | :----: | :-----------: | :-----: | :------: |
| `dataCriacao` | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `descricao`   | text  |   ✓    |       ✓       |    ·    |    ·     |
| `foto`        | text  |   ✓    |       ✓       |    ·    |    ·     |
| `idSetor`     | long  |   ✓    |       ✓       |    ·    |    ·     |
| `numeroNota`  | text  |   ✓    |       ✓       |    ·    |    ·     |
| `valor`       | float |   ✓    |       ✓       |    ✓    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/despesa-s3a?dataCriacao=VALOR&limite=20

# agregacao: soma de valor por descricao.keyword
GET /v1/despesa-s3a/agregado?por=descricao.keyword&metrica=sum:valor&top=10

# serie temporal: soma de valor por mes
GET /v1/despesa-s3a/agregado?por=dataCriacao:mes&metrica=sum:valor&dataCriacao.gte=2025-01-01&dataCriacao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `equipe-vendas-setor`

- **Endpoint base:** `GET /v1/equipe-vendas-setor`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/equipe-vendas-setor/{id}`
- **Busca textual (`q`):** `contasEspeciais`, `descCidadePrincipal`, `descRegCoord`, `descRegVda`, `descricaoSegmento`, `emailCoord`, `emailGN`, `emailGV`, `id`, `nomeCoord`, `nomeGN`, `nomeGV`, `nomeRepres`, `tipoRepres`, `tipoSetor`, `ufCidadePrincipal`

| Campo                   | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `codCidadePrincipal`    | long |   ✓    |       ✓       |    ·    |    ·     |
| `codCoord`              | long |   ✓    |       ✓       |    ·    |    ·     |
| `codDistritoPrincipal`  | long |   ✓    |       ✓       |    ·    |    ·     |
| `codGV`                 | long |   ✓    |       ✓       |    ·    |    ·     |
| `codRegCoord`           | long |   ✓    |       ✓       |    ·    |    ·     |
| `codRegVda`             | long |   ✓    |       ✓       |    ·    |    ·     |
| `contasEspeciais`       | text |   ✓    |       ✓       |    ·    |    ·     |
| `dataAtualizacao`       | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataEntradaOrigRepres` | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataEntradaRepres`     | date |   ✓    |       ✓       |    ·    |    ✓     |
| `descCidadePrincipal`   | text |   ✓    |       ✓       |    ·    |    ·     |
| `descRegCoord`          | text |   ✓    |       ✓       |    ·    |    ·     |
| `descRegVda`            | text |   ✓    |       ✓       |    ·    |    ·     |
| `descricaoSegmento`     | text |   ✓    |       ✓       |    ·    |    ·     |
| `emailCoord`            | text |   ✓    |       ✓       |    ·    |    ·     |
| `emailGN`               | text |   ✓    |       ✓       |    ·    |    ·     |
| `emailGV`               | text |   ✓    |       ✓       |    ·    |    ·     |
| `empresa`               | long |   ✓    |       ✓       |    ·    |    ·     |
| `id`                    | text |   ✓    |       ✓       |    ·    |    ·     |
| `nomeCoord`             | text |   ✓    |       ✓       |    ·    |    ·     |
| `nomeGN`                | text |   ✓    |       ✓       |    ·    |    ·     |
| `nomeGV`                | text |   ✓    |       ✓       |    ·    |    ·     |
| `nomeRepres`            | text |   ✓    |       ✓       |    ·    |    ·     |
| `representante`         | long |   ✓    |       ✓       |    ·    |    ·     |
| `repsetDataInicio`      | date |   ✓    |       ✓       |    ·    |    ✓     |
| `segmentoSetor`         | long |   ✓    |       ✓       |    ·    |    ·     |
| `setor`                 | long |   ✓    |       ✓       |    ·    |    ·     |
| `setorDataInicio`       | date |   ✓    |       ✓       |    ·    |    ✓     |
| `tipoRepres`            | text |   ✓    |       ✓       |    ·    |    ·     |
| `tipoSetor`             | text |   ✓    |       ✓       |    ·    |    ·     |
| `ufCidadePrincipal`     | text |   ✓    |       ✓       |    ·    |    ·     |
| `usuarioGN`             | long |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/equipe-vendas-setor?codCidadePrincipal=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `roteiro`

- **Endpoint base:** `GET /v1/roteiro`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/roteiro/{id}`
- **Busca textual (`q`):** `descCidade`, `erro`, `id`, `observacao`, `status`, `ufCidade`

| Campo             | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `dataFim`         | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataInicio`      | date |   ✓    |       ✓       |    ·    |    ✓     |
| `descCidade`      | text |   ✓    |       ✓       |    ·    |    ·     |
| `erro`            | text |   ✓    |       ✓       |    ·    |    ·     |
| `id`              | text |   ✓    |       ✓       |    ·    |    ·     |
| `idEmpresa`       | long |   ✓    |       ✓       |    ·    |    ·     |
| `idRepresentante` | long |   ✓    |       ✓       |    ·    |    ·     |
| `idSetor`         | long |   ✓    |       ✓       |    ·    |    ·     |
| `idUsuario`       | long |   ✓    |       ✓       |    ·    |    ·     |
| `observacao`      | text |   ✓    |       ✓       |    ·    |    ·     |
| `status`          | text |   ✓    |       ✓       |    ·    |    ·     |
| `tipoRoteiro`     | long |   ✓    |       ✓       |    ·    |    ·     |
| `ufCidade`        | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/roteiro?dataFim=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `pagto-dinheiro-promox`

- **Endpoint base:** `GET /v1/pagto-dinheiro-promox`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/pagto-dinheiro-promox/{id}`
- **Busca textual (`q`):** `id`

| Campo             | Tipo  | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------------- | ----- | :----: | :-----------: | :-----: | :------: |
| `dataApuracao`    | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `dataConsulta`    | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `dataPagamento`   | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `id`              | text  |   ✓    |       ✓       |    ·    |    ·     |
| `idEmpresa`       | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idPromocao`      | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idRepresentante` | long  |   ✓    |       ✓       |    ·    |    ·     |
| `vlrPagamento`    | float |   ✓    |       ✓       |    ✓    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/pagto-dinheiro-promox?dataApuracao=VALOR&limite=20

# agregacao: soma de vlrPagamento por id.keyword
GET /v1/pagto-dinheiro-promox/agregado?por=id.keyword&metrica=sum:vlrPagamento&top=10

# serie temporal: soma de vlrPagamento por mes
GET /v1/pagto-dinheiro-promox/agregado?por=dataApuracao:mes&metrica=sum:vlrPagamento&dataApuracao.gte=2025-01-01&dataApuracao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `log-visita-ivendas`

- **Endpoint base:** `GET /v1/log-visita-ivendas`
- **Identificador (GET por id):** `REVISAR` → `GET /v1/log-visita-ivendas/{id}`
- **Busca textual (`q`):** `coordenadaCliente`, `coordenadaSetor`, `evento`, `nroInscricao`, `observacao`

| Campo               | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ------------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `coordenadaCliente` | text |   ✓    |       ✓       |    ·    |    ·     |
| `coordenadaSetor`   | text |   ✓    |       ✓       |    ·    |    ·     |
| `dataEvento`        | date |   ✓    |       ✓       |    ·    |    ✓     |
| `distancia`         | long |   ✓    |       ✓       |    ✓    |    ·     |
| `evento`            | text |   ✓    |       ✓       |    ·    |    ·     |
| `nroInscricao`      | text |   ✓    |       ✓       |    ·    |    ·     |
| `observacao`        | text |   ✓    |       ✓       |    ·    |    ·     |
| `setor`             | long |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/log-visita-ivendas?coordenadaCliente.keyword=VALOR&limite=20

# agregacao: soma de distancia por coordenadaCliente.keyword
GET /v1/log-visita-ivendas/agregado?por=coordenadaCliente.keyword&metrica=sum:distancia&top=10

# serie temporal: soma de distancia por mes
GET /v1/log-visita-ivendas/agregado?por=dataEvento:mes&metrica=sum:distancia&dataEvento.gte=2025-01-01&dataEvento.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `cliente-ivendas-v2`

- **Endpoint base:** `GET /v1/cliente-ivendas-v2`
- **Identificador (GET por id):** `id` → `GET /v1/cliente-ivendas-v2/{id}`
- **Busca textual (`q`):** `coordenadaNew`, `descricaoCidadeSemDistrito`, `descricaoOcorCadastro`, `descricaoOcorCredito`

| Campo                        | Tipo         | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ---------------------------- | ------------ | :----: | :-----------: | :-----: | :------: |
| `bairro`                     | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `cep`                        | long         |   ✓    |       ✓       |    ·    |    ·     |
| `complemento`                | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `coordenada`                 | geo_point    |   ·    |       ·       |    ·    |    ·     |
| `coordenadaNew`              | text         |   ✓    |       ✓       |    ·    |    ·     |
| `dataAlteracao`              | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `dataImplantacao`            | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `dataUltimoPedido`           | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `descricaoCidade`            | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `descricaoCidadeSemDistrito` | text         |   ✓    |       ✓       |    ·    |    ·     |
| `descricaoOcorCadastro`      | text         |   ✓    |       ✓       |    ·    |    ·     |
| `descricaoOcorCredito`       | text         |   ✓    |       ✓       |    ·    |    ·     |
| `email`                      | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `endereco`                   | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `estado`                     | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `fantasia`                   | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `id`                         | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `idCidade`                   | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idCliente`                  | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idOcorCadastro`             | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idOcorCredito`              | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idRamoAtividade`            | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idSetor`                    | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idZonaRoteirizacao`         | long         |   ✓    |       ✓       |    ·    |    ·     |
| `logradouro`                 | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `nroEndereco`                | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `origemImplantacao`          | short        |   ✓    |       ✓       |    ·    |    ·     |
| `razaoSocial`                | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `recurso.dataAtualizacao`    | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `recurso.saldo`              | scaled_float |   ✓    |       ✓       |    ✓    |    ·     |
| `rede`                       | long         |   ✓    |       ✓       |    ·    |    ·     |
| `responsavelCobranca.id`     | long         |   ✓    |       ✓       |    ·    |    ·     |
| `responsavelCobranca.nome`   | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `setores`                    | long         |   ✓    |       ✓       |    ·    |    ·     |
| `tipoPessoa`                 | short        |   ✓    |       ✓       |    ·    |    ·     |
| `vlrSaldoCashback`           | long         |   ✓    |       ✓       |    ✓    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/cliente-ivendas-v2?bairro=VALOR&limite=20

# agregacao: soma de recurso.saldo por bairro
GET /v1/cliente-ivendas-v2/agregado?por=bairro&metrica=sum:recurso.saldo&top=10

# serie temporal: soma de recurso.saldo por mes
GET /v1/cliente-ivendas-v2/agregado?por=dataAlteracao:mes&metrica=sum:recurso.saldo&dataAlteracao.gte=2025-01-01&dataAlteracao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `mercadoria-brindes-pontuacao-promox`

- **Endpoint base:** `GET /v1/mercadoria-brindes-pontuacao-promox`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/mercadoria-brindes-pontuacao-promox/{id}`
- **Busca textual (`q`):** `descricaoCompleta`, `id`

| Campo               | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ------------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `codigoFoto`        | long |   ✓    |       ✓       |    ·    |    ·     |
| `dataAtualizacao`   | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataConsulta`      | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataExpiracao`     | date |   ✓    |       ✓       |    ·    |    ✓     |
| `descricaoCompleta` | text |   ✓    |       ✓       |    ·    |    ·     |
| `id`                | text |   ✓    |       ✓       |    ·    |    ·     |
| `idEmpresa`         | long |   ✓    |       ✓       |    ·    |    ·     |
| `idMercadoria`      | long |   ✓    |       ✓       |    ·    |    ·     |
| `idPromocao`        | long |   ✓    |       ✓       |    ·    |    ·     |
| `qtdePontos`        | long |   ✓    |       ✓       |    ✓    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/mercadoria-brindes-pontuacao-promox?codigoFoto=VALOR&limite=20

# agregacao: soma de qtdePontos por descricaoCompleta.keyword
GET /v1/mercadoria-brindes-pontuacao-promox/agregado?por=descricaoCompleta.keyword&metrica=sum:qtdePontos&top=10

# serie temporal: soma de qtdePontos por mes
GET /v1/mercadoria-brindes-pontuacao-promox/agregado?por=dataAtualizacao:mes&metrica=sum:qtdePontos&dataAtualizacao.gte=2025-01-01&dataAtualizacao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `gratificacao_motorista`

- **Endpoint base:** `GET /v1/gratificacao_motorista`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/gratificacao_motorista/{id}`
- **Busca textual (`q`):** `descricaoCdm`, `emailGerente`, `id`, `nome`, `nomeGerente`

| Campo                                | Tipo    | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ------------------------------------ | ------- | :----: | :-----------: | :-----: | :------: |
| `atingiuPontualidade`                | boolean |   ✓    |       ·       |    ·    |    ·     |
| `dataCompetencia`                    | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataFimCompetencia`                 | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataInicioCompetencia`              | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `descricaoCdm`                       | text    |   ✓    |       ✓       |    ·    |    ·     |
| `emailGerente`                       | text    |   ✓    |       ✓       |    ·    |    ·     |
| `entregaPontual`                     | long    |   ✓    |       ✓       |    ·    |    ·     |
| `id`                                 | text    |   ✓    |       ✓       |    ·    |    ·     |
| `idMotorista`                        | long    |   ✓    |       ✓       |    ·    |    ·     |
| `kmEsperadoTotal`                    | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `kmEsperadoTotalComMargemTolerancia` | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `kmRodadoTotal`                      | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `mediaAtingida`                      | float   |   ✓    |       ✓       |    ·    |    ·     |
| `mediaEsperada`                      | float   |   ✓    |       ✓       |    ·    |    ·     |
| `nome`                               | text    |   ✓    |       ✓       |    ·    |    ·     |
| `nomeGerente`                        | text    |   ✓    |       ✓       |    ·    |    ·     |
| `pcrCdaMotorista`                    | float   |   ✓    |       ✓       |    ·    |    ·     |
| `pcrEntregaPontual`                  | float   |   ✓    |       ✓       |    ·    |    ·     |
| `pctEntregaPontual`                  | float   |   ✓    |       ✓       |    ·    |    ·     |
| `percentualAtingidoKmRodado`         | float   |   ✓    |       ✓       |    ✓    |    ·     |
| `percentualMetaConsumo`              | float   |   ✓    |       ✓       |    ✓    |    ·     |
| `percentualPremioKm`                 | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `percentualPremioMediaConsumo`       | float   |   ✓    |       ✓       |    ✓    |    ·     |
| `premioBonificacaoExtra`             | float   |   ✓    |       ✓       |    ·    |    ·     |
| `premioKm`                           | float   |   ✓    |       ✓       |    ✓    |    ·     |
| `premioMedia`                        | float   |   ✓    |       ✓       |    ·    |    ·     |
| `qtdTotalEntrega`                    | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `teveAcidente`                       | boolean |   ✓    |       ·       |    ·    |    ·     |
| `teveExcessoVelocidade`              | boolean |   ✓    |       ·       |    ·    |    ·     |
| `teveMulta`                          | boolean |   ✓    |       ·       |    ·    |    ·     |
| `valorBase`                          | float   |   ✓    |       ✓       |    ✓    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/gratificacao_motorista?atingiuPontualidade=VALOR&limite=20

# agregacao: soma de kmEsperadoTotal por descricaoCdm.keyword
GET /v1/gratificacao_motorista/agregado?por=descricaoCdm.keyword&metrica=sum:kmEsperadoTotal&top=10

# serie temporal: soma de kmEsperadoTotal por mes
GET /v1/gratificacao_motorista/agregado?por=dataCompetencia:mes&metrica=sum:kmEsperadoTotal&dataCompetencia.gte=2025-01-01&dataCompetencia.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `mercadoria-promox-v3`

- **Endpoint base:** `GET /v1/mercadoria-promox-v3`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/mercadoria-promox-v3/{id}`
- **Busca textual (`q`):** `descricaoDivisao`, `descricaoMercadoria`, `id`, `idDivisao`, `imagemDivisao`, `regioesVendaEstado.estado`, `validoElasticsearch`

| Campo                              | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ---------------------------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `codigoFoto`                       | long |   ✓    |       ✓       |    ·    |    ·     |
| `dataConsulta`                     | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataFim`                          | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataInicio`                       | date |   ✓    |       ✓       |    ·    |    ✓     |
| `descricaoDivisao`                 | text |   ✓    |       ✓       |    ·    |    ·     |
| `descricaoMercadoria`              | text |   ✓    |       ✓       |    ·    |    ·     |
| `id`                               | text |   ✓    |       ✓       |    ·    |    ·     |
| `idDivisao`                        | text |   ✓    |       ✓       |    ·    |    ·     |
| `idEmpresa`                        | long |   ✓    |       ✓       |    ·    |    ·     |
| `idMercadoria`                     | long |   ✓    |       ✓       |    ·    |    ·     |
| `idPromocao`                       | long |   ✓    |       ✓       |    ·    |    ·     |
| `idsRegiaoVendas`                  | long |   ✓    |       ✓       |    ·    |    ·     |
| `idsSegmentos`                     | long |   ✓    |       ✓       |    ·    |    ·     |
| `imagemDivisao`                    | text |   ✓    |       ✓       |    ·    |    ·     |
| `leveQtdMercadoria`                | long |   ✓    |       ✓       |    ✓    |    ·     |
| `modeloPromocao`                   | long |   ✓    |       ✓       |    ·    |    ·     |
| `pagueQtdMercadoria`               | long |   ✓    |       ✓       |    ✓    |    ·     |
| `regioesVendaEstado.estado`        | text |   ✓    |       ✓       |    ·    |    ·     |
| `regioesVendaEstado.idRegiaoVenda` | long |   ✓    |       ✓       |    ·    |    ·     |
| `relatorio`                        | long |   ✓    |       ✓       |    ·    |    ·     |
| `tipoPromocao`                     | long |   ✓    |       ✓       |    ·    |    ·     |
| `validoElasticsearch`              | text |   ✓    |       ✓       |    ·    |    ·     |
| `valor`                            | long |   ✓    |       ✓       |    ✓    |    ·     |
| `versaoFoto`                       | long |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/mercadoria-promox-v3?codigoFoto=VALOR&limite=20

# agregacao: soma de leveQtdMercadoria por descricaoDivisao.keyword
GET /v1/mercadoria-promox-v3/agregado?por=descricaoDivisao.keyword&metrica=sum:leveQtdMercadoria&top=10

# serie temporal: soma de leveQtdMercadoria por mes
GET /v1/mercadoria-promox-v3/agregado?por=dataConsulta:mes&metrica=sum:leveQtdMercadoria&dataConsulta.gte=2025-01-01&dataConsulta.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `roteiro-ivendas`

- **Endpoint base:** `GET /v1/roteiro-ivendas`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/roteiro-ivendas/{id}`
- **Busca textual (`q`):** `id`, `paradas.tipoCriacao`, `tipoRoteiro`

| Campo                          | Tipo  | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ------------------------------ | ----- | :----: | :-----------: | :-----: | :------: |
| `dataCriacao`                  | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `dataFim`                      | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `dataInicio`                   | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `distanciaPercorridaEmMetros`  | long  |   ✓    |       ✓       |    ✓    |    ·     |
| `id`                           | text  |   ✓    |       ✓       |    ·    |    ·     |
| `idRepresentante`              | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idRepresentanteCriador`       | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idSetor`                      | long  |   ✓    |       ✓       |    ·    |    ·     |
| `latitudeOrigem`               | float |   ✓    |       ✓       |    ·    |    ·     |
| `longitudeOrigem`              | float |   ✓    |       ✓       |    ·    |    ·     |
| `paradas.cliente`              | long  |   ✓    |       ✓       |    ·    |    ·     |
| `paradas.dataAtualizacao`      | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `paradas.dataCriacao`          | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `paradas.dataSaida`            | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `paradas.dataVisitaIVendas`    | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `paradas.latitude`             | float |   ✓    |       ✓       |    ·    |    ·     |
| `paradas.latitudeSaida`        | float |   ✓    |       ✓       |    ·    |    ·     |
| `paradas.longitude`            | float |   ✓    |       ✓       |    ·    |    ·     |
| `paradas.longitudeSaida`       | float |   ✓    |       ✓       |    ·    |    ·     |
| `paradas.matriculaCriacao`     | long  |   ✓    |       ✓       |    ·    |    ·     |
| `paradas.origemInclusao`       | long  |   ✓    |       ✓       |    ·    |    ·     |
| `paradas.tipoCriacao`          | text  |   ✓    |       ✓       |    ·    |    ·     |
| `tipoRoteiro`                  | text  |   ✓    |       ✓       |    ·    |    ·     |
| `visitasFalhas.cliente`        | long  |   ✓    |       ✓       |    ·    |    ·     |
| `visitasFalhas.dataCriacao`    | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `visitasFalhas.latitudeAtual`  | float |   ✓    |       ✓       |    ·    |    ·     |
| `visitasFalhas.longitudeAtual` | float |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/roteiro-ivendas?dataCriacao=VALOR&limite=20

# agregacao: soma de distanciaPercorridaEmMetros por id.keyword
GET /v1/roteiro-ivendas/agregado?por=id.keyword&metrica=sum:distanciaPercorridaEmMetros&top=10

# serie temporal: soma de distanciaPercorridaEmMetros por mes
GET /v1/roteiro-ivendas/agregado?por=dataCriacao:mes&metrica=sum:distanciaPercorridaEmMetros&dataCriacao.gte=2025-01-01&dataCriacao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `brinde-vendedor-promox`

- **Endpoint base:** `GET /v1/brinde-vendedor-promox`
- **Identificador (GET por id):** `id` → `GET /v1/brinde-vendedor-promox/{id}`
- **Busca textual (`q`):** `brindeValido`, `descricaoMercadoria`

| Campo                  | Tipo  | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ---------------------- | ----- | :----: | :-----------: | :-----: | :------: |
| `brindeValido`         | text  |   ✓    |       ✓       |    ·    |    ·     |
| `codigoFotoMercadoria` | long  |   ✓    |       ✓       |    ·    |    ·     |
| `custoMercadoria`      | float |   ✓    |       ✓       |    ✓    |    ·     |
| `dataAlteracao`        | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `dataConsulta`         | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `dataEmissao`          | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `descricaoMercadoria`  | text  |   ✓    |       ✓       |    ·    |    ·     |
| `id`                   | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idEmpresa`            | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idFilial`             | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idMercadoria`         | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idNfSaida`            | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idPedido`             | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idPromocao`           | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idRepresentante`      | long  |   ✓    |       ✓       |    ·    |    ·     |
| `quantidade`           | long  |   ✓    |       ✓       |    ·    |    ·     |
| `sequencia`            | long  |   ✓    |       ✓       |    ·    |    ·     |
| `versaoFotoMercadoria` | long  |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/brinde-vendedor-promox?brindeValido.keyword=VALOR&limite=20

# agregacao: soma de custoMercadoria por brindeValido.keyword
GET /v1/brinde-vendedor-promox/agregado?por=brindeValido.keyword&metrica=sum:custoMercadoria&top=10

# serie temporal: soma de custoMercadoria por mes
GET /v1/brinde-vendedor-promox/agregado?por=dataAlteracao:mes&metrica=sum:custoMercadoria&dataAlteracao.gte=2025-01-01&dataAlteracao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `setor_rca_detalhes`

- **Endpoint base:** `GET /v1/setor_rca_detalhes`
- **Identificador (GET por id):** `REVISAR` → `GET /v1/setor_rca_detalhes/{id}`
- **Busca textual (`q`):** `cidade`, `estado`, `nome`, `tipo_usuario`

| Campo                   | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `cidade`                | text |   ✓    |       ✓       |    ·    |    ·     |
| `clientes.ativos`       | long |   ✓    |       ✓       |    ·    |    ·     |
| `clientes.bloqueados`   | long |   ✓    |       ✓       |    ·    |    ·     |
| `clientes.inativos`     | long |   ✓    |       ✓       |    ·    |    ·     |
| `codigo`                | long |   ✓    |       ✓       |    ·    |    ·     |
| `data_entrada`          | date |   ✓    |       ✓       |    ·    |    ✓     |
| `estado`                | text |   ✓    |       ✓       |    ·    |    ·     |
| `nome`                  | text |   ✓    |       ✓       |    ·    |    ·     |
| `regiao_venda`          | long |   ✓    |       ✓       |    ·    |    ·     |
| `segmento`              | long |   ✓    |       ✓       |    ·    |    ·     |
| `setor`                 | long |   ✓    |       ✓       |    ·    |    ·     |
| `tipo_usuario`          | text |   ✓    |       ✓       |    ·    |    ·     |
| `ultimo_acompanhamento` | date |   ✓    |       ✓       |    ·    |    ✓     |

**Exemplos:**

```
# listagem
GET /v1/setor_rca_detalhes?cidade.keyword=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `detalhes-roteiro`

- **Endpoint base:** `GET /v1/detalhes-roteiro`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/detalhes-roteiro/{id}`
- **Busca textual (`q`):** `descRegiaoVenda`, `descTipoRegiao`, `dsCidade`, `ehRespNac`, `id`, `nome`, `nomeMacro`, `nomeVenda`, `roteiros.observacao`, `roteiros.tipoEnvio`, `tipoGerencia`, `uf`

| Campo                      | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| -------------------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `cidade`                   | long |   ✓    |       ✓       |    ·    |    ·     |
| `dataAtualizacao`          | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataGozoFim`              | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataGozoInicio`           | date |   ✓    |       ✓       |    ·    |    ✓     |
| `descRegiaoVenda`          | text |   ✓    |       ✓       |    ·    |    ·     |
| `descTipoRegiao`           | text |   ✓    |       ✓       |    ·    |    ·     |
| `distrito`                 | long |   ✓    |       ✓       |    ·    |    ·     |
| `dsCidade`                 | text |   ✓    |       ✓       |    ·    |    ·     |
| `ehRespNac`                | text |   ✓    |       ✓       |    ·    |    ·     |
| `fimAfast`                 | date |   ✓    |       ✓       |    ·    |    ✓     |
| `funcionario`              | long |   ✓    |       ✓       |    ·    |    ·     |
| `gerente`                  | long |   ✓    |       ✓       |    ·    |    ·     |
| `id`                       | text |   ✓    |       ✓       |    ·    |    ·     |
| `inicioAfast`              | date |   ✓    |       ✓       |    ·    |    ✓     |
| `nome`                     | text |   ✓    |       ✓       |    ·    |    ·     |
| `nomeMacro`                | text |   ✓    |       ✓       |    ·    |    ·     |
| `nomeVenda`                | text |   ✓    |       ✓       |    ·    |    ·     |
| `regiaoMacro`              | long |   ✓    |       ✓       |    ·    |    ·     |
| `regiaoVenda`              | long |   ✓    |       ✓       |    ·    |    ·     |
| `roteiros.dataCadastro`    | date |   ✓    |       ✓       |    ·    |    ✓     |
| `roteiros.dataFim`         | date |   ✓    |       ✓       |    ·    |    ✓     |
| `roteiros.dataInicio`      | date |   ✓    |       ✓       |    ·    |    ✓     |
| `roteiros.dataProcesso`    | date |   ✓    |       ✓       |    ·    |    ✓     |
| `roteiros.idEmpresa`       | long |   ✓    |       ✓       |    ·    |    ·     |
| `roteiros.idRepresentante` | long |   ✓    |       ✓       |    ·    |    ·     |
| `roteiros.idSetor`         | long |   ✓    |       ✓       |    ·    |    ·     |
| `roteiros.idUsuario`       | long |   ✓    |       ✓       |    ·    |    ·     |
| `roteiros.observacao`      | text |   ✓    |       ✓       |    ·    |    ·     |
| `roteiros.setoresMissao`   | long |   ✓    |       ✓       |    ·    |    ·     |
| `roteiros.tipoEnvio`       | text |   ✓    |       ✓       |    ·    |    ·     |
| `roteiros.tipoRoteiro`     | long |   ✓    |       ✓       |    ·    |    ·     |
| `setor`                    | long |   ✓    |       ✓       |    ·    |    ·     |
| `tipoGerencia`             | text |   ✓    |       ✓       |    ·    |    ·     |
| `tipoRegiao`               | long |   ✓    |       ✓       |    ·    |    ·     |
| `uf`                       | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/detalhes-roteiro?cidade=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `resumo-acompanhamento-sig`

- **Endpoint base:** `GET /v1/resumo-acompanhamento-sig`
- **Identificador (GET por id):** `id` → `GET /v1/resumo-acompanhamento-sig/{id}`
- **Busca textual (`q`):** `informacaoLogistica`, `informacaoMercado`, `observacao`, `solucaoInfoMercado`, `sugestao`, `tipoResumoAcompanhamento`

| Campo                      | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| -------------------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `dataEnvio`                | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataFim`                  | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataInicio`               | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataResumo`               | date |   ✓    |       ✓       |    ·    |    ✓     |
| `id`                       | long |   ✓    |       ✓       |    ·    |    ·     |
| `idEmpresa`                | long |   ✓    |       ✓       |    ·    |    ·     |
| `idSetor`                  | long |   ✓    |       ✓       |    ·    |    ·     |
| `informacaoLogistica`      | text |   ✓    |       ✓       |    ·    |    ·     |
| `informacaoMercado`        | text |   ✓    |       ✓       |    ·    |    ·     |
| `observacao`               | text |   ✓    |       ✓       |    ·    |    ·     |
| `setorRoteiro`             | long |   ✓    |       ✓       |    ·    |    ·     |
| `solucaoInfoMercado`       | text |   ✓    |       ✓       |    ·    |    ·     |
| `sugestao`                 | text |   ✓    |       ✓       |    ·    |    ·     |
| `tipoResumoAcompanhamento` | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/resumo-acompanhamento-sig?dataEnvio=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `historico-item-nf-ivendas`

- **Endpoint base:** `GET /v1/historico-item-nf-ivendas`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/historico-item-nf-ivendas/{id}`
- **Busca textual (`q`):** `cnpj`, `codBarraCxaMae`, `codBarraEmb`, `descMerc`, `id`, `unidadeVenda`

| Campo             | Tipo    | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------------- | ------- | :----: | :-----------: | :-----: | :------: |
| `cnpj`            | text    |   ✓    |       ✓       |    ·    |    ·     |
| `codBarraCxaMae`  | text    |   ✓    |       ✓       |    ·    |    ·     |
| `codBarraEmb`     | text    |   ✓    |       ✓       |    ·    |    ·     |
| `dataAtualizacao` | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataEmissao`     | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `descMerc`        | text    |   ✓    |       ✓       |    ·    |    ·     |
| `embLista`        | long    |   ✓    |       ✓       |    ·    |    ·     |
| `exclusivo`       | boolean |   ✓    |       ·       |    ·    |    ·     |
| `id`              | text    |   ✓    |       ✓       |    ·    |    ·     |
| `idFilial`        | long    |   ✓    |       ✓       |    ·    |    ·     |
| `idMercadoria`    | long    |   ✓    |       ✓       |    ·    |    ·     |
| `nroInscricao`    | long    |   ✓    |       ✓       |    ·    |    ·     |
| `nroNota`         | long    |   ✓    |       ✓       |    ·    |    ·     |
| `qtdDevolvida`    | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `qtdVendida`      | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `unidadeVenda`    | text    |   ✓    |       ✓       |    ·    |    ·     |
| `valorNf`         | float   |   ✓    |       ✓       |    ✓    |    ·     |
| `vlrTotal`        | float   |   ✓    |       ✓       |    ✓    |    ·     |
| `vlrUnitario`     | float   |   ✓    |       ✓       |    ✓    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/historico-item-nf-ivendas?cnpj.keyword=VALOR&limite=20

# agregacao: soma de qtdDevolvida por cnpj.keyword
GET /v1/historico-item-nf-ivendas/agregado?por=cnpj.keyword&metrica=sum:qtdDevolvida&top=10

# serie temporal: soma de qtdDevolvida por mes
GET /v1/historico-item-nf-ivendas/agregado?por=dataAtualizacao:mes&metrica=sum:qtdDevolvida&dataAtualizacao.gte=2025-01-01&dataAtualizacao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `gerente-vendas`

- **Endpoint base:** `GET /v1/gerente-vendas`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/gerente-vendas/{id}`
- **Busca textual (`q`):** `descRegiao`, `emailGerente`, `id`, `nomeGerente`, `tipoGerencia`

| Campo               | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ------------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `codGerente`        | long |   ✓    |       ✓       |    ·    |    ·     |
| `cpfGerente`        | long |   ✓    |       ✓       |    ·    |    ·     |
| `dataAtualizacao`   | date |   ✓    |       ✓       |    ·    |    ✓     |
| `descRegiao`        | text |   ✓    |       ✓       |    ·    |    ·     |
| `emailGerente`      | text |   ✓    |       ✓       |    ·    |    ·     |
| `empresa`           | long |   ✓    |       ✓       |    ·    |    ·     |
| `id`                | text |   ✓    |       ✓       |    ·    |    ·     |
| `matriculaGerente`  | long |   ✓    |       ✓       |    ·    |    ·     |
| `nascimentoGerente` | date |   ✓    |       ✓       |    ·    |    ✓     |
| `nomeGerente`       | text |   ✓    |       ✓       |    ·    |    ·     |
| `regiaoGerente`     | long |   ✓    |       ✓       |    ·    |    ·     |
| `regiaoMacro`       | long |   ✓    |       ✓       |    ·    |    ·     |
| `tipoGerencia`      | text |   ✓    |       ✓       |    ·    |    ·     |
| `tipoRegiao`        | long |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/gerente-vendas?codGerente=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `atom-resumo-viagem`

- **Endpoint base:** `GET /v1/atom-resumo-viagem`
- **Identificador (GET por id):** `idEmpresa` → `GET /v1/atom-resumo-viagem/{id}`
- **Busca textual (`q`):** `dataViagem`, `ddd`, `nomeMotorista`, `telefone`

| Campo           | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| --------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `dataViagem`    | text |   ✓    |       ✓       |    ·    |    ·     |
| `ddd`           | text |   ✓    |       ✓       |    ·    |    ·     |
| `idEmpresa`     | long |   ✓    |       ✓       |    ·    |    ·     |
| `idViagem`      | long |   ✓    |       ✓       |    ·    |    ·     |
| `idsClientes`   | long |   ✓    |       ✓       |    ·    |    ·     |
| `motorista`     | long |   ✓    |       ✓       |    ·    |    ·     |
| `nomeMotorista` | text |   ✓    |       ✓       |    ·    |    ·     |
| `telefone`      | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/atom-resumo-viagem?dataViagem.keyword=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `foto-aceite-transportadora`

- **Endpoint base:** `GET /v1/foto-aceite-transportadora`
- **Identificador (GET por id):** `REVISAR` → `GET /v1/foto-aceite-transportadora/{id}`
- **Busca textual (`q`):** `foto`, `qrCode`, `status`

| Campo         | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `dataCriacao` | date |   ✓    |       ✓       |    ·    |    ✓     |
| `foto`        | text |   ✓    |       ✓       |    ·    |    ·     |
| `qrCode`      | text |   ✓    |       ✓       |    ·    |    ·     |
| `status`      | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/foto-aceite-transportadora?dataCriacao=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `item-pedido-v2`

- **Endpoint base:** `GET /v1/item-pedido-v2`
- **Identificador (GET por id):** `id` → `GET /v1/item-pedido-v2/{id}`

| Campo                    | Tipo         | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ------------------------ | ------------ | :----: | :-----------: | :-----: | :------: |
| `cnpjCliente`            | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `dataAtualizacao`        | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `dataEntrada`            | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `dataPedidoPda`          | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `descCidade`             | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `descCondVenda`          | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `descDivisao`            | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `descFormaPagamento`     | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `descGrupo`              | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `descRamoAtividade`      | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `descSubGrupo`           | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `descricaoCompleta`      | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `divisao`                | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `embLista`               | short        |   ✓    |       ✓       |    ·    |    ·     |
| `estado`                 | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `filialEmissora`         | short        |   ✓    |       ✓       |    ·    |    ·     |
| `grupo`                  | short        |   ✓    |       ✓       |    ·    |    ·     |
| `id`                     | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `idCidade`               | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idCliente`              | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idCondVenda`            | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idEmpresa`              | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idFormaPagamento`       | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idMercadoria`           | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idPedido`               | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idPedidoElastic`        | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `idRamoAtividade`        | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idRepresentante`        | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idSetor`                | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idSetorOrigem`          | long         |   ✓    |       ✓       |    ·    |    ·     |
| `mapex`                  | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `nfDataEmissao`          | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `nfDataSaida`            | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `nomeUsuario`            | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `notaFiscal`             | long         |   ✓    |       ✓       |    ·    |    ·     |
| `origemPedido`           | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `precoUnitario`          | scaled_float |   ✓    |       ✓       |    ✓    |    ·     |
| `qtdBonificacao`         | short        |   ✓    |       ✓       |    ✓    |    ·     |
| `qtdCorte`               | short        |   ✓    |       ✓       |    ✓    |    ·     |
| `qtdVendida`             | short        |   ✓    |       ✓       |    ✓    |    ·     |
| `razaoSocial`            | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `sequenciaItem`          | short        |   ✓    |       ✓       |    ·    |    ·     |
| `situacaoPedido`         | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `subGrupo`               | short        |   ✓    |       ✓       |    ·    |    ·     |
| `tipoPedido`             | short        |   ✓    |       ✓       |    ·    |    ·     |
| `unidadeVenda`           | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `usuario`                | long         |   ✓    |       ✓       |    ·    |    ·     |
| `valorPedido`            | scaled_float |   ✓    |       ✓       |    ✓    |    ·     |
| `vlrBaseCalculoComissao` | scaled_float |   ✓    |       ✓       |    ✓    |    ·     |
| `vlrComissao`            | scaled_float |   ✓    |       ✓       |    ✓    |    ·     |
| `vlrTotalItem`           | float        |   ✓    |       ✓       |    ✓    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/item-pedido-v2?cnpjCliente=VALOR&limite=20

# agregacao: soma de precoUnitario por cnpjCliente
GET /v1/item-pedido-v2/agregado?por=cnpjCliente&metrica=sum:precoUnitario&top=10

# serie temporal: soma de precoUnitario por mes
GET /v1/item-pedido-v2/agregado?por=dataAtualizacao:mes&metrica=sum:precoUnitario&dataAtualizacao.gte=2025-01-01&dataAtualizacao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `localizacao-usuario-ivendas`

- **Endpoint base:** `GET /v1/localizacao-usuario-ivendas`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/localizacao-usuario-ivendas/{id}`
- **Busca textual (`q`):** `acao`, `id`, `nroInscricaoCliente`

| Campo                 | Tipo  | Filtro | Ordena/Agrupa | Métrica | Temporal |
| --------------------- | ----- | :----: | :-----------: | :-----: | :------: |
| `acao`                | text  |   ✓    |       ✓       |    ·    |    ·     |
| `acuracidade`         | float |   ✓    |       ✓       |    ·    |    ·     |
| `dataCriacao`         | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `dataCriacaoServer`   | date  |   ✓    |       ✓       |    ·    |    ✓     |
| `direcao`             | float |   ✓    |       ✓       |    ·    |    ·     |
| `distancia`           | float |   ✓    |       ✓       |    ✓    |    ·     |
| `id`                  | text  |   ✓    |       ✓       |    ·    |    ·     |
| `idRepresentante`     | long  |   ✓    |       ✓       |    ·    |    ·     |
| `idSetor`             | long  |   ✓    |       ✓       |    ·    |    ·     |
| `latitude`            | float |   ✓    |       ✓       |    ·    |    ·     |
| `latitudeCliente`     | float |   ✓    |       ✓       |    ·    |    ·     |
| `longitude`           | float |   ✓    |       ✓       |    ·    |    ·     |
| `longitudeCliente`    | float |   ✓    |       ✓       |    ·    |    ·     |
| `nroInscricaoCliente` | text  |   ✓    |       ✓       |    ·    |    ·     |
| `velocidade`          | float |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/localizacao-usuario-ivendas?acao.keyword=VALOR&limite=20

# agregacao: soma de distancia por acao.keyword
GET /v1/localizacao-usuario-ivendas/agregado?por=acao.keyword&metrica=sum:distancia&top=10

# serie temporal: soma de distancia por mes
GET /v1/localizacao-usuario-ivendas/agregado?por=dataCriacao:mes&metrica=sum:distancia&dataCriacao.gte=2025-01-01&dataCriacao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `pedido-v2`

- **Endpoint base:** `GET /v1/pedido-v2`
- **Identificador (GET por id):** `id` → `GET /v1/pedido-v2/{id}`

| Campo                | Tipo         | Filtro | Ordena/Agrupa | Métrica | Temporal |
| -------------------- | ------------ | :----: | :-----------: | :-----: | :------: |
| `cnpjCliente`        | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `dataAtualizacao`    | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `dataEntrada`        | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `dataPedidoPda`      | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `descCidade`         | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `descCondVenda`      | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `descFormaPagamento` | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `descRamoAtividade`  | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `estado`             | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `filialEmissora`     | short        |   ✓    |       ✓       |    ·    |    ·     |
| `id`                 | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `idCidade`           | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idCliente`          | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idCondVenda`        | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idEmpresa`          | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idFormaPagamento`   | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idPedido`           | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idRamoAtividade`    | short        |   ✓    |       ✓       |    ·    |    ·     |
| `idRepresentante`    | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idSetor`            | long         |   ✓    |       ✓       |    ·    |    ·     |
| `idSetorOrigem`      | long         |   ✓    |       ✓       |    ·    |    ·     |
| `nfDataEmissao`      | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `nfDataSaida`        | date         |   ✓    |       ✓       |    ·    |    ✓     |
| `nomeUsuario`        | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `notaFiscal`         | long         |   ✓    |       ✓       |    ·    |    ·     |
| `origemPedido`       | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `razaoSocial`        | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `situacaoPedido`     | keyword      |   ✓    |       ✓       |    ·    |    ·     |
| `tipoPedido`         | short        |   ✓    |       ✓       |    ·    |    ·     |
| `usuario`            | long         |   ✓    |       ✓       |    ·    |    ·     |
| `valorPedido`        | scaled_float |   ✓    |       ✓       |    ✓    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/pedido-v2?cnpjCliente=VALOR&limite=20

# agregacao: soma de valorPedido por cnpjCliente
GET /v1/pedido-v2/agregado?por=cnpjCliente&metrica=sum:valorPedido&top=10

# serie temporal: soma de valorPedido por mes
GET /v1/pedido-v2/agregado?por=dataAtualizacao:mes&metrica=sum:valorPedido&dataAtualizacao.gte=2025-01-01&dataAtualizacao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `acompanhamento-setor`

- **Endpoint base:** `GET /v1/acompanhamento-setor`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/acompanhamento-setor/{id}`
- **Busca textual (`q`):** `erro`, `id`, `status`

| Campo              | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ------------------ | ---- | :----: | :-----------: | :-----: | :------: |
| `dataInicio`       | date |   ✓    |       ✓       |    ·    |    ✓     |
| `empresa`          | long |   ✓    |       ✓       |    ·    |    ·     |
| `erro`             | text |   ✓    |       ✓       |    ·    |    ·     |
| `id`               | text |   ✓    |       ✓       |    ·    |    ·     |
| `representante`    | long |   ✓    |       ✓       |    ·    |    ·     |
| `setor`            | long |   ✓    |       ✓       |    ·    |    ·     |
| `setorAcompanhado` | long |   ✓    |       ✓       |    ·    |    ·     |
| `status`           | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/acompanhamento-setor?dataInicio=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `regiao-gerente`

- **Endpoint base:** `GET /v1/regiao-gerente`
- **Identificador (GET por id):** `id.keyword` → `GET /v1/regiao-gerente/{id}`
- **Busca textual (`q`):** `descRegiaoVenda`, `descTipoRegiao`, `dsCidade`, `ehRespNac`, `id`, `nome`, `nomeMacro`, `nomeVenda`, `tipoGerencia`, `uf`

| Campo             | Tipo | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------------- | ---- | :----: | :-----------: | :-----: | :------: |
| `cidade`          | long |   ✓    |       ✓       |    ·    |    ·     |
| `dataAtualizacao` | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataGozoFim`     | date |   ✓    |       ✓       |    ·    |    ✓     |
| `dataGozoInicio`  | date |   ✓    |       ✓       |    ·    |    ✓     |
| `descRegiaoVenda` | text |   ✓    |       ✓       |    ·    |    ·     |
| `descTipoRegiao`  | text |   ✓    |       ✓       |    ·    |    ·     |
| `distrito`        | long |   ✓    |       ✓       |    ·    |    ·     |
| `dsCidade`        | text |   ✓    |       ✓       |    ·    |    ·     |
| `ehRespNac`       | text |   ✓    |       ✓       |    ·    |    ·     |
| `fimAfast`        | date |   ✓    |       ✓       |    ·    |    ✓     |
| `funcionario`     | long |   ✓    |       ✓       |    ·    |    ·     |
| `gerente`         | long |   ✓    |       ✓       |    ·    |    ·     |
| `id`              | text |   ✓    |       ✓       |    ·    |    ·     |
| `inicioAfast`     | date |   ✓    |       ✓       |    ·    |    ✓     |
| `nome`            | text |   ✓    |       ✓       |    ·    |    ·     |
| `nomeMacro`       | text |   ✓    |       ✓       |    ·    |    ·     |
| `nomeVenda`       | text |   ✓    |       ✓       |    ·    |    ·     |
| `regiaoMacro`     | long |   ✓    |       ✓       |    ·    |    ·     |
| `regiaoVenda`     | long |   ✓    |       ✓       |    ·    |    ·     |
| `setor`           | long |   ✓    |       ✓       |    ·    |    ·     |
| `tipoGerencia`    | text |   ✓    |       ✓       |    ·    |    ·     |
| `tipoRegiao`      | long |   ✓    |       ✓       |    ·    |    ·     |
| `uf`              | text |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/regiao-gerente?cidade=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `promocoes-promox`

- **Endpoint base:** `GET /v1/promocoes-promox`
- **Identificador (GET por id):** `id` → `GET /v1/promocoes-promox/{id}`
- **Busca textual (`q`):** `descricao`, `faixasPremio.descricaoMercadoria`, `faixasPremio.descricaoMercadoria2`, `geraBrindeTablet`, `observacao`, `participaTelevendas`, `permiteBonificacao`, `permiteCommodity`, `regioesVendaEstado.estado`, `segmentos.descricaoMercadoria`, `tipoBrinde`, `trocaBrinde`, `validoElasticsearch`

| Campo                                     | Tipo    | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------------------------------------- | ------- | :----: | :-----------: | :-----: | :------: |
| `dataAtualizacao`                         | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataFim`                                 | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dataInicio`                              | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `descricao`                               | text    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPorRepresentante`                  | boolean |   ✓    |       ·       |    ·    |    ·     |
| `faixasPremio.codigoFotoMercadoria`       | long    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremio.codigoFotoMercadoria2`      | long    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremio.custo`                      | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `faixasPremio.custo2`                     | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `faixasPremio.descricaoMercadoria`        | text    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremio.descricaoMercadoria2`       | text    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremio.faixa`                      | long    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremio.idMercadoria`               | long    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremio.idMercadoria2`              | long    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremio.idRepresentante`            | long    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremio.qtdMercadoria`              | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `faixasPremio.qtdMercadoria2`             | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `faixasPremio.valorFinal`                 | float   |   ✓    |       ✓       |    ✓    |    ·     |
| `faixasPremio.valorInicial`               | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `faixasPremio.valorPremio`                | float   |   ✓    |       ✓       |    ✓    |    ·     |
| `faixasPremio.versaoFoto`                 | long    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremio.versaoFoto2`                | long    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremio.voltagem`                   | long    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremio.voltagem2`                  | long    |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremioEngorda.multiplicaComissao`  | float   |   ✓    |       ✓       |    ✓    |    ·     |
| `faixasPremioEngorda.premioFaixaPromocao` | float   |   ✓    |       ✓       |    ·    |    ·     |
| `faixasPremioEngorda.vlrFaixaGeral`       | float   |   ✓    |       ✓       |    ✓    |    ·     |
| `faixasPremioEngorda.vlrFaixaPromocao`    | float   |   ✓    |       ✓       |    ✓    |    ·     |
| `geraBrindeTablet`                        | text    |   ✓    |       ✓       |    ·    |    ·     |
| `id`                                      | long    |   ✓    |       ✓       |    ·    |    ·     |
| `idEmpresa`                               | long    |   ✓    |       ✓       |    ·    |    ·     |
| `idPromocaoComposicao`                    | long    |   ✓    |       ✓       |    ·    |    ·     |
| `idPromocaoPrimaria`                      | long    |   ✓    |       ✓       |    ·    |    ·     |
| `idsRegiaoVendas`                         | long    |   ✓    |       ✓       |    ·    |    ·     |
| `idsSegmentos`                            | long    |   ✓    |       ✓       |    ·    |    ·     |
| `leveQtdMercadoria`                       | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `metaParticipacao`                        | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `modeloPromocao`                          | long    |   ✓    |       ✓       |    ·    |    ·     |
| `observacao`                              | text    |   ✓    |       ✓       |    ·    |    ·     |
| `pagueQtdMercadoria`                      | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `participaTelevendas`                     | text    |   ✓    |       ✓       |    ·    |    ·     |
| `permiteBonificacao`                      | text    |   ✓    |       ✓       |    ·    |    ·     |
| `permiteCommodity`                        | text    |   ✓    |       ✓       |    ·    |    ·     |
| `qtdDiasInativoFim`                       | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `qtdDiasInativoInicio`                    | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `regioesVendaEstado.estado`               | text    |   ✓    |       ✓       |    ·    |    ·     |
| `regioesVendaEstado.idRegiaoVenda`        | long    |   ✓    |       ✓       |    ·    |    ·     |
| `relatorio`                               | long    |   ✓    |       ✓       |    ·    |    ·     |
| `segmentos.custo`                         | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `segmentos.custo2`                        | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `segmentos.descricaoMercadoria`           | text    |   ✓    |       ✓       |    ·    |    ·     |
| `segmentos.faixa`                         | long    |   ✓    |       ✓       |    ·    |    ·     |
| `segmentos.valorFinal`                    | float   |   ✓    |       ✓       |    ✓    |    ·     |
| `segmentos.valorInicial`                  | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `segmentos.valorPremio`                   | long    |   ✓    |       ✓       |    ✓    |    ·     |
| `tipoBrinde`                              | text    |   ✓    |       ✓       |    ·    |    ·     |
| `tipoCalculo`                             | long    |   ✓    |       ✓       |    ·    |    ·     |
| `tipoPromocao`                            | long    |   ✓    |       ✓       |    ·    |    ·     |
| `trocaBrinde`                             | text    |   ✓    |       ✓       |    ·    |    ·     |
| `validoElasticsearch`                     | text    |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/promocoes-promox?dataAtualizacao=VALOR&limite=20

# agregacao: soma de faixasPremio.custo por descricao.keyword
GET /v1/promocoes-promox/agregado?por=descricao.keyword&metrica=sum:faixasPremio.custo&top=10

# serie temporal: soma de faixasPremio.custo por mes
GET /v1/promocoes-promox/agregado?por=dataAtualizacao:mes&metrica=sum:faixasPremio.custo&dataAtualizacao.gte=2025-01-01&dataAtualizacao.lt=2026-01-01
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---

## `funcionario-asc`

- **Endpoint base:** `GET /v1/funcionario-asc`
- **Identificador (GET por id):** `id` → `GET /v1/funcionario-asc/{id}`
- **Busca textual (`q`):** `email`, `nome`, `tags`

| Campo             | Tipo    | Filtro | Ordena/Agrupa | Métrica | Temporal |
| ----------------- | ------- | :----: | :-----------: | :-----: | :------: |
| `dataAtualizacao` | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `dddFone`         | long    |   ✓    |       ✓       |    ·    |    ·     |
| `demissao`        | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `ehMotorista`     | boolean |   ✓    |       ·       |    ·    |    ·     |
| `email`           | text    |   ✓    |       ✓       |    ·    |    ·     |
| `fone`            | long    |   ✓    |       ✓       |    ·    |    ·     |
| `id`              | long    |   ✓    |       ✓       |    ·    |    ·     |
| `matricula`       | long    |   ✓    |       ✓       |    ·    |    ·     |
| `nascimento`      | date    |   ✓    |       ✓       |    ·    |    ✓     |
| `nome`            | text    |   ✓    |       ✓       |    ·    |    ·     |
| `tags`            | text    |   ✓    |       ✓       |    ·    |    ·     |

**Exemplos:**

```
# listagem
GET /v1/funcionario-asc?dataAtualizacao=VALOR&limite=20
```

> **Semântica a revisar (preencher):** significado de campos de código, definição de datas (pedido vs faturado), códigos de situação.

---
