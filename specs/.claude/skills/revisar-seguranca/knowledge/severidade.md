# Severidade — calibração (não infle, não minimize)

Níveis: `CRITICA | ALTA | MEDIA | BAIXA | INFO`. IDs por severidade:
`C1..Cn, A1..An, M1..Mn, B1..Bn, I1..In`.

## Tabela de calibração (cenários reais, stack ARCOM)

| Cenário | CVSS aprox. | Severidade |
|---|---|---|
| Endpoint aceita `usuario_id` do corpo/query e usa pra decidir dono do recurso (privilege escalation) | 9.1 | CRITICA |
| Handler que grava no banco sem middleware de autenticação | 8.6 | CRITICA |
| Token de sessão/convite com `math/rand` (PRNG previsível) em uso ativo | 9.0 | CRITICA |
| Segredo (chave de API, segredo de assinatura) em variável `VITE_*` (vai pro bundle público) | 9.1 | CRITICA |
| SQL montado com `fmt.Sprintf`/concatenação (SQLi confirmado) | 9.3 | CRITICA |
| Webhook que processa payload sem validar assinatura | 7.1–7.3 | ALTA |
| Rate limit contornável (fail-open no erro do limitador, ou key previsível) | 7.0–7.5 | ALTA |
| IDOR — troca de id na URL acessa recurso de outro usuário | 7.5–8.1 | ALTA |
| Mass assignment — campo sensível (`role`, `admin`) atualizável pelo mesmo `Updates` do formulário | 7.5 | ALTA |
| Login por senha coexistindo com SSO obrigatório (bypass) | 7.5 | ALTA |
| Token de sessão em `sessionStorage`/`localStorage` | 5.9–6.1 | MEDIA |
| `.env` commitado só com valor de dev/placeholder (sem segredo real) | 5.3 | MEDIA |
| CSP com `unsafe-inline`/`unsafe-eval` | 5.4 | MEDIA |
| Dependência com CVE conhecido mas sem caminho de exploração alcançável no código (`govulncheck` não confirma reachability) | 3.1–4.0 | BAIXA |
| PII (e-mail/CPF/telefone) em log sem máscara | 3.1 | BAIXA |
| Arquivo de debug/comentário com dado de teste esquecido | 2.0 | INFO |

## DO / DON'T classificar como CRÍTICO

**Classifique como CRÍTICO:**
- Função que permite escalonar privilégio sem checagem de auth adequada.
- Segredo em bundle público que quebra uma camada de segurança real.
- Token gerado com <128 bits de entropia efetiva, em uso ativo.
- Endpoint administrativo exposto sem qualquer autenticação.

**NÃO classifique como CRÍTICO:**
- `.env.example` com placeholder óbvio (`troque-em-producao`).
- Endpoint `GET /api/health` sem autenticação (é liveness, por design).
- `CORS *` num endpoint que já exige sessão válida pra fazer qualquer coisa
  (a sessão é a barreira real — isso é BAIXA, não CRÍTICA).
- Tabela retornando `200 []` (RLS/authz funcionando, não vazamento).

## Mapa CWE (as mais comuns nesta stack)

- CWE-306: Falta de autenticação para função crítica
- CWE-639: Bypass de autorização por chave controlada pelo usuário (IDOR)
- CWE-269: Gestão de privilégio inadequada
- CWE-330/338: Aleatoriedade insuficiente / PRNG fraco
- CWE-352: CSRF
- CWE-345/347: Verificação insuficiente de autenticidade (webhook sem HMAC)
- CWE-798: Uso de credencial hardcoded
- CWE-540: Informação sensível incluída no código-fonte
- CWE-284: Controle de acesso impróprio
- CWE-307: Restrição insuficiente de tentativas de autenticação
- CWE-79: XSS
- CWE-915: Mass assignment
- CWE-89: SQL Injection
- CWE-918: SSRF
- CWE-601: Redirecionamento aberto

## Categorias (OWASP Top 10 + extras do padrão)

`A01 Quebra de controle de acesso` · `A02 Configuração insegura` ·
`A03 Falha de supply chain` · `A04 Falha criptográfica` · `A05 Injeção` ·
`A06 Design inseguro` · `A07 Falha de autenticação` ·
`A08 Falha de integridade de dados` · `A09 Falha de log/alerta` ·
`A10 Tratamento incorreto de exceção` · `ARCOM-Padrao` (viola algo definido em
`padroes/`) · `LGPD` · `Cliente` (frontend-specific).

## Schema JSON de finding (saída canônica — guards e este skill usam o mesmo formato)

```json
{
  "id": "C1",
  "titulo": "Título curto e descritivo",
  "severidade": "CRITICA | ALTA | MEDIA | BAIXA | INFO",
  "descricao": "O que está errado, tecnicamente",
  "impacto": "O que um atacante consegue fazer de concreto",
  "correcao": "Passo a passo com código",
  "arquivo": "caminho/relativo/do/arquivo.go",
  "linha": 42,
  "trecho": "Trecho real do código (máx. 5 linhas) — nunca genérico",
  "categoria": "A01 - Quebra de controle de acesso",
  "cwe": "CWE-639",
  "cvss": 8.1,
  "classificacao_app": "INTERNA | EXTERNA | MISTA",
  "precisa_esclarecimento": false
}
```

Regras: ordenar por `cvss` desc; `linha: 0` quando o achado é **ausência** de
algo (ex.: nenhum middleware de auth no grupo de rotas); nunca incluir achado
que caiu em `knowledge/falsos-positivos.md`.

## Status de comparação com relatório anterior

`CONFIRMADO` (mesma severidade) · `MITIGADO` (corrigido) · `AGRAVADO`
(regressão, piorou) · `NOVO` (não existia antes).
