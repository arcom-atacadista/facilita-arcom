package cobranca

import (
	"math"
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
	ValorTotal  float64 `json:"valorTotal"`
	Economia    float64 `json:"economia"`
	MaxParcelas int     `json:"maxParcelas"`
	Entrada     float64 `json:"entrada"`
}

// Ofertas são as condições calculadas para uma dívida. Parcelado é nulo
// quando a política daquela faixa só permite à vista.
type Ofertas struct {
	Avista    Oferta  `json:"avista"`
	Parcelado *Oferta `json:"parcelado"`
}

// CalcularOfertas aplica a política da faixa sobre o valor original.
// Política nula (dívida fora da régua) significa nenhum desconto e pagamento
// à vista — nunca "desconto livre".
func CalcularOfertas(valorOriginal float64, p *Politica) Ofertas {
	var descAvista, descParcelado, entradaPct float64
	maxParcelas := 1
	if p != nil {
		descAvista, descParcelado = p.DescontoAvista, p.DescontoParcelado
		entradaPct, maxParcelas = p.EntradaMinimaPct, p.MaxParcelas
	}

	totalAvista := Centavos(valorOriginal * (1 - descAvista/100))
	ofertas := Ofertas{
		Avista: Oferta{
			DescontoPct: descAvista,
			ValorTotal:  totalAvista,
			Economia:    Centavos(valorOriginal - totalAvista),
			MaxParcelas: 1,
		},
	}

	// O teto da política é limitado pelo que o valor comporta: uma política
	// de 10x aplicada a uma dívida de R$ 80 vira 4x, não 10 parcelas de R$ 8.
	if teto := MaxParcelasPara(Centavos(valorOriginal * (1 - descParcelado/100))); maxParcelas > teto {
		maxParcelas = teto
	}

	if maxParcelas > 1 {
		totalParcelado := Centavos(valorOriginal * (1 - descParcelado/100))
		ofertas.Parcelado = &Oferta{
			DescontoPct: descParcelado,
			ValorTotal:  totalParcelado,
			Economia:    Centavos(valorOriginal - totalParcelado),
			MaxParcelas: maxParcelas,
			Entrada:     Centavos(totalParcelado * entradaPct / 100),
		}
	}
	return ofertas
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
