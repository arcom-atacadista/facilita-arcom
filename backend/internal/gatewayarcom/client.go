// Package gatewayarcom é o único lugar do projeto que fala com o Gateway de
// dados da ARCOM. Nada de negócio aqui — só HTTP + auth.
//
// O Gateway é somente leitura (todas as rotas são GET) e é a fonte dos
// débitos reais da empresa. Ver system-design/padroes/12-gateway-arcom.md.
package gatewayarcom

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BaseURL é fixa, do padrão. Nunca montada a partir de input do cliente —
// é o que fecha SSRF nesta superfície (ver skill seguranca).
const BaseURL = "https://kabana-api.arcom.com.br"

// Dataset é o "index" do Gateway. Tipo próprio e conjunto fechado de
// propósito: dataset entra no caminho da URL, e aceitar string livre aqui
// deixaria alguém montar "../" e sair do /v1 pretendido.
type Dataset string

const (
	// DatasetDebitos é a carteira em atraso da empresa — a origem real das
	// dívidas, no lugar da planilha/CSV que o modelo antigo importava.
	DatasetDebitos Dataset = "debitos"

	// DatasetDisparosNines é o histórico do que a Nines já disparou —
	// telefone, template, status, urlBoleto, dataDisparo.
	//
	// Vale insistir no que ele NÃO é: o Gateway inteiro é somente leitura
	// (todas as rotas são GET), então este dataset conta o que já foi
	// enviado, e não é um canal de envio. O Facilita ARCOM não tem, hoje, por
	// onde disparar mensagem — ver internal/disparo/canal.go.
	DatasetDisparosNines Dataset = "disparo-mensagem-nines"
)

func (d Dataset) valido() bool {
	switch d {
	case DatasetDebitos, DatasetDisparosNines:
		return true
	}
	return false
}

var (
	// ErrDatasetDesconhecido protege contra dataset fora do documentado —
	// 12-gateway-arcom.md: "Não invente dataset, campo ou endpoint".
	ErrDatasetDesconhecido = errors.New("dataset não previsto no padrão do Gateway")

	// ErrSemCredencial é o caso normal em dev: a X-API-Key só existe em
	// produção, provisionada pelo time de TI.
	ErrSemCredencial = errors.New("GATEWAY_ARCOM_API_KEY não configurada")

	// ErrNaoAutorizado é 401/403 do Gateway — chave ausente, errada ou sem
	// permissão para o dataset.
	ErrNaoAutorizado = errors.New("gateway recusou a credencial")
)

// ErroResposta é um status de erro devolvido pelo Gateway. O corpo fica
// disponível para o log, nunca para a resposta ao cliente final.
type ErroResposta struct {
	Status int
	Corpo  string
}

func (e *ErroResposta) Error() string {
	return fmt.Sprintf("gateway arcom respondeu %d", e.Status)
}

type Cliente struct {
	apiKey string
	base   string
	http   *http.Client
}

// NovoCliente recebe a API key já lida de config no boot — o client nunca lê
// variável de ambiente sozinho. Chave vazia é aceita: o client existe, e cada
// chamada devolve ErrSemCredencial. É assim que dá pra rodar em dev sem a
// credencial de produção, e a feature que depende dela falha com uma mensagem
// clara em vez de o processo não subir.
func NovoCliente(apiKey string) *Cliente {
	return &Cliente{
		apiKey: apiKey,
		base:   BaseURL,
		// Timeout sempre presente: sem ele uma chamada pendurada segura um
		// worker do backend indefinidamente.
		http: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Cliente) TemCredencial() bool { return c.apiKey != "" }

// ComBaseURL devolve uma cópia do client apontando para outro endereço.
//
// Existe para teste: em produção a base é sempre BaseURL, e é justamente por
// ela ser constante que esta superfície não tem SSRF. Um teste que aponta
// para httptest não muda essa garantia, porque o endereço nunca vem de input
// de cliente.
func (c *Cliente) ComBaseURL(base string) *Cliente {
	copia := *c
	copia.base = strings.TrimRight(base, "/")
	return &copia
}

// Consultar faz GET /v1/{dataset} com os filtros dados e devolve o JSON crú.
// Quem decide dataset, campos e filtros é o service do recurso.
func (c *Cliente) Consultar(ctx context.Context, dataset Dataset, filtros url.Values) ([]byte, error) {
	return c.get(ctx, dataset, "", filtros)
}

// Agregar faz GET /v1/{dataset}/agregado — métricas e série temporal
// (por=, metrica=, top=, incluir_campos=).
func (c *Cliente) Agregar(ctx context.Context, dataset Dataset, filtros url.Values) ([]byte, error) {
	return c.get(ctx, dataset, "/agregado", filtros)
}

func (c *Cliente) get(ctx context.Context, dataset Dataset, sufixo string, filtros url.Values) ([]byte, error) {
	if !dataset.valido() {
		return nil, fmt.Errorf("%w: %q", ErrDatasetDesconhecido, dataset)
	}
	if c.apiKey == "" {
		return nil, ErrSemCredencial
	}

	// url.JoinPath escapa o segmento; junto com o conjunto fechado de
	// Dataset, o caminho não tem como sair de /v1.
	alvo, err := url.JoinPath(c.base, "v1", string(dataset)+sufixo)
	if err != nil {
		return nil, fmt.Errorf("montar url do gateway: %w", err)
	}
	if len(filtros) > 0 {
		alvo += "?" + filtros.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, alvo, nil)
	if err != nil {
		return nil, fmt.Errorf("montar requisição do gateway: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		// Nunca embrulhe a URL na mensagem: ela sai no log e a query pode
		// carregar CNPJ de cliente.
		return nil, fmt.Errorf("chamar gateway arcom (%s): %w", dataset, err)
	}
	defer resp.Body.Close()

	// Teto de leitura: um corpo inesperadamente gigante não pode estourar a
	// memória do backend.
	corpo, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("ler resposta do gateway: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return nil, ErrNaoAutorizado
	case resp.StatusCode >= 400:
		return nil, &ErroResposta{Status: resp.StatusCode, Corpo: string(corpo)}
	}
	return corpo, nil
}
