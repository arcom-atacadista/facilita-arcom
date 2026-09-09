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

// posicaoDoExemplo são os números que o desenho da mesa usa, e servem de
// documentação executável da política: saldo de R$ 84.210,00 do qual
// R$ 4.370,00 é juros e multa.
func posicaoDoExemplo() cobranca.Posicao {
	return cobranca.Posicao{
		Saldo:            84210.00,
		Encargos:         4370.00,
		Principal:        79840.00,
		Titulos:          6,
		DiasAtrasoMaximo: 42,
	}
}

func TestDescontoIncideSomenteSobreEncargos(t *testing.T) {
	// A REGRA MAIS CARA DO SISTEMA. A política da ARCOM concede desconto só
	// sobre juros e multa, nunca sobre mercadoria. A conta antiga aplicava o
	// percentual no saldo inteiro e transformava 20% em R$ 16.842,00 de
	// abatimento em vez de R$ 874,00 — dezenove vezes mais, numa tela em que o
	// cliente fecha o acordo sozinho.
	p := &cobranca.Politica{DescontoAvista: 20, MaxParcelas: 1}
	o := cobranca.CalcularOfertas(posicaoDoExemplo(), p, 100)

	if quer := 874.00; o.Avista.Desconto != quer {
		t.Errorf("desconto = %v, quer %v (20%% de R$ 4.370,00 de encargos)", o.Avista.Desconto, quer)
	}
	if quer := 83336.00; o.Avista.ValorTotal != quer {
		t.Errorf("valor à vista = %v, quer %v", o.Avista.ValorTotal, quer)
	}
	// A trava contra a volta da conta antiga: 20% do saldo seriam R$ 16.842,00.
	if o.Avista.Desconto > posicaoDoExemplo().Encargos {
		t.Errorf("desconto de %v passou dos encargos (%v) — o desconto voltou a incidir sobre o principal",
			o.Avista.Desconto, posicaoDoExemplo().Encargos)
	}
}

func TestSemEncargosSeparadosNaoDaDesconto(t *testing.T) {
	// Estado de toda dívida enquanto a semântica dos campos do Gateway não é
	// confirmada. Zero desconto é o lado seguro do erro: deixar de oferecer é
	// uma conversa com o analista, oferecer o que a empresa não autorizou é
	// prejuízo que já saiu.
	posicao := cobranca.Posicao{Saldo: 10000, Encargos: 0, Principal: 10000, Titulos: 1, DiasAtrasoMaximo: 40}
	p := &cobranca.Politica{DescontoAvista: 50, DescontoParcelado: 50, MaxParcelas: 6}

	o := cobranca.CalcularOfertas(posicao, p, 100)
	if o.Avista.Desconto != 0 {
		t.Errorf("desconto sem encargos separados = %v, quer 0", o.Avista.Desconto)
	}
	if o.Avista.ValorTotal != 10000 {
		t.Errorf("valor à vista = %v, quer o saldo cheio 10000", o.Avista.ValorTotal)
	}
}

func TestCalcularOfertasSemPoliticaNaoDaDesconto(t *testing.T) {
	// Dívida fora da régua (menos de 3 ou mais de 90 dias) não tem política.
	// O default nunca pode ser "desconto livre".
	o := cobranca.CalcularOfertas(posicaoDoExemplo(), nil, 100)

	if o.Avista.DescontoPct != 0 {
		t.Errorf("desconto à vista sem política = %v, quer 0", o.Avista.DescontoPct)
	}
	if quer := 84210.00; o.Avista.ValorTotal != quer {
		t.Errorf("valor à vista sem política = %v, quer o saldo cheio %v", o.Avista.ValorTotal, quer)
	}
	if o.Parcelado != nil {
		t.Error("sem política não pode haver parcelamento")
	}
}

func TestAlcadaLimitaODescontoEExpoeACondicaoTravada(t *testing.T) {
	// A política permite 50%, o usuário só pode 20%. A condição maior não é
	// escondida: o analista precisa saber que ela existe para pedir aprovação,
	// e precisa não oferecer antes de conseguir.
	p := &cobranca.Politica{DescontoAvista: 50, DescontoParcelado: 50, MaxParcelas: 3}
	o := cobranca.CalcularOfertas(posicaoDoExemplo(), p, 20)

	if quer := 20.0; o.Avista.DescontoPct != quer {
		t.Errorf("desconto oferecido = %v%%, quer o teto da alçada (%v%%)", o.Avista.DescontoPct, quer)
	}
	if o.Avista.ExigeAprovacao {
		t.Error("a condição dentro da alçada não deveria exigir aprovação")
	}

	if o.Ampliada == nil {
		t.Fatal("política de 50%% com alçada de 20%% deveria expor a condição ampliada")
	}
	if quer := 50.0; o.Ampliada.DescontoPct != quer {
		t.Errorf("desconto ampliado = %v%%, quer %v%%", o.Ampliada.DescontoPct, quer)
	}
	if !o.Ampliada.ExigeAprovacao {
		t.Error("a condição acima da alçada tem que vir marcada como exigindo aprovação")
	}
	if quer := 2185.00; o.Ampliada.Desconto != quer {
		t.Errorf("desconto ampliado = %v, quer %v (50%% dos encargos)", o.Ampliada.Desconto, quer)
	}
}

func TestAlcadaQueCobreAPoliticaNaoExpoeCondicaoTravada(t *testing.T) {
	p := &cobranca.Politica{DescontoAvista: 20, DescontoParcelado: 10, MaxParcelas: 3}
	if o := cobranca.CalcularOfertas(posicaoDoExemplo(), p, 60); o.Ampliada != nil {
		t.Error("alçada acima do teto da política não deveria produzir condição travada")
	}
}

