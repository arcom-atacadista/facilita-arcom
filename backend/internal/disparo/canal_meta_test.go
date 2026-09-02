package disparo_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"facilitaarcom/internal/disparo"
)

// Nenhum teste aqui toca a Meta de verdade: o canal aponta para um servidor
// falso, que é o que permite escrever e validar o envio antes de a conta
// WhatsApp Business existir.
func canalApontandoPara(t *testing.T, srv *httptest.Server) disparo.Canal {
	t.Helper()
	c, err := disparo.NovoCanalMeta(disparo.ConfigMeta{
		IDNumero:  "1234567890",
		Token:     "token-de-teste",
		VersaoAPI: "v25.0",
		BaseURL:   srv.URL,
	})
	if err != nil {
		t.Fatalf("montar canal: %v", err)
	}
	return c
}

func mensagemDeTeste() disparo.Mensagem {
	return disparo.Mensagem{
		Telefone:   "5531988887777",
		Texto:      "Olá Mercado do João. O contrato CT-9001…",
		Template:   "arcom_cobranca_atraso_medio",
		Idioma:     "pt_BR",
		Parametros: []string{"Mercado do João", "CT-9001", "R$ 1.250,90", "40", "https://facilita.arcom.com.br/negociar/abc"},
	}
}

func TestEnvioMontaOCorpoQueAMetaEspera(t *testing.T) {
	var caminho, auth, tipoConteudo string
	var recebido map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caminho = r.URL.Path
		auth = r.Header.Get("Authorization")
		tipoConteudo = r.Header.Get("Content-Type")
		bruto, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bruto, &recebido)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.ABC123"}]}`))
	}))
	defer srv.Close()

	referencia, err := canalApontandoPara(t, srv).Enviar(context.Background(), mensagemDeTeste())
	if err != nil {
		t.Fatalf("Enviar: %v", err)
	}

	if referencia != "wamid.ABC123" {
		t.Errorf("referência = %q, quer o wamid devolvido pela Meta", referencia)
	}
	if caminho != "/v25.0/1234567890/messages" {
		t.Errorf("caminho = %q", caminho)
	}
	if auth != "Bearer token-de-teste" {
		t.Errorf("Authorization = %q", auth)
	}
	if tipoConteudo != "application/json" {
		t.Errorf("Content-Type = %q", tipoConteudo)
	}

	if recebido["messaging_product"] != "whatsapp" {
		t.Errorf("messaging_product = %v", recebido["messaging_product"])
	}
	if recebido["type"] != "template" {
		t.Errorf("type = %v — fora da janela de atendimento a Meta só aceita template", recebido["type"])
	}
	if recebido["to"] != "5531988887777" {
		t.Errorf("to = %v", recebido["to"])
	}

	template, _ := recebido["template"].(map[string]any)
	if template["name"] != "arcom_cobranca_atraso_medio" {
		t.Errorf("template.name = %v", template["name"])
	}
	if idioma, _ := template["language"].(map[string]any); idioma["code"] != "pt_BR" {
		t.Errorf("template.language.code = %v", idioma["code"])
	}

	componentes, _ := template["components"].([]any)
	if len(componentes) != 1 {
		t.Fatalf("components = %v, quer um componente de body", componentes)
	}
	body, _ := componentes[0].(map[string]any)
	if body["type"] != "body" {
		t.Errorf("components[0].type = %v", body["type"])
	}

	params, _ := body["parameters"].([]any)
	if len(params) != 5 {
		t.Fatalf("mandou %d parâmetros, quer 5 — a Meta recusa contagem diferente da do template aprovado", len(params))
	}
	// A ordem é o contrato com o template: {{1}} é o primeiro da lista.
	esperados := []string{"Mercado do João", "CT-9001", "R$ 1.250,90", "40", "https://facilita.arcom.com.br/negociar/abc"}
	for i, quer := range esperados {
		p, _ := params[i].(map[string]any)
		if p["type"] != "text" {
			t.Errorf("parâmetro %d: type = %v", i+1, p["type"])
		}
		if p["text"] != quer {
			t.Errorf("parâmetro %d = %q, quer %q", i+1, p["text"], quer)
		}
	}

	// O texto montado é registro nosso; não faz parte do que a Meta recebe.
	if _, tem := recebido["text"]; tem {
		t.Error("o corpo não deveria carregar texto livre")
	}
}

func TestEnvioSemTemplateEhRecusadoAntesDaRede(t *testing.T) {
	chamou := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { chamou = true }))
	defer srv.Close()

	m := mensagemDeTeste()
	m.Template = ""

	if _, err := canalApontandoPara(t, srv).Enviar(context.Background(), m); err == nil {
		t.Fatal("aceitou envio sem template aprovado")
	}
	if chamou {
		t.Error("chegou a chamar a rede sem template")
	}
}

