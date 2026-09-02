# 10 — Segurança (política e classificação)

Este guia é a política; o **como fazer** dia a dia está nas skills:
`.claude/skills/seguranca` (preventivo — escrever seguro),
`.claude/skills/autenticacao` (login/sessão/ownership) e
`.claude/skills/revisar-seguranca` (detectivo — auditar o que foi escrito).
O **o que é bloqueado automaticamente** está em `11-guards.md`.

## Classifique o projeto antes de escrever a primeira linha

A maioria dos projetos ARCOM é **backoffice interno**. Isso muda o desenho de
autenticação e o que é "normal" ver no código — decida antes de codar:

| Classificação | Sinais | Implicação |
|---|---|---|
| **INTERNA** | nome tipo "painel/gestão/backoffice/admin"; sem cadastro público; usuário é colaborador da ARCOM | login por conta corporativa/convite, **sem** signup público; se existir SSO, não pode coexistir rota de login por senha "por atalho" — isso é bypass do SSO |
| **EXTERNA** | tem cadastro público, checkout, landing de captação de lead | signup público é esperado; mesmo assim, sem segredo/dado de outro cliente vazando por troca de id na URL |
| **MISTA** | uma parte pública (catálogo) + uma parte de gestão (pedidos) | trate cada área com a régua acima — documente qual rota é qual |

## Escala de severidade (não infle, não minimize)

| Severidade | Exemplo |
|---|---|
| **CRÍTICA** | escalonar privilégio sem autenticação válida; segredo em bundle público; token com <128 bits de entropia em uso ativo |
| **ALTA** | webhook sem validar assinatura; rate limit efetivamente contornável; IDOR (acessar recurso de outro usuário trocando id) |
| **MÉDIA** | `.env` commitado só com valor público; CSP com `unsafe-inline`; token em `sessionStorage` |
| **BAIXA** | PII em log sem máscara; hash fraco em identificador não-crítico |
| **INFO** | arquivo de debug esquecido no repo |

**Princípio:** tabela vazia (`200 OK`, `[]`) não é vulnerabilidade — é controle
de acesso funcionando. `CORS *` num endpoint que já exige sessão válida é
severidade baixa, não crítica — o que barra o acesso é a sessão, não o CORS.
Na dúvida entre classificar e simplesmente perguntar, pergunte
("isso é comportamento esperado ou é falha?") em vez de inflar o achado.

## Mapa de referência (OWASP Top 10 + extras do padrão ARCOM)

`A01 Quebra de controle de acesso` · `A02 Configuração insegura` ·
`A03 Falha de supply chain` · `A04 Falha criptográfica` · `A05 Injeção` ·
`A06 Design inseguro` · `A07 Falha de autenticação` ·
`A08 Falha de integridade de dados` · `A09 Falha de log/alerta` ·
`A10 Tratamento incorreto de exceção` · `ARCOM-Padrão` (violação do que este
diretório `padroes/` define — ex.: lib fora do allowlist, backend criado sem
necessidade) · `LGPD` · `Segurança no cliente` (frontend).

## LGPD (contexto brasileiro — sempre que houver dado pessoal)

- **Dado sensível (Art. 5º, II):** origem racial/étnica, convicção religiosa,
  opinião política, dado de saúde, biométrico, genético, vida sexual — exige
  cuidado redobrado e base legal explícita (Art. 7º) pra tratar.
- **Direitos do titular (Art. 18):** acesso, correção, exclusão, portabilidade
  — se o projeto guarda dado de pessoa física, alguém vai pedir isso; o
  desenho de dados precisa suportar.
- **Nunca** logue e-mail/CPF/telefone/token em texto puro — mascare
  (`u***@dominio.com`) ou não logue.
- PII acessível sem autenticação é severidade **crítica** — em incidente real,
  há prazo legal de notificação à ANPD (72h, Art. 48).

## Quando um pedido esbarra aqui

Pare, explique o risco em uma frase, ofereça o caminho seguro — nunca entregue
a versão insegura "só pra funcionar" (mesma regra de `05-regras-para-a-ia.md`).
