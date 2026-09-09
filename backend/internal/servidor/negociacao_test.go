package servidor_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// gerarLink pede o link público de uma dívida e devolve só o token — é o que
// o cliente recebe no WhatsApp.
func gerarLink(t *testing.T, h http.Handler, cookie *http.Cookie, divida uuid.UUID) string {
	t.Helper()

	rec := chamar(t, h, http.MethodPost, "/api/v1/carteira/"+divida.String()+"/link", nil, cookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("gerar link: status %d (%s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Link string `json:"link"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("resposta inesperada: %s", rec.Body.String())
	}
	if !strings.HasPrefix(resp.Link, "https://facilita.arcom.com.br/negociar/") {
		t.Fatalf("link fora do endereço público da aplicação: %s", resp.Link)
	}
	return resp.Link[strings.LastIndex(resp.Link, "/")+1:]
}

func TestTokenNaoEhGravadoEmClaro(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	token := gerarLink(t, h, cookie, divida)

	// Um dump do banco não pode devolver links de negociação utilizáveis.
	var achou int64
	if err := gdb.Raw(
		`SELECT count(*) FROM dividas WHERE encode(token_hash, 'escape') LIKE ?`, "%"+token+"%",
	).Scan(&achou).Error; err != nil {
		t.Fatalf("consultar token: %v", err)
	}
	if achou != 0 {
		t.Error("o token em claro foi parar no banco — deveria estar só o hash")
	}

	var temHash bool
	if err := gdb.Raw(`SELECT token_hash IS NOT NULL FROM dividas WHERE id = ?`, divida).Scan(&temHash).Error; err != nil {
		t.Fatalf("consultar hash: %v", err)
	}
	if !temHash {
		t.Error("nenhum hash foi gravado")
	}
}

func TestClienteNegociaPeloLinkSemLogin(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	// 40 dias: faixa 31-60, política de 5% à vista / 10 parcelas... (a semente
	// da migration dá 10% à vista, 5% parcelado, até 6x, entrada de 10%).
	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	token := gerarLink(t, h, cookie, divida)

	// Sem cookie nenhum: é o cliente devedor abrindo do celular.
	rec := chamar(t, h, http.MethodGet, "/api/v1/negociar/"+token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("consultar proposta: status %d (%s)", rec.Code, rec.Body.String())
	}

	var p struct {
		Cliente    string `json:"cliente"`
		Contrato   string `json:"contrato"`
		DiasAtraso int    `json:"diasAtraso"`
		Ofertas    struct {
			Avista struct {
				DescontoPct float64 `json:"descontoPct"`
				ValorTotal  float64 `json:"valorTotal"`
			} `json:"avista"`
			Parcelado *struct {
				MaxParcelas int `json:"maxParcelas"`
			} `json:"parcelado"`
		} `json:"ofertas"`
		Acordo any `json:"acordo"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("proposta inesperada: %s", rec.Body.String())
	}

	if p.DiasAtraso != 40 {
		t.Errorf("diasAtraso = %d, quer 40", p.DiasAtraso)
	}
	if p.Ofertas.Avista.DescontoPct != 10 {
		t.Errorf("desconto à vista = %v, quer 10 (política da faixa 31-60)", p.Ofertas.Avista.DescontoPct)
	}
	if p.Ofertas.Avista.ValorTotal != 900 {
		t.Errorf("valor à vista = %v, quer 900", p.Ofertas.Avista.ValorTotal)
	}
	if p.Acordo != nil {
		t.Error("dívida sem acordo não pode vir com acordo na proposta")
	}

	// Só o primeiro nome sai — um link vazado não vira consulta à ficha.
	if strings.Contains(p.Cliente, " ") {
		t.Errorf("proposta expôs nome completo: %q", p.Cliente)
	}
	if strings.Contains(rec.Body.String(), "11111111111") {
		t.Error("o CPF/CNPJ do cliente vazou na tela pública")
	}
	if strings.Contains(rec.Body.String(), "31988887777") {
		t.Error("o telefone do cliente vazou na tela pública")
	}

	// Aceita parcelado.
	if p.Ofertas.Parcelado == nil {
		t.Fatal("faixa 31-60 deveria oferecer parcelamento")
	}
	rec = chamar(t, h, http.MethodPost, "/api/v1/negociar/"+token+"/aceitar",
		map[string]any{"tipo": "parcelado", "parcelas": 6})
	if rec.Code != http.StatusCreated {
		t.Fatalf("aceitar: status %d (%s)", rec.Code, rec.Body.String())
	}

	// A dívida virou negociada e o acordo aparece na consulta seguinte.
	var status string
	if err := gdb.Raw(`SELECT status FROM dividas WHERE id = ?`, divida).Scan(&status).Error; err != nil {
		t.Fatalf("consultar status: %v", err)
	}
	if status != "negociado" {
		t.Errorf("status da dívida = %q, quer negociado", status)
	}
}

func TestAceiteEmDobroNoMesmoLinkEhRecusado(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	token := gerarLink(t, h, cookie, divida)
	corpo := map[string]any{"tipo": "avista", "parcelas": 1}

	if rec := chamar(t, h, http.MethodPost, "/api/v1/negociar/"+token+"/aceitar", corpo); rec.Code != http.StatusCreated {
		t.Fatalf("primeiro aceite: status %d (%s)", rec.Code, rec.Body.String())
	}
	// Clicar duas vezes no botão não pode gerar dois acordos.
	if rec := chamar(t, h, http.MethodPost, "/api/v1/negociar/"+token+"/aceitar", corpo); rec.Code != http.StatusConflict {
		t.Fatalf("segundo aceite: status %d, quer 409 (%s)", rec.Code, rec.Body.String())
	}

	var quantos int64
	if err := gdb.Raw(`SELECT count(*) FROM acordos WHERE divida_id = ?`, divida).Scan(&quantos).Error; err != nil {
		t.Fatalf("contar acordos: %v", err)
	}
	if quantos != 1 {
		t.Errorf("gravou %d acordos para a mesma dívida, quer 1", quantos)
	}
}

func TestLinkExpiradoNaoAbre(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	token := gerarLink(t, h, cookie, divida)

	// Envelhece o link. No modelo antigo o token não tinha validade: um link
	// de um WhatsApp de meses atrás continuava abrindo a proposta.
	if err := gdb.Exec(`UPDATE dividas SET token_expira_em = ? WHERE id = ?`,
		time.Now().UTC().Add(-time.Hour), divida).Error; err != nil {
		t.Fatalf("expirar token: %v", err)
	}

	rec := chamar(t, h, http.MethodGet, "/api/v1/negociar/"+token, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, quer 404 (%s)", rec.Code, rec.Body.String())
	}

	var p struct {
		Codigo string `json:"codigo"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &p)
	if p.Codigo != "link_invalido" {
		t.Errorf("codigo = %q, quer link_invalido", p.Codigo)
	}
}

func TestTokenInventadoNaoAbre(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)

	for _, token := range []string{
		"curto",
		strings.Repeat("a", 43), // do tamanho certo, mas não existe
		"../../etc/passwd",
	} {
		rec := chamar(t, h, http.MethodGet, "/api/v1/negociar/"+token, nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("token %q: status = %d, quer 404", token, rec.Code)
		}
	}
}

