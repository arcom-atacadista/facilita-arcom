---
name: seguranca
description: Checklist de segurança contra os ataques mais comuns (injeção, XSS, auth quebrada, vazamento de segredo, CSRF, SSRF, mass assignment, rate limit contornável) mapeado pra stack ARCOM (React + Go + Postgres/Redis + cookie de sessão). Use SEMPRE que criar login, formulário, endpoint que recebe dado do usuário, ou qualquer coisa que envolva backend + frontend juntos.
---

# Segurança — os ataques mais comuns e como barrar nesta stack

Aplique isto ao gerar qualquer código que receba dado do usuário, autentique, ou
ligue frontend a backend. Vale mais que conveniência: na dúvida, feche.
Fluxo completo de login/sessão: skill `autenticacao`. Auditar código já
escrito: skill `revisar-seguranca`. Classificação/severidade: `10-seguranca.md`.

## 1. Injeção de SQL

- **Sempre** use o `gorm` ou `pgx` com parâmetros. **Nunca** monte SQL
  concatenando string com dado do usuário (`"WHERE nome='"+nome+"'"` = brecha,
  nem com `fmt.Sprintf` dentro de `Raw`/`Exec`).
- `db.Where("email = ?", email)` — certo. `fmt.Sprintf` numa query — errado.
- Coluna/ordenação dinâmica vinda do cliente (`?ordenar=nome`) nunca é
  interpolada direto — passa por **allowlist** (`map[string]string`).

## 2. XSS (script injetado na tela)

- O React já escapa texto por padrão — mantenha assim.
- **Nunca** use `dangerouslySetInnerHTML` com conteúdo que veio do usuário/API
  (bloqueado por ESLint — `react/no-danger`). Se for inevitável (editor de
  texto rico), só via um componente único que sanitiza com `dompurify`.
- Não injete HTML de terceiros sem sanitizar.

## 3. Autenticação e sessão

- Senha: hash com `bcrypt`, **cost 12** (não o `DefaultCost`). **Nunca** guarde
  ou logue senha em texto puro.
- Sessão: cookie `HttpOnly; Secure; SameSite=Lax`, definido pelo backend no
  login. O frontend **nunca** lê nem guarda o token (nada de
  `localStorage`/`sessionStorage` — bloqueado por ESLint). Ver skill
  `autenticacao`.
- Token/código aleatório (reset de senha, convite, sessão): `crypto/rand`
  (nunca `math/rand` — previsível), com ≥128 bits de entropia. Comparação de
  token/HMAC sempre com `crypto/subtle.ConstantTimeCompare`, nunca `==`
  (timing attack).
- Valide a sessão num **middleware** que protege as rotas privadas.

## 4. Controle de acesso quebrado (o mais comum e o mais grave)

- Cheque autorização no **backend, em toda rota protegida** — nunca confie que o
  frontend "escondeu o botão".
- Cheque que o usuário é **dono** do recurso que pede (o usuário A não acessa o
  pedido do usuário B trocando o id na URL — IDOR).
- **Mass assignment:** nunca faça `db.Model(&x).Updates(corpoDoRequest)` com o
  corpo inteiro do request. Use DTO explícito ou `Select("campo1", "campo2")`
  — um campo que o cliente não deveria poder mudar (ex.: `role`, `admin`)
  nunca pode vir do mesmo `Updates` que o resto do formulário.

## 5. Vazamento de segredo

- Segredo (chave de API, senha de banco, segredo de assinatura de sessão) vive
  **só no backend**, via ambiente. **Nunca** no bundle do frontend (só
  `VITE_*` público — qualquer `VITE_*_KEY`/`_SECRET`/`_TOKEN`/`_PASSWORD` é
  crítico) nem commitado. O `.dockerignore` já impede o `.env` de entrar na
  imagem.
- Não logue token, senha nem dado pessoal (CPF, e-mail, telefone) em texto
  puro — mascare (`u***@dominio.com`) ou não logue.
- Em produção, secrets sem fallback (`${VAR:?}`) e HTTPS no proxy da frente.

## 6. Front + back juntos (o caso deste pedido)

- Frontend fala com backend **só em `/api/v1`** (mesma origem, via proxy do
  nginx em produção / do Vite em dev) — isso evita CORS e o risco de abrir
  CORS pra qualquer origem. **Não** habilite `Access-Control-Allow-Origin: *`
  (o `cors: false` do `vite.config.ts` em dev, e o nginx não adicionando
  header de CORS nenhum em produção, já garantem isso).
- Sessão em cookie, não header → **CSRF entra em jogo**: todo
  `POST/PUT/PATCH/DELETE` passa pelo guard de origem do backend (checa
  `Origin` contra o host da requisição, ver `internal/servidor/middleware.go`).
- O backend já responde com headers de segurança por padrão (ver
  `internal/servidor/middleware.go`); mantenha.

## 7. Validação de entrada

- Valide **tudo** que vem do cliente no backend com `go-playground/validator`
  + `DisallowUnknownFields()`. Validação no formulário (front, com `zod`) é só
  conveniência, não segurança.

## 8. SSRF / chamadas a APIs externas

- Se o backend chama uma API externa, **não** monte a URL de destino com input
  cru do usuário (ele poderia te fazer chamar um endereço interno). Use base
  fixa do `config` + parâmetros validados; `http.Client` sempre com `Timeout`.
- **RabbitMQ:** `RABBITMQ_URL`, credencial e nome de fila/exchange vêm do
  time de infra — nunca invente/assuma um nome de fila, e nunca monte a
  conexão com dado do cliente. Ver `01-stack-permitida.md`.

## 9. Força bruta / abuso / rate limit

- Em login e endpoints sensíveis, limite tentativas (`httprate`, com limite
  mais agressivo que o global). Mensagem de erro genérica ("e-mail ou senha
  inválidos"), sem dizer qual dos dois errou.
- **Rate limit tem que falhar fechado.** Se o Redis (ou o que conta as
  tentativas) estiver indisponível, **negue** a requisição — nunca libere
  geral "pra não travar o app". Fail-open em rate limit é a forma mais comum
  de contorná-lo.
- Não confie no primeiro IP de uma lista de proxy não confiável pra
  identificar quem está sendo limitado — ver a nota sobre
  `X-Forwarded-For`/`ClientIPFromXFFTrustedProxies` em `03-backend.md`.

## 10. Configuração e dependências

- Container roda non-root (já vem assim). Não rode como root.
- Erro pro cliente é genérico — **nunca** devolva stack trace / detalhe
  interno (`err.Error()` no corpo é proibido, ver `09-contrato-api.md`).
- `.npmrc` com `ignore-scripts=true` e `go.sum` já barram dependência
  maliciosa — não afrouxe. Mantenha as libs do `01-stack-permitida.md`.
- `govulncheck`/`golangci-lint` (backend) e `pnpm audit` (frontend) fazem
  parte de `scripts/verificar.sh` — rode antes de considerar algo pronto.

## Quando algo esbarrar aqui

Pare, explique o risco em uma frase e ofereça o caminho seguro — não entregue a
versão insegura "só pra funcionar".
