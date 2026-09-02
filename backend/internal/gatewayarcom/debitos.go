package gatewayarcom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

// ─────────────────────────────────────────────────────────────────────────
// PENDENTE DE CONFIRMAÇÃO COM O TIME DE TI DA ARCOM
//
// Duas coisas que este arquivo precisa e que a documentação do Gateway não
// responde. Nada aqui foi adivinhado — está isolado e sinalizado, como manda
// 05-regras-para-a-ia.md ("não invente o mecanismo de acesso").
//
//  1. O envelope da resposta. O openapi.yaml descreve o 200 apenas como
//     "pagina de resultados", sem schema — não diz se a lista vem em "itens",
//     "data", "hits" ou como array puro. decodificarLista abaixo aceita as
//     formas plausíveis e falha com mensagem explícita se vier outra, em vez
//     de devolver lista vazia como se não houvesse débito.
//
//  2. Quais campos são de fato o vencimento e o valor da dívida. O dataset
//     tem data_vencto, data_vct e data_emissao; e deb_vlr_docto e
//     vlr_liquido_deb. O próprio api.md marca "Semântica a revisar: definição
//     de datas (pedido vs faturado)". A escolha está em Mapear(), num lugar
//     só, para trocar numa linha quando a resposta vier.
//
// Até a confirmação, a sincronização não deve rodar contra produção.
// ─────────────────────────────────────────────────────────────────────────

// Debito é o subconjunto do dataset "debitos" que a cobrança usa. Só os
// campos que a tela precisa — 12-gateway-arcom.md proíbe repassar o payload
// do Gateway inteiro.
type Debito struct {
	Cliente             string  `json:"cliente"`
	RazaoSocial         string  `json:"razao_social"`
	CNPJ                string  `json:"cnpj"`
	Documento           string  `json:"documento"`
	SeqDebito           string  `json:"seq_debito"`
	DataVencto          string  `json:"data_vencto"`
	DataVct             string  `json:"data_vct"`
	DebValorDocumento   float64 `json:"deb_vlr_docto"`
	ValorLiquido        float64 `json:"vlr_liquido_deb"`
	Filial              string  `json:"filial"`
	ResponsavelCobranca string  `json:"responsavel_cobranca"`
	NomeRespCobranca    string  `json:"nome_resp_cob"`
}

// LinhaCarteira é o débito já traduzido para o vocabulário da cobrança.
type LinhaCarteira struct {
	NomeCliente         string
	Documento           string
	Contrato            string
	Valor               float64
	Vencimento          string
	Filial              string
	ResponsavelCobranca string
}

// ErrEnvelopeDesconhecido é o "não sei ler isto" explícito. Sem ele, um
// envelope diferente do previsto viraria zero débitos — e uma carteira vazia
// parece um dia sem inadimplência, não um erro de integração.
var ErrEnvelopeDesconhecido = errors.New(
	"formato de resposta do Gateway não reconhecido — confirmar o envelope com o time de TI da ARCOM")

// BuscarDebitos lê a carteira em atraso. responsavel vazio traz todos.
func (c *Cliente) BuscarDebitos(ctx context.Context, responsavel string, limite int) ([]Debito, error) {
	filtros := url.Values{}
	if responsavel != "" {
		// .keyword é a forma de filtro exato documentada em api.md.
		filtros.Set("responsavel_cobranca.keyword", responsavel)
	}
	if limite > 0 {
		filtros.Set("limite", strconv.Itoa(limite))
	}

	corpo, err := c.Consultar(ctx, DatasetDebitos, filtros)
	if err != nil {
		return nil, err
	}

	var debitos []Debito
	if err := decodificarLista(corpo, &debitos); err != nil {
		return nil, fmt.Errorf("ler débitos do gateway: %w", err)
	}
	return debitos, nil
}

// decodificarLista tolera as formas de envelope plausíveis enquanto a real
// não é confirmada. A ordem é a de probabilidade; array puro por último
// porque é o menos comum em API paginada.
func decodificarLista(corpo []byte, destino any) error {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(corpo, &envelope); err == nil {
		for _, chave := range []string{"itens", "items", "data", "resultados", "results", "hits", "registros"} {
			bruto, ok := envelope[chave]
			if !ok {
				continue
			}
			if err := json.Unmarshal(bruto, destino); err != nil {
				return fmt.Errorf("campo %q não é uma lista de registros: %w", chave, err)
			}
			return nil
		}
		return ErrEnvelopeDesconhecido
	}

	// Array puro na raiz.
	if err := json.Unmarshal(corpo, destino); err != nil {
		return ErrEnvelopeDesconhecido
	}
	return nil
}

// Mapear traduz um débito do Gateway para a linha de carteira.
//
// É AQUI que moram as duas escolhas pendentes de confirmação (ver o bloco no
// topo do arquivo). Estão concentradas neste ponto justamente para que a
// resposta do time de TI vire uma troca de uma linha, e não uma caçada pelo
// código.
func Mapear(d Debito) (LinhaCarteira, error) {
	// ESCOLHA PENDENTE 1 — vencimento. data_vencto é o nome mais direto para
	// "data de vencimento"; data_vct fica como reserva por parecer a mesma
	// coisa abreviada. Confirmar qual é a data de vencimento do título.
	vencimento := d.DataVencto
	if vencimento == "" {
		vencimento = d.DataVct
	}

	// ESCOLHA PENDENTE 2 — valor. vlr_liquido_deb ("valor líquido do débito")
	// é o que mais se parece com o saldo devedor; deb_vlr_docto seria o valor
	// de face do documento. Cobrar o valor errado é erro caro nos dois
	// sentidos, então isto precisa de confirmação antes de ir a produção.
	valor := d.ValorLiquido
	if valor <= 0 {
		valor = d.DebValorDocumento
	}

	documento := primeiroNaoVazio(d.CNPJ, d.Documento)
	nome := primeiroNaoVazio(d.RazaoSocial, d.Cliente)

	// Sem identificação ou sem vencimento não dá para montar régua nenhuma —
	// a linha é recusada em vez de entrar torta na carteira.
	switch {
	case documento == "":
		return LinhaCarteira{}, fmt.Errorf("débito %q sem CNPJ nem documento", d.SeqDebito)
	case vencimento == "":
		return LinhaCarteira{}, fmt.Errorf("débito %q sem data de vencimento", d.SeqDebito)
	case valor <= 0:
		return LinhaCarteira{}, fmt.Errorf("débito %q sem valor positivo", d.SeqDebito)
	}

	return LinhaCarteira{
		NomeCliente:         nome,
		Documento:           somenteDigitos(documento),
		Contrato:            d.SeqDebito,
		Valor:               valor,
		Vencimento:          vencimento[:min(10, len(vencimento))],
		Filial:              d.Filial,
		ResponsavelCobranca: d.ResponsavelCobranca,
	}, nil
}

func primeiroNaoVazio(valores ...string) string {
	for _, v := range valores {
		if v != "" {
			return v
		}
	}
	return ""
}

func somenteDigitos(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return string(out)
}
