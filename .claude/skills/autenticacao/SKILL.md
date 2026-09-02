---
name: autenticacao
description: Fluxo de login/sessão/logout e checagem de posse (ownership) na stack ARCOM — cookie HttpOnly, sem token no frontend. Use quando o usuário pedir login, cadastro, "área restrita", permissão, ou qualquer tela/endpoint que só usuário autenticado pode acessar.
---

# Autenticação (login/sessão/ownership)

Antes de escrever, leia `padroes/02-frontend.md` (seção Sessão),
`padroes/03-backend.md` (seção Sessão/auth) e a skill `seguranca`. Só crie
isto se o pedido exigir mesmo — ver `07-frontend-primeiro.md` (login sozinho
já é motivo suficiente pra ter backend).

## Classifique antes de desenhar (ver `10-seguranca.md`)

A maioria dos projetos ARCOM é **backoffice interno**: login por conta
corporativa, sem cadastro público. Se o projeto já usa SSO da empresa, **não**
crie login por senha "de atalho" ao lado — isso é bypass do SSO e conta como
achado de severidade alta numa auditoria.

## O modelo: sessão em cookie, nunca token no frontend

- No login (`POST /api/v1/sessao`), o backend valida credencial e responde
  definindo o cookie:
  ```go
  http.SetCookie(w, &http.Cookie{
      Name:     "sessao",
      Value:    token, // JWT ou opaco — ver abaixo
      HttpOnly: true,
      Secure:   true,
      SameSite: http.SameSiteLaxMode,
      Path:     "/api",
      MaxAge:   int((15 * time.Minute).Seconds()),
  })
  ```
- O frontend **nunca** lê esse cookie nem guarda nada em `localStorage`/
  `sessionStorage` (bloqueado por ESLint). `core/api.ts` já manda o cookie
  sozinho (`withCredentials: true`); `core/sessao.tsx` só pergunta
  `GET /api/v1/sessao` pra saber quem está logado.
- Logout (`POST /api/v1/sessao/logout`) apaga o cookie (`MaxAge: -1`) e — se
  usar refresh/denylist — revoga o token no backend também.

## Token: JWT ou opaco?

- **Opaco** (string aleatória, `crypto/rand`, ≥128 bits) + registro no
  Postgres/Redis é mais simples de revogar (apaga o registro = sessão morta
  na hora). Prefira isto pra maioria dos projetos.
- **JWT**, se precisar (ex.: microsserviço lendo o mesmo token): HS256 com
  segredo ≥32 bytes, **valide o algoritmo explicitamente**
  (`jwt.WithValidMethods([]string{"HS256"})` — bloqueia "alg confusion"),
  `exp` curto (15 min), `iss`/`aud` conferidos. Revogação antes do `exp`
  precisa de denylist de `jti` no Redis — sem isso, "deslogar" não invalida
  o token que já foi emitido.

## Middleware de autenticação (backend)

```go
func (s *Servidor) exigirSessao(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cookie, err := r.Cookie("sessao")
        if err != nil {
            s.escreverProblema(w, r, http.StatusUnauthorized, "Não autenticado",
                "Faça login para continuar.", "token_ausente", nil)
            return
        }
        usuario, err := validarToken(cookie.Value)
        if err != nil {
            s.escreverProblema(w, r, http.StatusUnauthorized, "Sessão inválida",
                "Sua sessão expirou. Faça login novamente.", "token_expirado", nil)
            return
        }
        next.ServeHTTP(w, r.WithContext(comUsuario(r.Context(), usuario)))
    })
}
```

Aplique num `chi.Router` group (`v1.Group(func(g chi.Router) { g.Use(s.exigirSessao); ... })`),
nunca rota por rota manualmente (fácil esquecer uma).

## Ownership — a parte que mais falha em revisão

Autenticado ≠ autorizado. Toda rota que recebe um id
(`GET /api/v1/pedidos/{id}`) tem que checar que o recurso **pertence** a quem
está logado (ou que o papel dele permite ver qualquer um):

```go
pedido, err := service.Buscar(ctx, id)
if err != nil { return err }
if pedido.UsuarioID != usuarioAutenticado.ID {
    return servidor.SemPermissao() // não NaoEncontrado — mas também não vaze detalhe
}
```

Trocar o id na URL pra acessar o recurso de outro usuário (IDOR) é o achado
mais comum de auditoria em app interno — checar isso não é opcional.

## Frontend — rota protegida

```tsx
// rotas.tsx
<Route element={<RotaProtegida />}>
  <Route path="/painel" element={<Painel />} />
</Route>
```

`RotaProtegida` (`core/sessao.tsx`) redireciona pro login se `useSessao()` não
tiver usuário. A **permissão** (o que o usuário pode ver dentro do painel)
nunca é persistida no cliente — vem sempre da resposta mais recente do
backend; a UI pode esconder um botão por conveniência, mas a proteção real é
a rota do backend recusar a ação.

## Checklist antes de considerar pronto

- [ ] Cookie `HttpOnly; Secure; SameSite=Lax`, nunca token em `localStorage`.
- [ ] Middleware de sessão aplicado por grupo de rotas, não rota a rota.
- [ ] Toda rota com id de recurso confere ownership.
- [ ] Rate limit próprio (mais agressivo) no login.
- [ ] Mensagem de erro de login genérica (não diz se foi e-mail ou senha).
- [ ] Logout limpa o cookie e revoga o token no backend (se opaco/denylist).
