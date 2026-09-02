# Onboarding — Padrão ARCOM no Claude Code

Este repositório é o **Padrão ARCOM**: as regras técnicas, a identidade
visual e os guardrails automáticos que qualquer projeto feito com Claude
Code na ARCOM deve seguir.

## Projeto novo

1. No GitHub, clique em **Use this template** neste repositório.
2. Abra o Claude Code dentro do repositório novo — ele lê o `CLAUDE.md` da
   raiz sozinho, nenhuma instrução precisa ser colada no chat.
3. Rode `docker compose up --build` e abra http://localhost:8080. Já sobe
   uma tela on-brand com o backend conectado.
4. Peça o que precisar em português normal: *"cria uma tela de cadastro de
   fornecedor"*, *"adiciona autenticação por cookie"*, *"monta um painel com
   os pedidos do dia"*.

## Repositório que já existe

Não traga `frontend/`, `backend/` nem os `docker-compose*.yml` — o projeto já
tem os dele. Copie só o "padrão da plataforma" pra raiz do repositório:

- `CLAUDE.md`
- `.claude/` (hooks + skills)
- `system-design/` (a pasta inteira, como está — é travada, ninguém edita)

Na próxima vez que o Claude Code abrir ali, o Padrão ARCOM já está valendo, e
o guard `proteger-arquivo.js` recusa qualquer edição dentro de
`system-design/`.

**Stack diferente da ARCOM?** (não é React+Vite/Go) Crie também um arquivo
vazio `.claude/PROJETO-EXISTENTE`. As regras de stack passam a valer só pra
código novo, sem forçar migração do que já existe — segurança, Design System
e a trava de `system-design/` continuam valendo sempre.

## O que já vem garantido

- Visual sempre do Design System ARCOM (`system-design/design/ARCOM-Design-System.md`)
- Stack só do allowlist (`system-design/padroes/01-stack-permitida.md`)
- Segurança básica obrigatória (`system-design/padroes/10-seguranca.md`)
- Simplicidade primeiro: começa só com frontend, backend entra quando
  realmente precisar (`system-design/padroes/07-frontend-primeiro.md`)

## Antes de tudo

Leia `system-design/privacy/` — o que pode e o que não pode ser digitado numa
ferramenta de IA (dado de cliente, senha, informação financeira etc.).

## Se o padrão for atualizado

A versão atual está em `system-design/VERSAO.md`. Quando houver uma
atualização, repita os passos acima com os arquivos novos.

## Usando Claude Chat (claude.ai) em vez do Code?

Veja `system-design/LEIA-PRIMEIRO.md` — Modo 3 — pra usar via **Project** no claude.ai
(Project knowledge + campo Instruções) em vez do Claude Code.
