package servidor_test

import (
	"strings"
	"testing"
)

// A Meta cobra por mensagem entregue e decide a categoria lendo o TEXTO do
// template. No Brasil marketing custa cerca de 9x o preço de utility
// (~R$ 0,31-0,38 contra ~R$ 0,04-0,05), e desde abril de 2025 a
// reclassificação é automática: pedir UTILITY não adianta se o texto parecer
// promoção — ele é aprovado como MARKETING e a conta multiplica sem aviso.
//
// A regra da Meta: template utility não pode conter desconto, oferta nem
// linguagem promocional. Um lembrete de pagamento que oferece desconto é
// marketing.
//
// Este teste lê a tabela depois de TODAS as migrations, e não o texto dos
// arquivos: o que importa é o template que a régua vai usar de verdade.
var palavrasQueViramMarketing = []string{
	"desconto", "oferta", "promoção", "promocao", "promocional",
	"condições especiais", "condicoes especiais", "condição especial",
	"imperdível", "imperdivel", "aproveite", "não perca", "nao perca",
	"exclusivo", "exclusiva", "última chance", "ultima chance",
	"última oportunidade", "ultima oportunidade", "vantagem",
	"bônus", "bonus", "grátis", "gratis",
}

func TestTemplatesDaReguaSeClassificamComoUtility(t *testing.T) {
	gdb := bancoDeTeste(t)

	var campanhas []struct {
		Nome     string
		Template string
	}
	if err := gdb.Raw(`SELECT nome, template FROM campanhas ORDER BY faixa_min`).Scan(&campanhas).Error; err != nil {
		t.Fatalf("ler campanhas: %v", err)
	}
	if len(campanhas) == 0 {
		t.Fatal("nenhuma campanha semeada — o teste passaria sem verificar nada")
	}

	for _, c := range campanhas {
		minusculo := strings.ToLower(c.Template)
		for _, proibida := range palavrasQueViramMarketing {
			if strings.Contains(minusculo, proibida) {
				t.Errorf("campanha %q contém %q — a Meta classifica como MARKETING, ~9x o custo de utility.\n  Template: %s\n  O desconto deve ficar na página de negociação, que o link abre, não na mensagem.",
					c.Nome, proibida, c.Template)
			}
		}
	}
}

// Sem {link} a mensagem não leva a lugar nenhum; sem {contrato} e {valor} ela
// deixa de ser específica da conta, que é exatamente o que a Meta exige para
// aceitar um template como utility.
func TestTemplatesDaReguaSaoEspecificosDaConta(t *testing.T) {
	gdb := bancoDeTeste(t)

	var campanhas []struct {
		Nome     string
		Template string
	}
	if err := gdb.Raw(`SELECT nome, template FROM campanhas`).Scan(&campanhas).Error; err != nil {
		t.Fatalf("ler campanhas: %v", err)
	}

	for _, c := range campanhas {
		for _, marcador := range []string{"{nome}", "{contrato}", "{valor}", "{dias}", "{link}"} {
			if !strings.Contains(c.Template, marcador) {
				t.Errorf("campanha %q não usa %s — %s", c.Nome, marcador, c.Template)
			}
		}
	}
}
