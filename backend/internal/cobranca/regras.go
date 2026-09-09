package cobranca

import (
	"math"
	"slices"
	"strings"
	"time"
)

// FormatoData é o formato ISO usado em toda data sem hora que sai na API
// (09-contrato-api.md: nada de DD/MM/YYYY vindo do backend).
const FormatoData = "2006-01-02"

// Faixa é o agrupamento por dias de atraso que a operação usa. "fora" é
// tudo que não está entre 3 e 90 dias: ou ainda não entrou na régua, ou já
// passou dela e é caso de jurídico.
type Faixa string

const (
	Faixa3a30  Faixa = "3-30"
	Faixa31a60 Faixa = "31-60"
	Faixa61a90 Faixa = "61-90"
	FaixaFora  Faixa = "fora"
)

// DiasAtraso conta dias corridos entre o vencimento e hoje, em UTC e sempre
// no início do dia dos dois lados. Comparar timestamps direto faria o
// resultado depender da hora em que a consulta rodou.
func DiasAtraso(vencimento time.Time, agora time.Time) int {
	v := time.Date(vencimento.Year(), vencimento.Month(), vencimento.Day(), 0, 0, 0, 0, time.UTC)
	a := agora.UTC()
	hoje := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	return int(hoje.Sub(v).Hours() / 24)
}

func FaixaDe(dias int) Faixa {
	switch {
	case dias >= 3 && dias <= 30:
		return Faixa3a30
	case dias >= 31 && dias <= 60:
		return Faixa31a60
	case dias >= 61 && dias <= 90:
		return Faixa61a90
	default:
		return FaixaFora
	}
}

// ValorMinimoParcela é o piso de cada parcela de um acordo. Existe por dois
// motivos: parcelar R$ 30 em 10 vezes não faz sentido comercial, e sem piso a
// divisão de um saldo de centavos geraria parcela de R$ 0,00 (que o CHECK
// valor > 0 do banco recusa, virando erro interno na cara do cliente).
const ValorMinimoParcela = 20.0

// MaxParcelasPara diz em quantas vezes um valor pode ser dividido sem
// nenhuma parcela cair abaixo do piso. Nunca devolve menos de 1 — pagar à
// vista é sempre possível, por menor que seja o saldo.
func MaxParcelasPara(total float64) int {
	n := int(total / ValorMinimoParcela)
	if n < 1 {
		return 1
	}
	return n
}

// Oferta é uma condição que o cliente pode aceitar.
type Oferta struct {
	DescontoPct float64 `json:"descontoPct"`

	// Desconto é o valor abatido em reais. Sai de DescontoPct aplicado SOBRE OS
	// ENCARGOS, não sobre o saldo — ver CalcularOfertas.
	Desconto float64 `json:"desconto"`

	ValorTotal  float64 `json:"valorTotal"`
	Economia    float64 `json:"economia"`
	MaxParcelas int     `json:"maxParcelas"`
	Entrada     float64 `json:"entrada"`

	// ExigeAprovacao marca a condição que passa da alçada de quem está
	// olhando. A tela mostra, e mostra travada: o analista precisa saber que a
	// condição existe para pedir aprovação, e precisa não oferecer antes.
	ExigeAprovacao bool `json:"exigeAprovacao"`
}

// Ofertas são as condições calculadas para uma posição em atraso. Parcelado é
// nulo quando a política daquela faixa só permite à vista.
type Ofertas struct {
	Avista    Oferta  `json:"avista"`
	Parcelado *Oferta `json:"parcelado"`

	// Ampliada é a condição acima da alçada de quem consulta, quando a política
	// da faixa permite mais do que essa pessoa pode conceder sozinha. Nula
	// quando a alçada já cobre o teto da política.
	Ampliada *Oferta `json:"ampliada"`
}

