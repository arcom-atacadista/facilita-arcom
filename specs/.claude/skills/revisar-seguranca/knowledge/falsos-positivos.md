# Falsos positivos — não inflar o relatório

Cada linha aqui é um "parece vulnerabilidade, mas não é" — aplique antes de
reportar. Se um achado se encaixa numa linha desta tabela, **não reporte**
(ou reporte como INFO, se ainda achar que vale registrar).

| Cenário | Por que NÃO é vulnerabilidade |
|---|---|
| `GET /api/health` sem autenticação | É liveness, por design — não deveria ter auth |
| `GET /api/ready` expondo só `{"status":"ok"}` ou 503 | Readiness não vaza dado sensível |
| `.env.example` com placeholder óbvio (`troque-em-producao`, `dev-secret`) | Não é segredo real |
| Tabela retornando `200 []`/`200 {}` pra usuário sem permissão | Controle de acesso funcionando — ausência de dado, não vazamento |
| `CORS` desabilitado (`cors: false`) no Vite | É o comportamento correto do modelo same-origin do padrão, não brecha |
| `CORS *` num endpoint que já exige sessão válida pra fazer qualquer coisa | A sessão é a barreira real; CORS aberto aqui é BAIXA, não CRÍTICA |
| Endpoint público de fato público por design (ex.: página de evento pública, `/e/:slug`) | Checar se o dado exposto é mesmo público antes de reportar |
| Dependência com CVE conhecido mas sem call site alcançável (`govulncheck` não confirma reachability) | Reportar como BAIXA/INFO, não ALTA — risco teórico, não caminho de exploração |
| Log de `request_id`, método, path, status — sem dado pessoal | Log operacional normal, não é PII |
| `gorm` gerando SQL parametrizado internamente (não é o código do projeto concatenando) | O ORM já é seguro por padrão — não reporte "risco de SQLi" genérico sem achar o ponto real de concatenação |

## Critério de 3 condições (aplique a achado de "segredo hardcoded")

Só reporte string suspeita como segredo real se **todas** as três forem
verdadeiras ao mesmo tempo:
1. O valor não é aleatório/gerado (`senha123`, `admin`) — não é UUID nem hash.
2. Está atribuído a algo cujo nome indica credencial
   (`password`, `secret`, `api_key`, `token`).
3. Existe evidência de que o valor é **usado em runtime** (não é só um
   comentário ou exemplo em teste isolado).

## Regra geral

Na dúvida entre "é vulnerabilidade" e "é comportamento esperado", marque
`precisa_esclarecimento: true` no finding e pergunte — inflar o relatório com
falso positivo é o que faz quem lê parar de confiar nele.
