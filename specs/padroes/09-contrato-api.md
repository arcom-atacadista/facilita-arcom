# 09 — Contrato de API (front ↔ back)

Formato único de resposta entre o frontend e o backend Go. Existe pra front e back nunca
precisarem "combinar" nada na hora — leem este arquivo e já sabem o formato. Front e back leem
este guia **antes** de `02-frontend.md` / `03-backend.md`, porque os dois dependem dele.

## Prefixo e versão

Rotas de negócio ficam sob `/api/v1`. `GET /api/health` e `GET /api/ready` ficam **fora** da
versão (a infra/orquestrador depende deles, não podem mudar com a API).

```
/api/health          liveness — sempre 200, nunca toca banco/rede
/api/ready           readiness — 200 se Postgres/Redis respondem, 503 se não
/api/v1/...          toda rota de negócio
```

## Sucesso

Sem envelope: a resposta **é** o recurso.

```http
GET /api/v1/produtos/42
200 OK
Content-Type: application/json

{ "id": 42, "nome": "Caixa de suco", "preco": 12.5 }
```

Lista: envelope só com paginação, nunca aninhar o recurso mais uma vez.

```json
{
  "itens": [ { "id": 1, "nome": "..." }, { "id": 2, "nome": "..." } ],
  "paginacao": { "pagina": 1, "porPagina": 20, "total": 137, "totalPaginas": 7 }
}
```

Query params de entrada: `pagina` (1-based, default `1`), `porPagina` (default `20`, máximo
`100` — o backend recusa valor maior, não trunca em silêncio).

`POST`/`PUT`/`PATCH` que criam ou alteram devolvem o recurso resultante (200 ou 201).
`DELETE` que dá certo devolve `204 No Content`, corpo vazio — nunca `200 {}`.

## Erro — `application/problem+json` (RFC 9457)

Todo erro (400–599) sai neste formato, com `Content-Type: application/problem+json`:

```json
{
  "type": "about:blank",
  "title": "Sessão expirada",
  "status": 401,
  "detail": "Sua sessão expirou. Faça login novamente.",
  "codigo": "token_expirado",
  "campos": { "email": "e-mail inválido" }
}
```

| Campo | Obrigatório | Regra |
|---|---|---|
| `type` | sim | `"about:blank"` por padrão — o projeto não versiona uma página de doc por erro |
| `title` | sim | resumo curto e genérico do tipo de erro (fixo por status, não muda por requisição) |
| `status` | sim | repete o status HTTP da resposta (facilita log/monitoramento que só olha o body) |
| `detail` | sim | texto em português que **pode** ir pro usuário. Nunca stack trace, nunca nome de tabela/coluna, nunca caminho de arquivo |
| `codigo` | sim | string estável em `snake_case` — **é o contrato de máquina**. O frontend decide comportamento (ex.: forçar logout) pelo `codigo`, nunca comparando o texto de `detail` |
| `campos` | só em validação (422) | mapa `nome_do_campo -> mensagem`, um erro por campo |

**Regra inegociável:** em erro 5xx, `detail` é sempre uma mensagem genérica
("Erro interno. Tente novamente.") — o erro real vai só pro log do servidor (`slog`), nunca
pro corpo da resposta. Devolver `err.Error()` no `detail` é vazamento de informação interna
(nome de tabela, driver do banco, caminho) e está proibido.

### Mapa de status

| Status | Quando |
|---|---|
| 400 | Input malformado (JSON inválido, tipo errado) |
| 401 | Não autenticado (sem sessão ou sessão inválida) |
| 403 | Autenticado, mas sem permissão para o recurso |
| 404 | Recurso não existe (ou existe mas não é do dono — nunca revele a diferença) |
| 405 | Método HTTP não existe para essa rota |
| 409 | Conflito (ex.: e-mail já cadastrado) |
| 422 | Validação semântica (campo com formato errado, regra de negócio) |
| 429 | Rate limit excedido |
| 500 | Erro interno não mapeado |

### Catálogo de `codigo` (baseline — cada feature pode adicionar os seus)

| `codigo` | Status | Uso |
|---|---|---|
| `token_ausente` | 401 | Sem cookie de sessão |
| `token_invalido` | 401 | Cookie presente, assinatura/formato inválido |
| `token_expirado` | 401 | Sessão expirou — front redireciona pro login |
| `sem_permissao` | 403 | Autenticado, sem acesso ao recurso |
| `nao_encontrado` | 404 | Recurso (ou rota) inexistente |
| `metodo_nao_permitido` | 405 | Método HTTP errado para a rota |
| `conflito` | 409 | Já existe / choque de estado |
| `validacao` | 422 | Um ou mais campos inválidos — vem com `campos` |
| `origem_invalida` | 403 | Guard de CSRF recusou (Origin/Sec-Fetch-Site não bate) |
| `limite_de_requisicoes` | 429 | Rate limit |
| `erro_interno` | 500 | Qualquer coisa não mapeada |

`401`/`403` **nunca** dizem qual dos dois errou entre "e-mail" e "senha" no login — mensagem
genérica ("e-mail ou senha inválidos"), sempre `codigo` único (não vaza que o e-mail existe).

## Datas

Sempre **ISO 8601 em UTC**, sufixo `Z`: `"2026-07-29T12:00:00Z"`. O backend Go grava e devolve
tudo em UTC (`time.Time` sem fuso embutido); o frontend converte pro fuso local só na hora de
exibir, com `dayjs` (plugins `utc`/`timezone`, ver `02-frontend.md`). Nunca serializar data em
outro formato (nada de `DD/MM/YYYY` vindo do backend).

## Do lado do frontend

`core/erro.ts` normaliza a resposta `application/problem+json` pro tipo interno da aplicação
(nomes em português, consistentes com o resto do código React):

```ts
export type Problema = {
  status: number;
  titulo: string;     // vem de `title`
  mensagem: string;    // vem de `detail` — o texto que pode ir pro usuário
  codigo: string;
  campos?: Record<string, string>;
};
```

Nenhuma tela compara `mensagem` por texto (nem `includes`, nem `===`) pra decidir fluxo — só
`codigo`. `mensagem` é só pra mostrar (toast/campo de formulário).

## Do lado do backend

Um único ponto de conversão erro→resposta (`internal/servidor/problema.go`, ver
`03-backend.md`). Nenhum handler escreve JSON de erro na mão — todos passam pelo helper, que
garante o `Content-Type` certo e nunca deixa `err.Error()` cru chegar no `detail` de um 5xx.
