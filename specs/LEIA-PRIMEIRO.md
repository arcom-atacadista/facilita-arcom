> **Padrão ARCOM — v3.2** • se você recebeu uma versão mais nova, repita os
> passos abaixo com os arquivos novos (veja `VERSAO.md`).

# 👋 Leia primeiro

Este repositório é o **Padrão ARCOM**: as regras técnicas, a identidade
visual e os guardrails automáticos que qualquer projeto feito com Claude
(Chat ou Code) na ARCOM deve seguir. Ele é ao mesmo tempo:

- um **template do GitHub** — pra começar um projeto **novo**;
- um **pacote pra colar** dentro de um repositório que **já existe**.

Antes de tudo, leia `privacy/` — o que pode e o que não pode ser digitado
numa ferramenta de IA (dado de cliente, senha, informação financeira etc.).

## Modo 1 — Projeto novo (Claude Code)

1. No GitHub, clique em **Use this template** neste repositório.
2. Abra o Claude Code dentro do repositório novo. Ele lê o `CLAUDE.md` da
   raiz sozinho — nenhuma instrução precisa ser colada no chat.
3. Peça o que precisar em português normal: *"cria uma tela de login"*,
   *"faz um endpoint de produtos"*.
4. Rode tudo com um comando:

   ```bash
   docker compose up --build
   ```

   Abra **http://localhost:8080** — já sobe uma tela on-brand com o backend
   conectado (`GET /api/health` / `GET /api/ready`). Não precisa montar nada
   do zero.

## Modo 2 — Repositório que já existe (Claude Code)

Se o projeto já está rodando (já tem o frontend/backend dele), **não** traga
`frontend/`, `backend/` nem os `docker-compose*.yml` daqui — só o que é
"padrão da plataforma", sem mexer no que já existe:

- `CLAUDE.md` (raiz)
- `.claude/` (hooks + skills)
- `system-design/` (esta pasta inteira, como está — é travada, ninguém edita)

Copie essas três coisas pra raiz do repositório existente. Na próxima vez que
alguém abrir o Claude Code ali, o Padrão ARCOM já está valendo — e o guard
`proteger-arquivo.js` recusa qualquer tentativa de editar algo dentro de
`system-design/` a partir dali.

**Se a stack desse projeto já é diferente da ARCOM** (não é React+Vite no
frontend e Go no backend), crie também um arquivo vazio
`.claude/PROJETO-EXISTENTE`. Isso avisa o Claude Code que é um projeto
legado: as regras de stack do padrão passam a valer só pra código **novo**,
sem forçar migração do que já existe. Segurança, Design System e a trava de
`system-design/` continuam valendo sempre, com ou sem esse arquivo.

## Modo 3 — Claude Chat (claude.ai, sem Code)

1. Crie um **Projeto** novo no Claude (Projetos → Novo projeto).
2. Suba esta pasta (`system-design/`, com `padroes/`, `design/`, `privacy/` e
   o `INSTRUCOES-DO-PROJETO.md`) no **Conhecimento** do projeto.
3. Copie o conteúdo de `INSTRUCOES-DO-PROJETO.md` e cole no campo
   **Instruções** do Projeto (não no chat).
4. Converse no chat pedindo o que precisar — o Claude já segue os padrões.

## O que tem aqui dentro (`system-design/`)

- `INSTRUCOES-DO-PROJETO.md` — pra colar nas Instruções do Projeto (Modo 3).
- `padroes/` — guias curtos, um por assunto:
  - `00-visao-geral.md` — como o projeto funciona por cima
  - `01-stack-permitida.md` — o que você pode e o que **não** pode usar
  - `02-frontend.md` — a tela (React + Tailwind)
  - `03-backend.md` — o servidor em Go
  - `04-docker-deploy.md` — o modelo Docker e como rodar
  - `05-regras-para-a-ia.md` — as regras que o chat segue
  - `06-estrutura-repo.md` — a estrutura de pastas no GitHub
  - `07-frontend-primeiro.md` — nem todo projeto precisa de backend
  - `08-design-system.md` — identidade visual da ARCOM (obrigatório em UI)
  - `09-contrato-api.md` — formato de dado/erro entre front e back
  - `10-seguranca.md` — classificação de risco, severidade, LGPD
  - `11-guards.md` — o que é bloqueado automaticamente e por quê
- `design/` — identidade visual ARCOM (cores, logo, fontes, tom de voz)
- `privacy/` — termos de uso, privacidade e boas práticas de IA (leitura obrigatória)
- `scripts/verificar.sh` — varredura completa de lint/segurança, sob demanda
- `scripts/guards/*.js` — os scripts que os hooks de `.claude/settings.json` chamam
- `scripts/setup-storage-prod.sh` — teto de disco dos dados em produção
- `VERSAO.md` — versão atual do padrão

## E fora do `system-design/` (raiz do repo)

- `CLAUDE.md` — lido sozinho pelo Claude Code.
- `.claude/settings.json` — hooks que bloqueiam desvio do padrão na hora (ver `padroes/11-guards.md`)
- `.claude/skills/` — atalhos pro Claude Code: `nova-tela`, `novo-endpoint`,
  `seguranca`, `autenticacao`, `revisar-seguranca`, `deploy-producao`
- `docker-compose.yml` / `docker-compose.prod.yml` — só usados no Modo 1 (projeto novo)
- `.env.example` — variáveis (copie pra `.env` em dev ou `.env.prod` em produção)
- `frontend/`, `backend/` — só usados no Modo 1: um **hello-world que já sobe**
  (tela on-brand + `GET /api/health` + `GET /api/ready`), pronto pra construir por cima

## Regras de ouro (o resto está nos guias)

- Front chama o back sempre em `/api/...` (o Vite faz o proxy, sem CORS).
- Banco (postgres/redis) só se o projeto precisar — dá pra remover.
- Só bibliotecas da lista permitida (`padroes/01-stack-permitida.md`).
- Nunca coloque senha ou chave secreta no código do frontend.
- Visual sempre do Design System ARCOM (`design/ARCOM-Design-System.md`), sem emoji.

## Se o padrão for atualizado

A versão atual está em `VERSAO.md`. Repita o Modo 1 ou 2 com os arquivos
novos; no Modo 3, substitua os arquivos antigos no Conhecimento do Projeto.
