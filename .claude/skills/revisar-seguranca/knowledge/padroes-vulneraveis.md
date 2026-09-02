# Padrões vulneráveis — stack ARCOM (Go + chi + gorm + React)

Cada item: código vulnerável → como detectar → correção. Adaptado de padrões
de auditoria genéricos pra esta stack específica (Go não é mencionado nas
fontes originais — os exemplos de Go abaixo foram escritos do zero pro
Padrão ARCOM).

## §Auth (sessão, ownership, CSRF, rate limit de login)

### Bearer/cookie aceito sem validar
```go
// VULNERÁVEL: só confere que o header existe, nunca valida o token
cookie, err := r.Cookie("sessao")
if err == nil {
    next.ServeHTTP(w, r) // segue mesmo sem checar validarToken(cookie.Value)
}
```
**Detecção:** middleware que lê `Cookie`/`Authorization` mas não chama uma
função de validação de assinatura/expiração antes de `next.ServeHTTP`.
**Fix:** sempre `validarToken(...)` (ou equivalente) antes de deixar passar —
ver skill `autenticacao`.

### IDOR — falta checagem de posse
```go
// VULNERÁVEL: busca pelo id da URL, nunca confere se pertence ao usuário logado
pedido, _ := service.Buscar(ctx, chi.URLParam(r, "id"))
json.NewEncoder(w).Encode(pedido)
```
**Fix:**
```go
pedido, err := service.Buscar(ctx, id)
if err != nil { return err }
if pedido.UsuarioID != usuarioDoContexto(ctx).ID {
    return servidor.SemPermissao()
}
```

### Mass assignment
```go
// VULNERÁVEL: cliente controla todos os campos, incluindo "role"
var entrada map[string]interface{}
json.NewDecoder(r.Body).Decode(&entrada)
db.Model(&usuario).Updates(entrada) // atacante manda {"role": "admin"}
```
**Fix:** DTO explícito com só os campos permitidos, ou
`db.Model(&usuario).Select("nome", "bio").Updates(entrada)`.

### Login por senha coexistindo com SSO obrigatório
Se o projeto usa SSO corporativo, uma rota `POST /api/v1/sessao/login` que
aceita e-mail/senha **sem** exigir o mesmo domínio/política do SSO é bypass —
ALTA, mesmo que a rota "só" exista pra um usuário de teste.

### CSRF sem guard
Sessão em cookie sem checar `Origin`/`Sec-Fetch-Site` em métodos que mudam
estado = qualquer site pode disparar a ação usando o cookie do usuário
logado. Ver `internal/servidor/middleware.go` (`origemPermitida`) — todo
projeto tem que ter isso.

### Entropia insuficiente
```go
// VULNERÁVEL: previsível
token := fmt.Sprintf("%d", rand.Intn(999999)) // math/rand
```
**Fix:** `crypto/rand`, ≥128 bits (`make([]byte, 16)` + `hex.EncodeToString`).

### Comparação não constante
```go
if token == tokenEsperado { ... } // vaza timing
```
**Fix:** `subtle.ConstantTimeCompare([]byte(token), []byte(tokenEsperado)) == 1`.

## §Config (secrets, allowlist, headers, container)

### Segredo em variável pública do Vite
```
VITE_API_SECRET=sk_live_xxxxx   # CRÍTICO — vai pro bundle JS, legível no DevTools
```
Qualquer `VITE_*` com `KEY`/`SECRET`/`TOKEN`/`PASSWORD` no nome e valor que
parece credencial real é achado crítico automático.

### Dependência fora do allowlist
`package.json`/`go.mod` com lib que não está em `01-stack-permitida.md` —
reportar como `ARCOM-Padrao`, severidade proporcional ao risco da lib (uma
lib de criptografia própria é mais grave que um componente de UI a mais).

### Container rodando como root / sem `.dockerignore`
Ver `04-docker-deploy.md` — `USER node`/`USER appuser` tem que estar presente;
`.dockerignore` tem que cobrir `.env`, `.env.*`, `node_modules`, binário.

### Headers de segurança ausentes
Resposta HTTP (backend) ou do nginx do frontend sem
`X-Content-Type-Options`/`X-Frame-Options`/`Referrer-Policy`/CSP — checar
`internal/servidor/middleware.go` e `nginx.conf.template` (blocos `add_header`).

## §Injeção (SQLi, XSS, validação, SSRF)

### SQL concatenado
```go
// VULNERÁVEL
db.Raw("SELECT * FROM produtos WHERE nome = '" + nome + "'")
```
**Fix:** `db.Raw("SELECT * FROM produtos WHERE nome = ?", nome)` ou
`db.Where("nome = ?", nome)`.

### `dangerouslySetInnerHTML` sem sanitização
```tsx
<div dangerouslySetInnerHTML={{ __html: comentarioDoUsuario }} /> // VULNERÁVEL
```
**Fix:** sanitizar com `dompurify` num componente único, ou não renderizar
HTML de usuário.

### Validação ausente ou incompleta
Handler que faz `json.NewDecoder(r.Body).Decode(&dto)` sem chamar
`validate.Struct(dto)` depois — campo que devia ser obrigatório/formatado
passa cru pro banco.

### SSRF
```go
// VULNERÁVEL: URL de destino vem do cliente
resp, _ := http.Get(r.URL.Query().Get("url"))
```
**Fix:** allowlist de host fixo; nunca aceitar URL arbitrária do cliente pra
chamada server-to-server.

## §Dados/LGPD (log, PII, rate limit)

### PII em log sem máscara
```go
log.Info("login", "email", usuario.Email) // BAIXA-MEDIA — email em log plano
```
**Fix:** tipo com `LogValue()` mascarando (`u***@dominio.com`), ou não logar
o campo.

### Rate limit fail-open
```go
// VULNERÁVEL: erro do limitador libera a requisição
if err := limitador.Checar(id); err != nil {
    next.ServeHTTP(w, r) // deveria negar, não deixar passar
    return
}
```
**Fix:** erro do limitador = **negar** (fail-closed), nunca liberar.

### Confiança errada em `X-Forwarded-For`
Backend exposto direto (sem o proxy do Vite no meio) confiando em XFF sem
saber quantos hops existem = qualquer cliente forja o próprio "IP" e escapa
do rate limit. Ver a nota de `ClientIPFromXFFTrustedProxies` em
`03-backend.md` — só é seguro porque existe exatamente 1 proxy confiável no
caminho.

### Dado sensível (LGPD Art. 5º-II) sem tratamento especial
Campo de saúde, biometria, orientação sexual, opinião política/religiosa
tratado como campo comum (sem controle de acesso extra, sem base legal
documentada) — reportar como `LGPD`, severidade proporcional à exposição.
