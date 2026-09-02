# 12 — Gateway de dados ARCOM

A ARCOM tem um Gateway interno (`https://kabana-api.arcom.com.br`) que expõe,
somente leitura, os dados reais da empresa — representantes, clientes,
débitos, vendas, promoções, roteiros, RH etc. — como API REST. Qualquer
projeto que precisar de dado real da empresa (não fictício) consome esse
Gateway em vez de inventar uma fonte própria.

Referência completa (schema de cada dataset, campos, tipos, exemplos de
request/response): `padroes/gateway-arcom/api.md` (versão legível) e
`padroes/gateway-arcom/openapi.yaml` (spec OpenAPI completa). **Não invente
dataset, campo ou endpoint que não esteja em algum desses dois arquivos.**

## Convenções da API (resumo — o detalhe está em `api.md`)

- Auth: header `X-API-Key`. Todas as rotas são `GET`.
- Filtro: `campo=valor` (vários valores separados por vírgula = OR);
  intervalo com `campo.gte=`, `campo.lte=`, `campo.lt=`.
- Busca textual livre: `q=` nos campos listados em "Busca textual" de cada
  dataset.
- Agregação: `GET /v1/{dataset}/agregado?por=&metrica=&top=&incluir_campos=`,
  onde `metrica` é `count` | `sum:campo` | `avg:campo` | `min:campo` |
  `max:campo` (nomes em inglês, confirmados no `openapi.yaml` — não usar
  `soma`/`media`).
- Série temporal: sufixo `:mes`/`:semana`/`:dia`/`:ano` no campo usado em
  `por=`.
- Vários datasets terminam com `> Semântica a revisar (preencher)` — o
  significado de campos de código/status ainda não está documentado.
  Não adivinhe: avise e confirme com quem pediu antes de usar esse campo.

## A API key é segredo de produção — nunca do chat

**Este repositório não tem, e nunca deve ter, a `X-API-Key` real do
Gateway.** Ela só existe no ambiente de produção, provisionada pelo time de
TI da ARCOM. Isso significa, na prática:

- O chat/IA pode e deve escrever a integração inteira (client do Gateway,
  endpoint que consome e devolve os dados) sem precisar da chave — ela entra
  em runtime via variável de ambiente (`config.Obrigatorio`), nunca
  hardcoded, nunca pedida ao usuário pra colar no código.
- **Antes de subir pra produção qualquer feature que dependa do Gateway**,
  quem estiver implementando precisa acionar o time de TI/infraestrutura da
  ARCOM pra provisionar a `X-API-Key` no servidor. Sem isso, a chamada real
  ao Gateway não funciona — só o código fica pronto.
- Em ambiente local/dev, sem a chave, a chamada ao Gateway vai falhar
  (401/403) — isso é esperado; não é bug da implementação.

## Integrando no backend

- A chamada ao Gateway é sempre feita pelo **backend Go**, nunca pelo
  frontend (ver `03-backend.md` e `10-seguranca.md`).
- Um pacote próprio pro client do Gateway (ex.:
  `backend/internal/gatewayarcom/client.go`); o recurso do projeto que
  expõe o dado ao frontend segue a estrutura normal de
  `handler.go`/`service.go`/`repo.go` (skill `novo-endpoint`) e devolve só
  os campos que a tela precisa — nunca repasse o payload do Gateway inteiro.
- Erro do Gateway (timeout, 4xx/5xx) é tratado como erro de dependência
  externa, no formato de `09-contrato-api.md` — nunca estoura sem
  tratamento pro cliente final.

## Exemplo de implementação

O client não sabe nada de negócio — recebe **dataset** (o "index" — ex.
`debitos`, `cliente-ivendas-v2`) e os filtros já montados, e devolve o JSON
crú do Gateway. Quem decide dataset e campos é o `service.go` do recurso,
com base no que foi pedido.

`backend/internal/gatewayarcom/client.go`:

