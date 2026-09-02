package gatewayarcom

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// clienteApontandoPara devolve um Cliente falando com um servidor de teste.
// Nenhum teste aqui toca o Gateway de verdade: ele é produção da empresa, e
// a chave só existe lá.
func clienteApontandoPara(t *testing.T, srv *httptest.Server, chave string) *Cliente {
	t.Helper()
	return NovoCliente(chave).ComBaseURL(srv.URL)
}

func TestConsultarMandaAChaveEMontaOCaminho(t *testing.T) {
	var caminhoRecebido, chaveRecebida, queryRecebida string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caminhoRecebido = r.URL.Path
		chaveRecebida = r.Header.Get("X-API-Key")
		queryRecebida = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"itens":[]}`))
	}))
	defer srv.Close()

	c := clienteApontandoPara(t, srv, "chave-de-teste")

	filtros := url.Values{}
	filtros.Set("responsavel_cobranca.keyword", "ANA")
	filtros.Set("limite", "20")

	if _, err := c.Consultar(context.Background(), DatasetDebitos, filtros); err != nil {
		t.Fatalf("Consultar: %v", err)
	}

	if caminhoRecebido != "/v1/debitos" {
		t.Errorf("caminho = %q, quer /v1/debitos", caminhoRecebido)
	}
	if chaveRecebida != "chave-de-teste" {
		t.Errorf("X-API-Key = %q", chaveRecebida)
	}
	if !strings.Contains(queryRecebida, "limite=20") {
		t.Errorf("query = %q, quer conter limite=20", queryRecebida)
	}
}

func TestAgregarUsaOSufixoAgregado(t *testing.T) {
	var caminho string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caminho = r.URL.Path
		_, _ = w.Write([]byte(`{"itens":[]}`))
	}))
	defer srv.Close()

	c := clienteApontandoPara(t, srv, "k")
	if _, err := c.Agregar(context.Background(), DatasetDebitos, nil); err != nil {
		t.Fatalf("Agregar: %v", err)
	}
	if caminho != "/v1/debitos/agregado" {
		t.Errorf("caminho = %q, quer /v1/debitos/agregado", caminho)
	}
}

func TestDatasetForaDoPadraoEhRecusadoAntesDaRede(t *testing.T) {
	chamou := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { chamou = true }))
	defer srv.Close()

	c := clienteApontandoPara(t, srv, "k")

	// Inclui uma tentativa de sair do /v1 pelo caminho — dataset entra na URL.
	for _, d := range []Dataset{"inventado", "../admin", "debitos/../../etc"} {
		if _, err := c.Consultar(context.Background(), d, nil); !errors.Is(err, ErrDatasetDesconhecido) {
			t.Errorf("dataset %q: erro = %v, quer ErrDatasetDesconhecido", d, err)
		}
	}
	if chamou {
		t.Error("dataset inválido chegou a fazer requisição de rede")
	}
}

func TestSemChaveNaoChamaARede(t *testing.T) {
	chamou := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { chamou = true }))
	defer srv.Close()

	c := clienteApontandoPara(t, srv, "")
	if c.TemCredencial() {
		t.Error("TemCredencial devolveu true sem chave")
	}
	if _, err := c.Consultar(context.Background(), DatasetDebitos, nil); !errors.Is(err, ErrSemCredencial) {
		t.Errorf("erro = %v, quer ErrSemCredencial", err)
	}
	if chamou {
		t.Error("chamou a rede sem credencial")
	}
}

func TestRespostaDeErroViraErroTipado(t *testing.T) {
	casos := []struct {
		status int
		quer   error
	}{
		{http.StatusUnauthorized, ErrNaoAutorizado},
		{http.StatusForbidden, ErrNaoAutorizado},
	}

	for _, c := range casos {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(c.status)
			_, _ = w.Write([]byte(`{"erro":"sem permissao"}`))
		}))

		cli := clienteApontandoPara(t, srv, "k")
		_, err := cli.Consultar(context.Background(), DatasetDebitos, nil)
		if !errors.Is(err, c.quer) {
			t.Errorf("status %d: erro = %v, quer %v", c.status, err, c.quer)
		}
		srv.Close()
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`falha interna do gateway`))
	}))
	defer srv.Close()

	_, err := clienteApontandoPara(t, srv, "k").Consultar(context.Background(), DatasetDebitos, nil)
	var resp *ErroResposta
	if !errors.As(err, &resp) || resp.Status != http.StatusInternalServerError {
		t.Fatalf("erro = %v, quer ErroResposta 500", err)
	}
	// A mensagem do erro não pode carregar o corpo do Gateway: ela circula em
	// log e o corpo pode trazer dado de cliente.
	if strings.Contains(err.Error(), "falha interna do gateway") {
		t.Error("o corpo da resposta do gateway vazou na mensagem de erro")
	}
}

func TestDecodificarListaAceitaOsEnvelopesPlausiveis(t *testing.T) {
	casos := map[string]string{
		"itens":      `{"itens":[{"seq_debito":"1"}]}`,
		"data":       `{"data":[{"seq_debito":"1"}]}`,
		"hits":       `{"hits":[{"seq_debito":"1"}]}`,
		"results":    `{"results":[{"seq_debito":"1"}]}`,
		"array puro": `[{"seq_debito":"1"}]`,
	}

	for nome, corpo := range casos {
		t.Run(nome, func(t *testing.T) {
			var debitos []Debito
			if err := decodificarLista([]byte(corpo), &debitos); err != nil {
				t.Fatalf("decodificar: %v", err)
			}
			if len(debitos) != 1 || debitos[0].SeqDebito != "1" {
				t.Errorf("decodificou %+v", debitos)
			}
		})
	}
}

// Envelope diferente do previsto tem que falhar alto: se devolvesse lista
// vazia, uma carteira inteira sumiria parecendo "nenhum inadimplente hoje".
func TestEnvelopeDesconhecidoFalhaEmVezDeDevolverVazio(t *testing.T) {
	var debitos []Debito
	err := decodificarLista([]byte(`{"payload":{"linhas":[{"seq_debito":"1"}]}}`), &debitos)
	if !errors.Is(err, ErrEnvelopeDesconhecido) {
		t.Fatalf("erro = %v, quer ErrEnvelopeDesconhecido", err)
	}
}

func TestMapearRecusaLinhaIncompleta(t *testing.T) {
	casos := []struct {
		nome   string
		debito Debito
	}{
		{"sem documento", Debito{SeqDebito: "1", DataVencto: "2026-07-01", ValorLiquido: 100}},
		{"sem vencimento", Debito{SeqDebito: "1", CNPJ: "12345678000199", ValorLiquido: 100}},
		{"sem valor", Debito{SeqDebito: "1", CNPJ: "12345678000199", DataVencto: "2026-07-01"}},
		{"valor negativo", Debito{SeqDebito: "1", CNPJ: "12345678000199", DataVencto: "2026-07-01", ValorLiquido: -5}},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if _, err := Mapear(c.debito, MapeamentoPadrao()); err == nil {
				t.Error("aceitou linha que não dá pra cobrar")
			}
		})
	}
}

func TestMapearNormalizaOQueEntra(t *testing.T) {
	linha, err := Mapear(Debito{
		SeqDebito:           "CT-9001",
		CNPJ:                "12.345.678/0001-99",
		RazaoSocial:         "Mercado do João LTDA",
		Cliente:             "MERCADO JOAO",
		DataVencto:          "2026-07-01T00:00:00Z",
		ValorLiquido:        1250.90,
		DebValorDocumento:   1400,
		Filial:              "BH",
		ResponsavelCobranca: "ANA",
	}, MapeamentoPadrao())
	if err != nil {
		t.Fatalf("Mapear: %v", err)
	}

	if linha.Documento != "12345678000199" {
		t.Errorf("documento = %q, quer só dígitos", linha.Documento)
	}
	if linha.Vencimento != "2026-07-01" {
		t.Errorf("vencimento = %q, quer a data sem a hora", linha.Vencimento)
	}
	if linha.NomeCliente != "Mercado do João LTDA" {
		t.Errorf("nome = %q, quer a razão social", linha.NomeCliente)
	}
	if linha.Valor != 1250.90 {
		t.Errorf("valor = %v, quer o líquido do débito", linha.Valor)
	}
}