// Posicao é o conjunto de títulos em atraso de um mesmo CNPJ.
//
// O acordo é sempre do CNPJ, nunca de um título isolado: é regra de negócio da
// ARCOM, e é por isso que o cálculo recebe a posição consolidada em vez de uma
// dívida. Um cliente com seis títulos vencidos fecha um acordo, não seis.
type Posicao struct {
	// Saldo é a soma do que o cliente deve nos títulos em atraso.
	Saldo float64 `json:"saldo"`

	// Encargos é a parte do saldo que é juros e multa, somada. É a base do
	// desconto.
	Encargos float64 `json:"encargos"`

	// Principal é o resto: mercadoria, sobre a qual não há desconto.
	Principal float64 `json:"principal"`

	// Titulos é quantos documentos o acordo cobriria.
	Titulos int `json:"titulos"`

	// DiasAtrasoMaximo é o do título mais velho, e é ele que define a faixa —
	// a política aplicada é a do pior atraso, não a média.
	DiasAtrasoMaximo int `json:"diasAtrasoMaximo"`

	Faixa Faixa `json:"faixa"`
}

// ConsolidarPosicao soma os títulos em atraso num só conjunto.
//
// Ignora o que não está em atraso e o que já foi negociado: o acordo novo é
// sobre o que ainda está aberto.
func ConsolidarPosicao(dividas []Divida, agora time.Time) Posicao {
	var p Posicao
	for _, d := range dividas {
		if d.Status != StatusDividaAberta {
			continue
		}
		dias := DiasAtraso(d.Vencimento, agora)
		if dias < 0 {
			continue // ainda não venceu
		}

		p.Saldo = Centavos(p.Saldo + d.ValorOriginal)
		p.Encargos = Centavos(p.Encargos + d.ValorEncargos)
		p.Titulos++
		if dias > p.DiasAtrasoMaximo {
			p.DiasAtrasoMaximo = dias
		}
	}
	p.Principal = Centavos(p.Saldo - p.Encargos)
	p.Faixa = FaixaDe(p.DiasAtrasoMaximo)
	return p
}

// CalcularOfertas aplica a política da faixa sobre a posição consolidada,
// limitada pela alçada de quem está consultando.
//
// ─────────────────────────────────────────────────────────────────────────
// O DESCONTO INCIDE SOMENTE SOBRE OS ENCARGOS
//
// É política da ARCOM: juros e multa são negociáveis, mercadoria não. A conta
// antiga aplicava o percentual sobre o saldo inteiro, o que num saldo de
// R$ 84.210,00 com R$ 4.370,00 de encargos transformava 20% de desconto em
// R$ 16.842,00 de abatimento em vez de R$ 874,00 — dezenove vezes mais, numa
// tela em que o cliente fecha o acordo sozinho.
//
// Posição sem encargos separados (valor_encargos zerado, que é o estado de
// toda dívida enquanto a semântica dos campos do Gateway não é confirmada)
// resulta em desconto zero. É o lado seguro: deixar de oferecer desconto é uma
// conversa com o analista, oferecer o que a empresa não autorizou é prejuízo.
//
// alcadaPct é o teto de quem consulta (usuarios.alcada_maxima). A política
// pode permitir mais do que a pessoa pode conceder — e aí a condição maior vem
// em Ampliada, marcada como exigindo aprovação, em vez de ser escondida: o
// analista precisa saber que ela existe para pedir aprovação.
// ─────────────────────────────────────────────────────────────────────────
func CalcularOfertas(p Posicao, pol *Politica, alcadaPct float64) Ofertas {
	var descAvista, descParcelado, entradaPct float64
	maxParcelas := 1
	if pol != nil {
		descAvista, descParcelado = pol.DescontoAvista, pol.DescontoParcelado
		entradaPct, maxParcelas = pol.EntradaMinimaPct, pol.MaxParcelas
	}

	ofertas := Ofertas{
		Avista: montarOferta(p, min(descAvista, alcadaPct), 1, 0),
	}

	// O teto da política é limitado pelo que o valor comporta: uma política de
	// 10x aplicada a um saldo de R$ 80 vira 4x, não 10 parcelas de R$ 8.
	parcelado := montarOferta(p, min(descParcelado, alcadaPct), maxParcelas, entradaPct)
	if teto := MaxParcelasPara(parcelado.ValorTotal); maxParcelas > teto {
		maxParcelas = teto
		parcelado = montarOferta(p, min(descParcelado, alcadaPct), maxParcelas, entradaPct)
	}
	if maxParcelas > 1 {
		ofertas.Parcelado = &parcelado
	}

	// A condição que a política permite e a alçada não: existe só quando o teto
	// da faixa passa do que a pessoa pode conceder.
	if tetoDaPolitica := max(descAvista, descParcelado); tetoDaPolitica > alcadaPct {
		ampliada := montarOferta(p, tetoDaPolitica, 1, 0)
		ampliada.ExigeAprovacao = true
		ofertas.Ampliada = &ampliada
	}

	return ofertas
}

