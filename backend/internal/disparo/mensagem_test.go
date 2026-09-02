package disparo_test

import (
	"strings"
	"testing"

	"facilitaarcom/internal/disparo"
)

func TestTelefoneComDDI(t *testing.T) {
	casos := []struct {
		entrada string
		quer    string
	}{
		{"31988887777", "5531988887777"},       // celular com DDD
		{"3133334444", "553133334444"},         // fixo com DDD
		{"(31) 98888-7777", "5531988887777"},   // com máscara
		{"+55 31 98888-7777", "5531988887777"}, // já com DDI
		{"5531988887777", "5531988887777"},     // já normalizado
		{"", ""},                               // vazio segue vazio
		{"abc", ""},                            // sem dígito nenhum
	}

	for _, c := range casos {
		if got := disparo.TelefoneComDDI(c.entrada); got != c.quer {
			t.Errorf("TelefoneComDDI(%q) = %q, quer %q", c.entrada, got, c.quer)
		}
	}
}

func TestTelefoneValido(t *testing.T) {
	validos := []string{"5531988887777", "553133334444"}
	invalidos := []string{"", "55", "5531", "31988887777999999", "553"}

	for _, v := range validos {
		if !disparo.TelefoneValido(v) {
			t.Errorf("%q deveria ser válido", v)
		}
	}
	for _, v := range invalidos {
		if disparo.TelefoneValido(v) {
			t.Errorf("%q deveria ser inválido", v)
		}
	}
}

func TestMoeda(t *testing.T) {
	casos := []struct {
		valor float64
		quer  string
	}{
		{0, "R$ 0,00"},
		{1, "R$ 1,00"},
		{1250.90, "R$ 1.250,90"},
		{999.99, "R$ 999,99"},
		{1000, "R$ 1.000,00"},
		{1234567.89, "R$ 1.234.567,89"},
		{0.05, "R$ 0,05"},
		{-50.5, "-R$ 50,50"},
	}

	for _, c := range casos {
		if got := disparo.Moeda(c.valor); got != c.quer {
			t.Errorf("Moeda(%v) = %q, quer %q", c.valor, got, c.quer)
		}
	}
}

func TestMontarMensagemTrocaOsMarcadores(t *testing.T) {
	template := "Olá {nome}, o contrato {contrato} está com {dias} dias de atraso ({valor}). Negocie: {link}"

	got := disparo.MontarMensagem(template, disparo.Variaveis{
		Tratamento:   "Maria",
		NomeCompleto: "Maria Souza Oliveira",
		Contrato:     "CT-1001",
		Dias:         42,
		Valor:        1250.90,
		Link:         "https://facilita.arcom.com.br/negociar/abc",
	})

	quer := "Olá Maria, o contrato CT-1001 está com 42 dias de atraso (R$ 1.250,90). " +
		"Negocie: https://facilita.arcom.com.br/negociar/abc"
	if got != quer {
		t.Errorf("mensagem =\n%q\nquer\n%q", got, quer)
	}

	// Nenhum marcador pode sobrar sem troca: marcador cru chegando no
	// WhatsApp do cliente é erro visível.
	if strings.Contains(got, "{") || strings.Contains(got, "}") {
		t.Errorf("sobrou marcador na mensagem: %q", got)
	}
}

func TestMontarMensagemAceitaNomeCompleto(t *testing.T) {
	got := disparo.MontarMensagem("{nome} / {nome_completo}", disparo.Variaveis{
		Tratamento:   "Maria",
		NomeCompleto: "Maria Souza Oliveira",
	})
	if got != "Maria / Maria Souza Oliveira" {
		t.Errorf("got %q", got)
	}
}

// O template vem do banco, escrito pela coordenação. Ele é texto, não código:
// o que não for marcador conhecido tem que passar intacto.
func TestMontarMensagemNaoInterpretaOTemplate(t *testing.T) {
	casos := []string{
		"Use 50% de desconto {desconto_inexistente} aqui",
		"Chaves literais {} e {{nome}}",
		"Sem marcador nenhum",
	}

	for _, template := range casos {
		got := disparo.MontarMensagem(template, disparo.Variaveis{Tratamento: "Ana"})
		if strings.Contains(template, "{desconto_inexistente}") && !strings.Contains(got, "{desconto_inexistente}") {
			t.Errorf("marcador desconhecido foi consumido: %q -> %q", template, got)
		}
	}

	// {{nome}} contém {nome}, então a substituição interna acontece — o que
	// interessa é que o texto ao redor sobreviva.
	if got := disparo.MontarMensagem("Chaves {{nome}}", disparo.Variaveis{Tratamento: "Ana"}); got != "Chaves {Ana}" {
		t.Errorf("got %q", got)
	}
}
