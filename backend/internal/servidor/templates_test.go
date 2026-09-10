package servidor_test

import (
	"regexp"
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

// A Meta recusa o corpo de um template que COMEÇA ou TERMINA num marcador, e
// também dois marcadores colados — é a rejeição por "dangling parameter". Os
// três templates da régua já terminaram no link uma vez; nenhum deles teria
// passado da submissão, e a régua ficaria sem canal fora da janela de
// atendimento sem ninguém entender por quê.
//
// O teste roda sobre a tabela, depois de todas as migrations, porque é o
// texto submetido à Meta que precisa obedecer à regra.
var marcadorDeTemplate = regexp.MustCompile(`\{[a-z_]+\}`)

func TestTemplatesDaReguaNaoTerminamNemComecamEmMarcador(t *testing.T) {
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
		corpo := strings.TrimSpace(c.Template)
		posicoes := marcadorDeTemplate.FindAllStringIndex(corpo, -1)
		if len(posicoes) == 0 {
			continue // outro teste já cobre a ausência de marcador
		}

		if posicoes[0][0] == 0 {
			t.Errorf("campanha %q começa num marcador — a Meta recusa o template.\n  Template: %s",
				c.Nome, corpo)
		}
		if fim := posicoes[len(posicoes)-1][1]; fim == len(corpo) {
			t.Errorf("campanha %q termina num marcador — a Meta recusa o template. Feche com texto.\n  Template: %s",
				c.Nome, corpo)
		}

		for i := 1; i < len(posicoes); i++ {
			entre := corpo[posicoes[i-1][1]:posicoes[i][0]]
			if strings.TrimSpace(entre) == "" {
				t.Errorf("campanha %q tem dois marcadores colados — a Meta recusa o template.\n  Template: %s",
					c.Nome, corpo)
			}
		}
	}
}