```go
// Package gatewayarcom é o único lugar do projeto que fala com o Gateway
// de dados da ARCOM. Nada de negócio aqui — só HTTP + auth.
package gatewayarcom

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://kabana-api.arcom.com.br"

type Cliente struct {
	apiKey string
	http   *http.Client
}

// NovoCliente recebe a API key já lida de config.Obrigatorio no boot —
// o client nunca lê variável de ambiente sozinho.
func NovoCliente(apiKey string) *Cliente {
	return &Cliente{apiKey: apiKey, http: &http.Client{Timeout: 10 * time.Second}}
}

// Consultar faz GET /v1/{dataset} com os filtros dados. dataset é o "index"
// (ex.: "debitos"); filtros já vem no formato da query (campo=valor,
// campo.gte=, q=, limite= etc. — ver padroes/gateway-arcom/api.md).
func (c *Cliente) Consultar(ctx context.Context, dataset string, filtros url.Values) ([]byte, error) {
	return c.get(ctx, fmt.Sprintf("%s/v1/%s?%s", baseURL, dataset, filtros.Encode()))
}

// Agregar faz GET /v1/{dataset}/agregado — mesma ideia, para métricas/série
// temporal (por=, metrica=, top=, incluir_campos=).
func (c *Cliente) Agregar(ctx context.Context, dataset string, filtros url.Values) ([]byte, error) {
	return c.get(ctx, fmt.Sprintf("%s/v1/%s/agregado?%s", baseURL, dataset, filtros.Encode()))
}

func (c *Cliente) get(ctx context.Context, requestURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gateway arcom: %w", err)
	}
	defer resp.Body.Close()

	corpo, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("gateway arcom respondeu %d: %s", resp.StatusCode, corpo)
	}
	return corpo, nil
}
```

Uso num `service.go` de recurso — aqui um exemplo pra "débitos de um
cliente", que é quem decide o dataset (`debitos`) e os filtros
(`cliente`, `limite`):

```go
func (s *Service) DebitosDoCliente(ctx context.Context, cliente string) ([]Debito, error) {
	filtros := url.Values{}
	filtros.Set("cliente.keyword", cliente)
	filtros.Set("limite", "20")

	corpo, err := s.gateway.Consultar(ctx, "debitos", filtros)
	if err != nil {
		return nil, fmt.Errorf("consultar debitos: %w", err)
	}

	var resposta struct {
		Itens []struct {
			Cliente      string  `json:"cliente"`
			ValorLiquido float64 `json:"vlr_liquido_deb"`
			DataVencto   string  `json:"data_vencto"`
		} `json:"itens"` // formato real da envelope está em api.md/openapi.yaml
	}
	if err := json.Unmarshal(corpo, &resposta); err != nil {
		return nil, fmt.Errorf("decodificar resposta do gateway: %w", err)
	}

	// devolve só os campos que a tela usa — nunca o payload do gateway inteiro
	debitos := make([]Debito, 0, len(resposta.Itens))
	for _, it := range resposta.Itens {
		debitos = append(debitos, Debito{
			Cliente:    it.Cliente,
			Valor:      it.ValorLiquido,
			Vencimento: it.DataVencto,
		})
	}
	return debitos, nil
}
```

**O envelope de resposta real (nome do campo de lista, paginação etc.) vem
de `padroes/gateway-arcom/openapi.yaml` — o `itens` acima é ilustrativo, não
copie sem confirmar contra a spec.**

E no boot (`cmd/server/main.go`), a chave só é lida uma vez:

```go
chave, err := config.Obrigatorio("GATEWAY_ARCOM_API_KEY")
if err != nil {
	log.Fatal(err) // falha rápido no boot — nunca no meio de um request
}
gateway := gatewayarcom.NovoCliente(chave)
```

Resumindo a lógica pra quem for pedir uma feature: você (o dev/usuário) diz
**qual dado quer buscar** (dataset + campos/filtros); o código monta o
`dataset` (index) certo e manda a chamada — a `X-API-Key` é injetada
automaticamente a partir do ambiente, você nunca precisa saber ou colar
ela em lugar nenhum.