// Renovar o link tem que invalidar o anterior — senão revogar um link vazado
// seria impossível.
func TestGerarLinkNovoInvalidaOAnterior(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	antigo := gerarLink(t, h, cookie, divida)
	novo := gerarLink(t, h, cookie, divida)

	if antigo == novo {
		t.Fatal("o segundo link saiu igual ao primeiro")
	}
	if rec := chamar(t, h, http.MethodGet, "/api/v1/negociar/"+antigo, nil); rec.Code != http.StatusNotFound {
		t.Errorf("link antigo: status = %d, quer 404", rec.Code)
	}
	if rec := chamar(t, h, http.MethodGet, "/api/v1/negociar/"+novo, nil); rec.Code != http.StatusOK {
		t.Errorf("link novo: status = %d, quer 200", rec.Code)
	}
}

func TestAceiteComTipoOuParcelasInvalidasDevolve422(t *testing.T) {
	h, gdb := montarServidor(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	token := gerarLink(t, h, cookie, divida)

	casos := []map[string]any{
		{"tipo": "boleto", "parcelas": 1},
		{"tipo": "avista", "parcelas": 0},
		{"tipo": "parcelado", "parcelas": 25},
	}
	for _, corpo := range casos {
		rec := chamar(t, h, http.MethodPost, "/api/v1/negociar/"+token+"/aceitar", corpo)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%v: status = %d, quer 422 (%s)", corpo, rec.Code, rec.Body.String())
		}
	}
}

// É a única superfície do sistema aberta na internet sem sessão — o rate
// limit por IP (30/min) é o que impede varredura de token.
func TestNegociarTemRateLimitPorIP(t *testing.T) {
	h := servidorComBanco(t)

	bloqueou := false
	for i := 0; i < 31; i++ {
		rec := chamar(t, h, http.MethodGet, "/api/v1/negociar/"+strings.Repeat("a", tamanhoTokenTeste), nil)
		if rec.Code == http.StatusTooManyRequests {
			bloqueou = true
			break
		}
	}
	if !bloqueou {
		t.Fatal("31 tentativas em um minuto e nenhuma foi bloqueada")
	}
}

const tamanhoTokenTeste = 43
