---
name: revisar-seguranca
description: Auditoria de segurança do código já escrito neste projeto (Go + React na stack ARCOM) — encontra, classifica por severidade (CVSS/CWE/OWASP) e reporta vulnerabilidades com PoC e correção. Use quando o usuário pedir "revisão de segurança", "auditoria", "pentest", antes de um deploy de produção importante, ou depois de terminar uma feature de autenticação/pagamento/dado sensível.
---

# Revisar segurança (auditoria do código já escrito)

Esta skill é **detectiva** (audita o que existe); a skill `seguranca` é
**preventiva** (como escrever certo desde o início). Rode esta depois de uma
feature sensível pronta, ou sob pedido de auditoria/deploy importante.

## 0. Classifique o projeto antes de tudo

Leia `padroes/10-seguranca.md`. Determine INTERNA / EXTERNA / MISTA e escreva
isso no topo do relatório — muda o que é "esperado" (ex.: app interna sem
signup público é normal; app externa sem signup é suspeito).

## 1. Fan-out em 4 frentes (paralelo)

Lance 4 sub-agentes, cada um cobrindo uma fatia — evita um agente só tentando
lembrar de tudo:

| Agente | Escopo | Knowledge de referência |
|---|---|---|
| **auth** | sessão/cookie, middleware de autenticação, ownership/IDOR, CSRF, rate limit de login, JWT (se usado) | `knowledge/padroes-vulneraveis.md` §Auth |
| **config** | secrets (`.env`, `VITE_*`), allowlist de dependência (`01-stack-permitida.md`), Dockerfile/non-root, headers HTTP | `knowledge/padroes-vulneraveis.md` §Config |
| **injecao** | SQLi (`Raw`/`Exec`/`Sprintf`), mass assignment, `dangerouslySetInnerHTML`, validação de entrada (`validator`/`zod`), SSRF | `knowledge/padroes-vulneraveis.md` §Injeção |
| **dados** | LGPD (PII em log, dado sensível Art. 5º-II), rate limit fail-open, confiança em `X-Forwarded-For` | `knowledge/padroes-vulneraveis.md` §Dados/LGPD |

Cada agente lê o código relevante (handlers, middlewares, `core/api.ts`,
`.env.example`, `package.json`/`go.mod`) e devolve findings no schema JSON de
`knowledge/severidade.md`.

## 2. Anti-ruído — aplique ANTES de reportar

Leia `knowledge/falsos-positivos.md`. Regra central: **não infle**. Tabela
vazia (`200 {}`) não é vulnerabilidade. `CORS *` num endpoint que já exige
sessão válida é severidade baixa, não crítica. Na dúvida entre "é
vulnerabilidade real" e "é comportamento esperado", marque como
`precisa_esclarecimento` e pergunte — não decida sozinho pro lado do drama.

Todo finding CRÍTICO/ALTO precisa de PoC **estático** (trecho de código real,
nunca execução contra produção) e de correção com código concreto — "corrija a
validação" não é uma correção, mostrar o `validator` tag certo é.

## 3. Segundo turno — caçar o que passou batido

Depois da primeira rodada, rode **mais um turno** (agente novo, sem viés do
primeiro) perguntando especificamente: "o que os 4 agentes anteriores podem
ter deixado passar?" — foque em: rota nova sem middleware de auth; segredo
introduzido numa migration; dependência nova fora do allowlist; padrão
inseguro coexistindo com um seguro (a mesma operação implementada duas vezes,
uma seura e uma não — risco de regressão).

## 4. Comparação com execução anterior (se existir relatório prévio)

Pra cada finding antigo, marque: `CONFIRMADO` (mesmo, ainda lá) / `MITIGADO`
(corrigido) / `AGRAVADO` (piorou) / `NOVO`. Isso é o que detecta quando uma
edição posterior da IA **reintroduziu** algo já corrigido.

## 5. Relatório final

Formato completo em `knowledge/template-relatorio.md`. Resumo executivo em
português simples primeiro (o público é não-técnico), tabela quantitativa por
severidade, achados detalhados por último. Se um achado for crítico e ativo
(segredo vazado, IDOR explorável), aponte pra `knowledge/incidente.md` também.

## Checklist de qualidade antes de entregar

- [ ] Classificação INTERNA/EXTERNA/MISTA justificada.
- [ ] Todo CRÍTICO/ALTO tem PoC estático + correção com código.
- [ ] Falsos positivos de `knowledge/falsos-positivos.md` aplicados.
- [ ] CVSS/CWE/categoria OWASP preenchidos (ver `knowledge/severidade.md`).
- [ ] Segundo turno rodou e não achou nada crítico não coberto.
- [ ] Sumário executivo em português simples, sem jargão.
