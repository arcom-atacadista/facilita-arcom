# 07 — Frontend primeiro (nem todo projeto precisa de backend)

Regra de bolso: **todo projeto começa só com frontend.** Backend e banco de
dados só entram quando realmente forem necessários. Não crie servidor nem
Postgres "por via das dúvidas".

## Por que começar só com frontend

- É mais simples, sobe mais rápido e tem menos coisa pra dar errado.
- A maioria das ideias no começo é tela: landing page, protótipo, calculadora,
  formulário, painel com dados de exemplo, ferramenta que roda no navegador.
- Você adiciona o backend depois, sem retrabalho — a estrutura já prevê isso.

## Dá pra fazer só com frontend

- Telas e navegação (React + react-router-dom)
- Formulários (react-hook-form) que mostram/validam sem salvar em servidor
- Estado que vive só na sessão (memória do app enquanto está aberto)
- Layout, design, PWA — tudo isso é frontend

## O sinal de que chegou a hora do backend

Pense em criar backend **+ PostgreSQL** quando aparecer **persistência** — ou
seja, guardar dado que precisa continuar existindo:

- os dados precisam **sobreviver** depois de fechar/atualizar a página
- vários usuários (ou vários aparelhos) precisam **ver os mesmos dados**
- precisa de **login / contas de usuário**
- precisa **esconder um segredo** (chave de API privada) — segredo nunca fica no
  frontend
- precisa falar com uma **API externa** (o navegador não chama direto; o backend
  faz isso por você)
- tem uma **regra de negócio** que não pode ficar exposta no navegador

Se nada disso se aplica, siga só com frontend.

## Como o projeto reflete isso

**Projeto só frontend:** o `docker-compose.yml` fica só com o serviço
`frontend` (apague `backend`, `migrate` e `postgres`, e a variável
`DATABASE_URL`). Roda igual: `docker compose up --build`.

**Quando precisar de persistência:** peça ao chat pra adicionar o backend. Ele
vai:
1. criar a pasta `backend/` seguindo o `03-backend.md`
2. recolocar os serviços `backend` + `migrate` (aplica as migrations) +
   `postgres` no `docker-compose.yml`
3. ligar o frontend ao backend pelo `/api/v1` (o proxy já está pronto; formato
   de dado/erro em `09-contrato-api.md`)

Repare que o Redis **não** entra automaticamente junto com o backend — é uma
decisão à parte (ver `01-stack-permitida.md`). Só peça pro chat adicionar o
serviço `redis` quando aparecer uma necessidade concreta: cache, fila de
jobs (`asynq`), rate limit distribuído entre réplicas, sessão compartilhada.
Ter backend + Postgres não significa que o projeto precisa de Redis também.

Ou seja: você não perde nada começando pequeno. Backend, Postgres e Redis
entram encaixados, cada um na sua hora, quando a necessidade aparecer — nunca
todos de uma vez "por via das dúvidas".