func TestParceladoNuncaSaiMaisBaratoQueAvista(t *testing.T) {
	// Seria vantagem financeira em atrasar, e o desconto à vista existe
	// justamente para o contrário.
	p := &cobranca.Politica{
		DescontoAvista:    20,
		DescontoParcelado: 10,
		MaxParcelas:       10,
		EntradaMinimaPct:  15,
	}
	o := cobranca.CalcularOfertas(posicaoDoExemplo(), p, 100)

	if o.Parcelado == nil {
		t.Fatal("política com max_parcelas 10 deveria oferecer parcelamento")
	}
	if o.Parcelado.ValorTotal < o.Avista.ValorTotal {
		t.Errorf("parcelado (%v) saiu menor que à vista (%v)", o.Parcelado.ValorTotal, o.Avista.ValorTotal)
	}
	if quer := 437.00; o.Parcelado.Desconto != quer {
		t.Errorf("desconto parcelado = %v, quer %v (10%% dos encargos)", o.Parcelado.Desconto, quer)
	}
}

func TestCalcularOfertasPoliticaSemParcelamento(t *testing.T) {
	p := &cobranca.Politica{DescontoAvista: 5, MaxParcelas: 1}
	if o := cobranca.CalcularOfertas(posicaoDoExemplo(), p, 100); o.Parcelado != nil {
		t.Error("política com max_parcelas 1 não pode oferecer parcelamento")
	}
}

func TestConsolidarPosicaoSoSomaTituloAbertoEVencido(t *testing.T) {
	agora := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	dividas := []cobranca.Divida{
		// Em atraso e aberto: entra.
		{Status: "aberto", ValorOriginal: 1000, ValorEncargos: 100, Vencimento: agora.AddDate(0, 0, -40)},
		{Status: "aberto", ValorOriginal: 500, ValorEncargos: 50, Vencimento: agora.AddDate(0, 0, -70)},
		// Já negociado: pertence a outro acordo, e recalcular daria desconto
		// duas vezes sobre o mesmo encargo.
		{Status: "negociado", ValorOriginal: 9999, ValorEncargos: 999, Vencimento: agora.AddDate(0, 0, -50)},
		// Ainda não venceu: não é carteira em atraso.
		{Status: "aberto", ValorOriginal: 7777, ValorEncargos: 777, Vencimento: agora.AddDate(0, 0, 10)},
	}

	p := cobranca.ConsolidarPosicao(dividas, agora)

	if quer := 1500.00; p.Saldo != quer {
		t.Errorf("saldo = %v, quer %v", p.Saldo, quer)
	}
	if quer := 150.00; p.Encargos != quer {
		t.Errorf("encargos = %v, quer %v", p.Encargos, quer)
	}
	if quer := 1350.00; p.Principal != quer {
		t.Errorf("principal = %v, quer %v", p.Principal, quer)
	}
	if p.Titulos != 2 {
		t.Errorf("títulos = %d, quer 2", p.Titulos)
	}
	// A faixa é a do título mais velho, não a média: a política aplicada é a do
	// pior atraso.
	if p.DiasAtrasoMaximo != 70 {
		t.Errorf("dias de atraso = %d, quer 70 (o título mais velho)", p.DiasAtrasoMaximo)
	}
	if quer := cobranca.Faixa61a90; p.Faixa != quer {
		t.Errorf("faixa = %v, quer %v", p.Faixa, quer)
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
	// Saldo de R$ 90 com R$ 90 de encargos: o caso extremo em que o desconto
	// morde o máximo possível, e ainda assim o teto de parcelas tem que cair.
	posicao := cobranca.Posicao{Saldo: 90, Encargos: 90, Titulos: 1, DiasAtrasoMaximo: 40}
	o := cobranca.CalcularOfertas(posicao, p, 100)

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

// A carteira da ARCOM é quase toda pessoa jurídica. Chamar "Mercado do João
// LTDA" de "Mercado" — o que a regra de primeiro nome fazia — não identifica
// a empresa e parece erro do sistema na mensagem que o cliente recebe.
func TestNomeDeTratamento(t *testing.T) {
	casos := []struct {
		nome      string
		documento string
		quer      string
	}{
		// Pessoa física: primeiro nome.
		{"Maria Souza Oliveira", "12345678901", "Maria"},
		{"João", "12345678901", "João"},

		// Empresa: razão social sem a forma jurídica.
		{"Mercado do João LTDA", "12345678000199", "Mercado do João"},
		{"Padaria Estrela ME", "12.345.678/0001-99", "Padaria Estrela"},
		{"Distribuidora Alfa EIRELI", "12345678000199", "Distribuidora Alfa"},
		{"Comercial Beta S/A", "12345678000199", "Comercial Beta"},
		{"Atacado Gama LTDA ME", "12345678000199", "Atacado Gama"},
		{"Supermercados Delta", "12345678000199", "Supermercados Delta"},

		// Sem documento reconhecível, trata como empresa: errar para o nome
		// completo é menos ruim do que cortar na primeira palavra.
		{"Mercado do João LTDA", "", "Mercado do João"},

		// Nome vazio nunca vira mensagem começando com "Olá ,".
		{"", "12345678901", "Cliente"},
		{"   ", "12345678000199", "Cliente"},

		// Empresa cujo nome inteiro é a forma jurídica: não sobra nada para
		// cortar, e devolver vazio seria pior que devolver o original.
		{"LTDA", "12345678000199", "LTDA"},
	}

	for _, c := range casos {
		if got := cobranca.NomeDeTratamento(c.nome, c.documento); got != c.quer {
			t.Errorf("NomeDeTratamento(%q, %q) = %q, quer %q", c.nome, c.documento, got, c.quer)
		}
	}
}
