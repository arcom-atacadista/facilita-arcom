---
name: gateway-arcom
description: Consulta o Gateway de dados da ARCOM (API interna com ~42 datasets da empresa — clientes, débitos, vendas, roteiros, promoções, RH etc.), documentado em padroes/12-gateway-arcom.md e padroes/gateway-arcom/. Use quando o usuário pedir para buscar, listar, integrar, exibir ou cruzar dados reais da empresa vindos desse gateway.
---

# Gateway de dados ARCOM

Leia `padroes/12-gateway-arcom.md` primeiro — é o padrão que explica o que é
o Gateway, as convenções da API e (o mais importante) por que este
repositório nunca tem a `X-API-Key` real. A referência completa de cada
dataset — campos, tipos, filtros, exemplos — está em
`padroes/gateway-arcom/api.md` (legível) e `padroes/gateway-arcom/openapi.yaml`
(spec completa). **Não adivinhe campo, dataset ou endpoint: leia o dataset
relevante nesses arquivos antes de escrever qualquer chamada.**

## Antes de escrever código

1. **Descubra o dataset certo.** `padroes/gateway-arcom/api.md` lista os
   datasets como `## \`nome-do-dataset\``. Procure pelo domínio que o
   usuário descreveu (ex.: "débitos de cliente" → `debitos` ou
   `debitos-cobranca-v2`; "equipe de vendas" → `equipe-vendas-setor`). Se
   mais de um dataset parecer servir, pergunte qual é o certo em vez de
   escolher por conta própria.
2. **Pergunte o que faltar no pedido:**
   - quais campos o usuário quer ver, filtrar ou agrupar (a tabela de cada
     dataset mostra o que é filtrável, ordenável, métrica e temporal);
   - se é listagem simples, agregação (`/agregado?por=&metrica=&top=`) ou
     série temporal (`campo:mes`/`:semana`/`:dia`/`:ano`);
   - o intervalo de datas, quando houver filtro temporal.
3. **Não invente o significado de campos "a revisar".** A maioria dos
   datasets termina com `> Semântica a revisar (preencher)` — se o pedido
   depender desses campos, avise que o significado não está documentado e
   confirme com o usuário antes de usar.

## A API key é segredo de produção

Este repositório **não tem e nunca deve ter** a `X-API-Key` real do
Gateway — ela só existe no servidor de produção, provisionada pelo time de
TI da ARCOM (detalhe em `padroes/12-gateway-arcom.md`). Você pode e deve
escrever a integração inteira mesmo sem ela: a chave entra por
`config.Obrigatorio("GATEWAY_ARCOM_API_KEY")` em runtime. Antes de considerar
a feature pronta pra produção, avise o usuário que ele precisa acionar o
time de TI/infraestrutura pra provisionar essa chave no servidor — sem isso
o código funciona mas a chamada real ao Gateway não.

## Integrando no backend

- A chamada ao gateway é **sempre feita pelo backend Go**, nunca pelo
  frontend — a `X-API-Key` é um segredo (ver skill `seguranca`) e não pode
  chegar ao navegador.
- Siga a skill `novo-endpoint` pra estrutura: um cliente do gateway fica em
  algo como `backend/internal/gatewayarcom/client.go`; o endpoint do
  projeto que expõe o dado ao frontend é um recurso normal
  (`handler.go`/`service.go`) que chama esse client e devolve só os campos
  que o front precisa — não repasse o payload do gateway inteiro.
- `padroes/12-gateway-arcom.md` tem um exemplo completo em Go (client,
  service consumindo um dataset específico, boot lendo a chave) — use como
  modelo em vez de inventar a estrutura do zero.
- Trate erro do gateway (timeout, 4xx/5xx) como erro de dependência
  externa, sem deixar estourar sem tratamento pro cliente final; formato de
  erro segue `09-contrato-api.md`.
- Se o dado for exibido em tela, aplique o Design System ARCOM
  (`design/ARCOM-Design-System.md`) — não estilize à mão.

## Regras

- Nunca invente dataset, campo ou endpoint que não esteja documentado em
  `padroes/gateway-arcom/api.md`/`openapi.yaml`.
- Nunca hardcode a API key, peça pro usuário colar ela no código, nem a
  exponha no frontend ou em log.