// A Meta recusa parâmetro com quebra de linha, tabulação ou espaços demais —
// e devolve um erro genérico, difícil de rastrear. Um nome cadastrado com
// espaço duplo no ERP chegaria exatamente assim.
func TestParametrosSaoLimposAntesDeEnviar(t *testing.T) {
	var recebido map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bruto, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bruto, &recebido)
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.X"}]}`))
	}))
	defer srv.Close()

	m := mensagemDeTeste()
	m.Parametros = []string{"Mercado   do\tJoão", "CT-9001\nquebrado", "  R$ 10,00  ", "40", "https://x"}

	if _, err := canalApontandoPara(t, srv).Enviar(context.Background(), m); err != nil {
		t.Fatalf("Enviar: %v", err)
	}

	template, _ := recebido["template"].(map[string]any)
	componentes, _ := template["components"].([]any)
	body, _ := componentes[0].(map[string]any)
	params, _ := body["parameters"].([]any)

	quer := []string{"Mercado do João", "CT-9001 quebrado", "R$ 10,00", "40", "https://x"}
	for i, esperado := range quer {
		p, _ := params[i].(map[string]any)
		texto, _ := p["text"].(string)
		if texto != esperado {
			t.Errorf("parâmetro %d = %q, quer %q", i+1, texto, esperado)
		}
		if strings.ContainsAny(texto, "\n\t") || strings.Contains(texto, "  ") {
			t.Errorf("parâmetro %d ainda tem caractere que a Meta recusa: %q", i+1, texto)
		}
	}
}

// A separação entre recusa permanente e temporária é o que impede o worker de
// insistir três vezes num template reprovado.
func TestRecusaPermanenteEhDistinguidaDaTemporaria(t *testing.T) {
	casos := []struct {
		nome           string
		status         int
		corpo          string
		querPermanente bool
	}{
		{"template não aprovado", 400, `{"error":{"code":132001,"message":"Template not found"}}`, true},
		{"número não usa WhatsApp", 400, `{"error":{"code":131026,"message":"Receiver incapable"}}`, true},
		{"token expirado", 401, `{"error":{"code":190,"message":"Session expired"}}`, true},
		{"limite de envio", 429, `{"error":{"code":80007,"message":"Rate limit"}}`, false},
		{"instabilidade da Meta", 500, `{"error":{"code":1,"message":"Internal"}}`, false},
		{"indisponível", 503, `{"error":{"code":2,"message":"Service unavailable"}}`, false},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(c.status)
				_, _ = w.Write([]byte(c.corpo))
			}))
			defer srv.Close()

			_, err := canalApontandoPara(t, srv).Enviar(context.Background(), mensagemDeTeste())
			var meta *disparo.ErroMeta
			if !errors.As(err, &meta) {
				t.Fatalf("erro = %v, quer ErroMeta", err)
			}
			if meta.Permanente != c.querPermanente {
				t.Errorf("Permanente = %v, quer %v (status %d)", meta.Permanente, c.querPermanente, c.status)
			}
		})
	}
}

// O corpo da resposta da Meta pode trazer dado do cliente; ele vai para o log,
// nunca para a mensagem do erro que circula pela aplicação.
func TestErroNaoVazaOCorpoDaResposta(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":{"code":131026,"message":"5531988887777 nao recebe mensagem"}}`))
	}))
	defer srv.Close()

	_, err := canalApontandoPara(t, srv).Enviar(context.Background(), mensagemDeTeste())
	if err == nil {
		t.Fatal("deveria falhar")
	}
	if strings.Contains(err.Error(), "5531988887777") {
		t.Errorf("o telefone do cliente vazou na mensagem de erro: %v", err)
	}
}

// Sem o wamid não há como conciliar a entrega depois. Gravar como enviado
// nesse caso seria perder o rastro em silêncio.
func TestRespostaSemIdDaMensagemFalha(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"messages":[]}`))
	}))
	defer srv.Close()

	if _, err := canalApontandoPara(t, srv).Enviar(context.Background(), mensagemDeTeste()); err == nil {
		t.Fatal("aceitou resposta sem o id da mensagem")
	}
}

func TestCanalExigeCredenciais(t *testing.T) {
	if _, err := disparo.NovoCanalMeta(disparo.ConfigMeta{Token: "só o token"}); err == nil {
		t.Error("aceitou canal sem o id do número")
	}
	if _, err := disparo.NovoCanalMeta(disparo.ConfigMeta{IDNumero: "só o número"}); err == nil {
		t.Error("aceitou canal sem token")
	}
}

func TestValoresNaOrdemSegueACampanha(t *testing.T) {
	v := disparo.Variaveis{
		Tratamento: "Mercado do João", NomeCompleto: "Mercado do João LTDA",
		Contrato: "CT-1", Dias: 40, Valor: 1250.90, Link: "https://x",
	}

	valores, err := disparo.ValoresNaOrdem([]string{"nome", "contrato", "valor", "dias", "link"}, v)
	if err != nil {
		t.Fatalf("ValoresNaOrdem: %v", err)
	}
	quer := []string{"Mercado do João", "CT-1", "R$ 1.250,90", "40", "https://x"}
	for i := range quer {
		if valores[i] != quer[i] {
			t.Errorf("posição %d = %q, quer %q", i+1, valores[i], quer[i])
		}
	}

	// Ordem diferente na campanha tem que produzir ordem diferente aqui —
	// senão o {{1}} do template aprovado receberia o valor errado.
	invertido, _ := disparo.ValoresNaOrdem([]string{"link", "nome"}, v)
	if invertido[0] != "https://x" || invertido[1] != "Mercado do João" {
		t.Errorf("a ordem da campanha não foi respeitada: %v", invertido)
	}

	// Parâmetro inexistente é erro no enfileiramento, e não string vazia
	// chegando na Meta com recusa genérica.
	if _, err := disparo.ValoresNaOrdem([]string{"nao_existe"}, v); err == nil {
		t.Error("aceitou parâmetro que a campanha não sabe preencher")
	}
}
