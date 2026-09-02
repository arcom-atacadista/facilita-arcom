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

// CampoVencimento e CampoValor são as duas escolhas que a documentação do
// Gateway ainda não responde (ver o bloco no topo deste arquivo). Ficam
// configuráveis para que a resposta do time de TI vire troca de variável de
// ambiente, e não alteração de código com novo deploy.
type CampoVencimento string
type CampoValor string

const (
	// VencimentoPadrao é data_vencto — o nome mais direto para "data de
	// vencimento". data_vct fica como alternativa por parecer a mesma coisa
	// abreviada.
	VencimentoPadrao      CampoVencimento = "data_vencto"
	VencimentoAlternativo CampoVencimento = "data_vct"

	// ValorPadrao é vlr_liquido_deb ("valor líquido do débito"), o que mais se
	// parece com saldo devedor. deb_vlr_docto seria o valor de face do
	// documento.
	ValorPadrao    CampoValor = "vlr_liquido_deb"
	ValorDocumento CampoValor = "deb_vlr_docto"
)

// Mapeamento diz de quais campos do Gateway sair o vencimento e o valor.
type Mapeamento struct {
	Vencimento CampoVencimento
	Valor      CampoValor
}

// MapeamentoPadrao é o palpite mais defensável enquanto a semântica não é
// confirmada — e está isolado aqui para ser trocado numa linha.
func MapeamentoPadrao() Mapeamento {
	return Mapeamento{Vencimento: VencimentoPadrao, Valor: ValorPadrao}
}

// Valido recusa campo fora dos documentados, para uma variável de ambiente
// com erro de digitação não virar carteira vazia em silêncio.
func (m Mapeamento) Valido() error {
	switch m.Vencimento {
	case VencimentoPadrao, VencimentoAlternativo:
	default:
		return fmt.Errorf("campo de vencimento %q não existe no dataset debitos", m.Vencimento)
	}
	switch m.Valor {
	case ValorPadrao, ValorDocumento:
	default:
		return fmt.Errorf("campo de valor %q não existe no dataset debitos", m.Valor)
	}
	return nil
}

// Mapear traduz um débito do Gateway para a linha de carteira, usando o
// mapeamento configurado.
//
// É AQUI que moram as duas escolhas pendentes de confirmação (ver o bloco no
// topo do arquivo). Estão concentradas neste ponto justamente para que a
// resposta do time de TI seja uma troca de configuração.
func Mapear(d Debito, m Mapeamento) (LinhaCarteira, error) {
	vencimento := d.DataVencto
	if m.Vencimento == VencimentoAlternativo {
		vencimento = d.DataVct
	}
	// Cair para o outro campo quando o escolhido vem vazio é melhor que
	// descartar a linha — mas só quando o outro tem conteúdo.
	if vencimento == "" {
		if m.Vencimento == VencimentoAlternativo {
			vencimento = d.DataVencto
		} else {
			vencimento = d.DataVct
		}
	}

	valor := d.ValorLiquido
	if m.Valor == ValorDocumento {
		valor = d.DebValorDocumento
	}
	if valor <= 0 {
		if m.Valor == ValorDocumento {
			valor = d.ValorLiquido
		} else {
			valor = d.DebValorDocumento
		}
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
	case d.SeqDebito == "":
		return LinhaCarteira{}, errors.New("débito sem identificador (seq_debito)")
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

// --- telefone do devedor ---
//
// O dataset `debitos` não traz telefone: nenhum dos campos documentados tem
// contato do cliente. O único lugar do Gateway com o número do devedor é o
// histórico da própria Nines (`disparo-mensagem-nines`), que guarda para onde
// cada mensagem foi.
//
// Na prática isso é o que permite sair da Nines sem perder a base de
// telefones: ela é reconstruída a partir do que já foi disparado.

// DisparoNines é o subconjunto do histórico que interessa: para quem foi e
// de quem era a inscrição.
type DisparoNines struct {
	DataDisparo string `json:"dataDisparo"`
	// O nome do campo vem com a letra faltando na origem; mantido como está
	// para casar com o que a API devolve.
	NroIncricao string `json:"nroIncricao"`
	Payload     struct {
		Telefone     string `json:"telefone"`
		PhoneNumber  string `json:"phone_number"`
		Contato      string `json:"contato"`
		Document     string `json:"document"`
		NroInscricao string `json:"nroInscricao"`
		Nome         string `json:"nome"`
	} `json:"payload"`
}

// Documento devolve a inscrição do devedor, tentando os lugares onde ela
// aparece no histórico.
func (d DisparoNines) Documento() string {
	return somenteDigitos(primeiroNaoVazio(d.NroIncricao, d.Payload.NroInscricao, d.Payload.Document))
}

// Telefone devolve o número para onde a mensagem foi.
func (d DisparoNines) Telefone() string {
	return somenteDigitos(primeiroNaoVazio(d.Payload.Telefone, d.Payload.PhoneNumber, d.Payload.Contato))
}

// BuscarTelefonesDaNines lê o histórico de disparos e devolve o mapa
// documento -> telefone. Como a lista vem ordenada do mais recente para o
// mais antigo na maioria das consultas, a primeira ocorrência de cada
// documento é mantida — mas não dependemos disso: só sobrescrevemos quando o
// registro é mais novo.
func (c *Cliente) BuscarTelefonesDaNines(ctx context.Context, limite int) (map[string]string, error) {
	filtros := url.Values{}
	if limite > 0 {
		filtros.Set("limite", strconv.Itoa(limite))
	}
	filtros.Set("ordenar_por", "dataDisparo")

	corpo, err := c.Consultar(ctx, DatasetDisparosNines, filtros)
	if err != nil {
		return nil, err
	}

	var disparos []DisparoNines
	if err := decodificarLista(corpo, &disparos); err != nil {
		return nil, fmt.Errorf("ler histórico de disparos: %w", err)
	}

	telefones := make(map[string]string, len(disparos))
	maisRecente := make(map[string]string, len(disparos))

	for _, d := range disparos {
		doc, tel := d.Documento(), d.Telefone()
		if doc == "" || tel == "" {
			continue
		}
		if anterior, existe := maisRecente[doc]; existe && anterior >= d.DataDisparo {
			continue
		}
		telefones[doc] = tel
		maisRecente[doc] = d.DataDisparo
	}
	return telefones, nil
}