// montarOferta é a conta única de desconto do sistema: percentual sobre
// encargos, abatido do saldo.
func montarOferta(p Posicao, descontoPct float64, parcelas int, entradaPct float64) Oferta {
	if descontoPct < 0 {
		descontoPct = 0
	}
	desconto := Centavos(p.Encargos * descontoPct / 100)
	total := Centavos(p.Saldo - desconto)

	if parcelas < 1 {
		parcelas = 1
	}
	return Oferta{
		DescontoPct: descontoPct,
		Desconto:    desconto,
		ValorTotal:  total,
		Economia:    desconto,
		MaxParcelas: parcelas,
		Entrada:     Centavos(total * entradaPct / 100),
	}
}

// Centavos arredonda para duas casas. Dinheiro em float64 acumula resto
// binário (0,1 + 0,2 não dá exatamente 0,3); sem arredondar em cada etapa, a
// soma das parcelas deixa de bater com o total do acordo.
func Centavos(v float64) float64 { return math.Round(v*100) / 100 }

// DividirParcelas reparte o total em n parcelas de centavo fechado. A
// diferença do arredondamento vai toda para a última — assim a soma bate
// exatamente com o valor_total gravado no acordo.
//
// Chame só com n já validado contra MaxParcelasPara: com n maior que o total
// em centavos não existe divisão em n parcelas positivas, e devolver nil aqui
// é a recusa de fazer conta impossível, não um caso a tratar no banco.
func DividirParcelas(total float64, n int) []float64 {
	if n < 1 || Centavos(total)*100 < float64(n) {
		return nil
	}
	base := Centavos(total / float64(n))
	valores := make([]float64, n)
	for i := 0; i < n-1; i++ {
		valores[i] = base
	}
	valores[n-1] = Centavos(total - base*float64(n-1))
	return valores
}

// VencimentosDasParcelas gera as datas: a primeira em 3 dias (prazo pra o
// cliente conseguir pagar) e as seguintes de mês em mês.
func VencimentosDasParcelas(inicio time.Time, n int) []time.Time {
	base := inicio.UTC()
	datas := make([]time.Time, n)
	for i := range n {
		datas[i] = time.Date(base.Year(), base.Month()+time.Month(i), base.Day()+3, 0, 0, 0, 0, time.UTC)
	}
	return datas
}

// sufixosSocietarios são as formas jurídicas que aparecem no fim da razão
// social e não fazem parte do nome pelo qual a empresa é conhecida.
var sufixosSocietarios = []string{
	"ltda", "ltda.", "me", "epp", "eireli", "sa", "s.a", "s.a.", "s/a",
	"mei", "cia", "cia.", "eirelli",
}

// NomeDeTratamento é como o cliente é chamado na mensagem e na tela pública.
//
// A regra depende de quem é o devedor, e a carteira da ARCOM é quase toda
// pessoa jurídica:
//
//   - CPF: primeiro nome ("Maria Souza Oliveira" -> "Maria"). É o tratamento
//     natural e evita expor o nome completo a quem tiver o link.
//   - CNPJ: a razão social sem a forma jurídica ("Mercado do João LTDA" ->
//     "Mercado do João"). Cortar na primeira palavra devolveria "Mercado",
//     que não identifica a empresa e soa como erro do sistema.
func NomeDeTratamento(nome, documento string) string {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "Cliente"
	}

	palavras := strings.Fields(nome)
	if len(palavras) == 0 {
		return "Cliente"
	}

	// Sem documento reconhecível, trata como empresa: errar para o nome
	// completo é menos ruim do que chamar uma empresa pela primeira palavra.
	if len(somenteDigitos(documento)) == 11 {
		return palavras[0]
	}

	for len(palavras) > 1 {
		ultima := strings.ToLower(strings.TrimSuffix(palavras[len(palavras)-1], ","))
		if !slices.Contains(sufixosSocietarios, ultima) {
			break
		}
		palavras = palavras[:len(palavras)-1]
	}
	return strings.Join(palavras, " ")
}

func somenteDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
