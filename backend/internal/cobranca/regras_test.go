package cobranca_test

import (
	"testing"
	"time"

	"facilitaarcom/internal/cobranca"
)

func data(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestDiasAtraso(t *testing.T) {
	casos := []struct {
		nome       string
		vencimento string
		agora      time.Time
		quer       int
	}{
		{"vence hoje", "2026-09-02", data("2026-09-02"), 0},
		{"um dia", "2026-09-01", data("2026-09-02"), 1},
		{"trinta dias", "2026-08-03", data("2026-09-02"), 30},
		{"ainda vai vencer", "2026-09-10", data("2026-09-02"), -8},
		// Vencimento à noite e consulta de manhã não podem dar um dia a
		// menos: a conta é entre dias, não entre instantes.
		{"hora do dia não interfere", "2026-09-01", time.Date(2026, 9, 2, 23, 59, 0, 0, time.UTC), 1},
		{"virada de mês", "2026-01-31", data("2026-03-01"), 29},
		{"ano bissexto", "2024-02-28", data("2024-03-01"), 2},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := cobranca.DiasAtraso(data(c.vencimento), c.agora); got != c.quer {
				t.Errorf("DiasAtraso = %d, quer %d", got, c.quer)
			}
		})
	}
}

func TestFaixaDe(t *testing.T) {
	casos := []struct {
		dias int
		quer cobranca.Faixa
	}{
		{0, cobranca.FaixaFora},
		{2, cobranca.FaixaFora},
		{3, cobranca.Faixa3a30},
		{30, cobranca.Faixa3a30},
		{31, cobranca.Faixa31a60},
		{60, cobranca.Faixa31a60},
		{61, cobranca.Faixa61a90},
		{90, cobranca.Faixa61a90},
		{91, cobranca.FaixaFora},
		{-5, cobranca.FaixaFora},
	}

	for _, c := range casos {
		if got := cobranca.FaixaDe(c.dias); got != c.quer {
			t.Errorf("FaixaDe(%d) = %q, quer %q", c.dias, got, c.quer)
		}
	}
}

func TestDividirParcelasSempreSomaOTotal(t *testing.T) {
	// Valores escolhidos por não dividirem redondo: é aí que o centavo some.
	totais := []float64{100, 100.01, 1250.90, 999.99, 33.33, 7777.77, 480.05}
	quantidades := []int{1, 2, 3, 4, 6, 7, 10, 12, 24}

	for _, total := range totais {
		for _, n := range quantidades {
			if n > cobranca.MaxParcelasPara(total) {
				continue // combinação que o serviço recusa antes de chegar aqui
			}
			valores := cobranca.DividirParcelas(total, n)
			if len(valores) != n {
				t.Fatalf("DividirParcelas(%v, %d) devolveu %d parcelas", total, n, len(valores))
			}

			var soma float64
			for _, v := range valores {
				soma += v
			}
			if cobranca.Centavos(soma) != cobranca.Centavos(total) {
				t.Errorf("soma das parcelas de %v em %d vezes = %v, quer %v (valores: %v)",
					total, n, cobranca.Centavos(soma), total, valores)
			}

			for i, v := range valores {
				if v <= 0 {
					t.Errorf("parcela %d de %v em %d vezes ficou %v — o banco recusa (CHECK valor > 0)", i+1, total, n, v)
				}
			}
		}
	}
}

func TestCalcularOfertasSemPoliticaNaoDaDesconto(t *testing.T) {
	// Dívida fora da régua (menos de 3 ou mais de 90 dias) não tem política.
	// O default nunca pode ser "desconto livre".
	o := cobranca.CalcularOfertas(1000, nil)

	if o.Avista.DescontoPct != 0 {
		t.Errorf("desconto à vista sem política = %v, quer 0", o.Avista.DescontoPct)
	}
	if o.Avista.ValorTotal != 1000 {
		t.Errorf("valor à vista sem política = %v, quer 1000", o.Avista.ValorTotal)
	}
	if o.Parcelado != nil {
		t.Error("sem política não pode haver parcelamento")
	}
}

func TestCalcularOfertasAplicaAPolitica(t *testing.T) {
	p := &cobranca.Politica{
		DescontoAvista:    20,
		DescontoParcelado: 10,
		MaxParcelas:       10,
		EntradaMinimaPct:  15,
	}
	o := cobranca.CalcularOfertas(1250.90, p)

	if quer := 1000.72; o.Avista.ValorTotal != quer {
		t.Errorf("valor à vista = %v, quer %v", o.Avista.ValorTotal, quer)
	}
	if quer := 250.18; o.Avista.Economia != quer {
		t.Errorf("economia à vista = %v, quer %v", o.Avista.Economia, quer)
	}

	if o.Parcelado == nil {
		t.Fatal("política com max_parcelas 10 deveria oferecer parcelamento")
	}
	if quer := 1125.81; o.Parcelado.ValorTotal != quer {
		t.Errorf("valor parcelado = %v, quer %v", o.Parcelado.ValorTotal, quer)
	}
	if quer := 168.87; o.Parcelado.Entrada != quer {
		t.Errorf("entrada = %v, quer %v", o.Parcelado.Entrada, quer)
	}

	// O parcelado nunca pode sair mais barato que o à vista — seria vantagem
	// financeira em atrasar, e o desconto à vista existe justamente por isso.
	if o.Parcelado.ValorTotal < o.Avista.ValorTotal {
		t.Errorf("parcelado (%v) saiu menor que à vista (%v)", o.Parcelado.ValorTotal, o.Avista.ValorTotal)
	}
}

func TestCalcularOfertasPoliticaSemParcelamento(t *testing.T) {
	p := &cobranca.Politica{DescontoAvista: 5, MaxParcelas: 1}
	if o := cobranca.CalcularOfertas(500, p); o.Parcelado != nil {
		t.Error("max_parcelas 1 não pode gerar oferta parcelada")
	}
}

func TestVencimentosDasParcelas(t *testing.T) {
	datas := cobranca.VencimentosDasParcelas(data("2026-09-02"), 3)

	if len(datas) != 3 {
		t.Fatalf("gerou %d datas, quer 3", len(datas))
	}
	// Primeira em 3 dias, depois de mês em mês.
	esperadas := []string{"2026-09-05", "2026-10-05", "2026-11-05"}
	for i, quer := range esperadas {
		if got := datas[i].Format(cobranca.FormatoData); got != quer {
			t.Errorf("parcela %d vence em %s, quer %s", i+1, got, quer)
		}
	}

	// Cada parcela tem que vencer depois da anterior — a aritmética de mês do
	// Go normaliza (31 de janeiro + 1 mês = 3 de março), e o que não pode é
	// duas parcelas na mesma data ou fora de ordem.
	longa := cobranca.VencimentosDasParcelas(data("2026-01-29"), 12)
	for i := 1; i < len(longa); i++ {
		if !longa[i].After(longa[i-1]) {
			t.Errorf("parcela %d (%s) não vence depois da %d (%s)",
				i+1, longa[i].Format(cobranca.FormatoData), i, longa[i-1].Format(cobranca.FormatoData))
		}
	}
}

func TestMaxParcelasRespeitaOPisoDaParcela(t *testing.T) {
	casos := []struct {
		total float64
		quer  int
	}{
		{0.03, 1},  // saldo de centavos: só à vista
		{19.99, 1}, // abaixo do piso de uma parcela
		{20, 1},    // exatamente o piso
		{40, 2},
		{100, 5},
		{480, 24},
		{5000, 250}, // sem teto próprio — quem limita é a política
	}

	for _, c := range casos {
		if got := cobranca.MaxParcelasPara(c.total); got != c.quer {
			t.Errorf("MaxParcelasPara(%v) = %d, quer %d", c.total, got, c.quer)
		}
	}
}

func TestDividirParcelasRecusaDivisaoImpossivel(t *testing.T) {
	// 3 centavos não viram 4 parcelas positivas. Devolver nil aqui é o que
	// impede o acordo de chegar ao banco e bater no CHECK valor > 0.
	if v := cobranca.DividirParcelas(0.03, 4); v != nil {
		t.Errorf("DividirParcelas(0.03, 4) = %v, quer nil", v)
	}
	if v := cobranca.DividirParcelas(100, 0); v != nil {
		t.Errorf("DividirParcelas(100, 0) = %v, quer nil", v)
	}
}

func TestOfertaNuncaOferecePrestacaoAbaixoDoPiso(t *testing.T) {
	// Política generosa (10x) numa dívida pequena: o teto tem que cair para o
	// que o valor comporta, senão a tela oferece um parcelamento que o
	// servidor vai recusar no aceite.
	p := &cobranca.Politica{DescontoParcelado: 10, MaxParcelas: 10}
	o := cobranca.CalcularOfertas(90, p)

	if o.Parcelado == nil {
		t.Fatal("R$ 90 com desconto de 10% comporta parcelamento")
	}
	if o.Parcelado.MaxParcelas != 4 {
		t.Errorf("maxParcelas = %d, quer 4 (R$ 81,00 / R$ 20,00)", o.Parcelado.MaxParcelas)
	}

	valores := cobranca.DividirParcelas(o.Parcelado.ValorTotal, o.Parcelado.MaxParcelas)
	for i, v := range valores {
		if v < cobranca.ValorMinimoParcela {
			t.Errorf("parcela %d = %v, abaixo do piso de %v", i+1, v, cobranca.ValorMinimoParcela)
		}
	}
}
