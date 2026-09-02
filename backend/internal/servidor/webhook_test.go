package servidor_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorm.io/gorm"

	"facilitaarcom/internal/config"
	"facilitaarcom/internal/servidor"
)

const (
	appSecretDeTeste   = "segredo-da-aplicacao-de-teste"
	verifyTokenDeTeste = "token-combinado-com-a-meta"
)

// servidorComWebhook sobe o servidor com as credenciais do WhatsApp
// preenchidas, que é o que faz a rota do webhook existir.
func servidorComWebhook(t *testing.T) (http.Handler, *gorm.DB) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		Env: "development", Port: "3000", AppURL: "https://facilita.arcom.com.br",
		WhatsAppIDNumero:    "1234567890",
		WhatsAppToken:       "token",
		WhatsAppAppSecret:   appSecretDeTeste,
		WhatsAppVerifyToken: verifyTokenDeTeste,
	}
	gdb := bancoDeTeste(t)
	return servidor.Novo(cfg, log, gdb, nil), gdb
}

func assinar(corpo []byte) string {
	mac := hmac.New(sha256.New, []byte(appSecretDeTeste))
	mac.Write(corpo)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// postarWebhook manda o evento com a assinatura que a Meta mandaria.
func postarWebhook(t *testing.T, h http.Handler, corpo []byte, assinatura string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/whatsapp", bytes.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	if assinatura != "" {
		req.Header.Set("X-Hub-Signature-256", assinatura)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func eventoDeMensagem(de, wamid, texto string, quando time.Time) []byte {
	return []byte(fmt.Sprintf(`{
	  "object":"whatsapp_business_account",
	  "entry":[{"id":"102290129340398","changes":[{"field":"messages","value":{
	    "messaging_product":"whatsapp",
	    "metadata":{"display_phone_number":"553133334444","phone_number_id":"1234567890"},
	    "contacts":[{"profile":{"name":"João do Mercado"},"wa_id":"%s"}],
	    "messages":[{"from":"%s","id":"%s","timestamp":"%d","type":"text","text":{"body":"%s"}}]
	  }}]}]
	}`, de, de, wamid, quando.Unix(), texto))
}

// A assinatura é a única coisa que separa uma resposta real de uma forjada:
// o endereço é público e qualquer um pode chamá-lo.
func TestWebhookRecusaAssinaturaInvalida(t *testing.T) {
	h, gdb := servidorComWebhook(t)
	corpo := eventoDeMensagem("5531988887777", "wamid.FORJADO", "quero negociar", time.Now())

	casos := []struct {
		nome       string
		assinatura string
	}{
		{"sem assinatura", ""},
		{"assinatura vazia", "sha256="},
		{"assinatura de outro segredo", "sha256=" + hex.EncodeToString([]byte("nao-e-hmac-valido-mas-tem-tamanho"))},
		{"sem o prefixo sha256", hex.EncodeToString([]byte("qualquer coisa"))},
		{"hexadecimal inválido", "sha256=zzzz"},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			rec := postarWebhook(t, h, corpo, c.assinatura)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, quer 401", rec.Code)
			}
		})
	}

	var quantas int64
	if err := gdb.Raw(`SELECT count(*) FROM mensagens`).Scan(&quantas).Error; err != nil {
		t.Fatalf("contar: %v", err)
	}
	if quantas != 0 {
		t.Errorf("gravou %d mensagem(ns) de evento não autenticado", quantas)
	}
}

// Assinatura sobre o corpo cru: qualquer alteração no payload invalida.
func TestWebhookRecusaCorpoAlterado(t *testing.T) {
	h, _ := servidorComWebhook(t)

	original := eventoDeMensagem("5531988887777", "wamid.A", "original", time.Now())
	assinatura := assinar(original)
	adulterado := bytes.Replace(original, []byte("original"), []byte("adultera"), 1)

	if rec := postarWebhook(t, h, adulterado, assinatura); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, quer 401 — corpo alterado com assinatura do original", rec.Code)
	}
}

func TestWebhookAceitaEventoAssinadoEGravaAConversa(t *testing.T) {
	h, gdb := servidorComWebhook(t)

	quando := time.Now().UTC().Truncate(time.Second)
	corpo := eventoDeMensagem("5531988887777", "wamid.REAL1", "quero negociar minha dívida", quando)

	if rec := postarWebhook(t, h, corpo, assinar(corpo)); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, quer 200 (%s)", rec.Code, rec.Body.String())
	}

	var telefone, nomePerfil string
	var naoLidas int
	var janela *time.Time
	if err := gdb.Raw(
		`SELECT telefone, coalesce(nome_perfil,''), nao_lidas, janela_expira_em FROM conversas`,
	).Row().Scan(&telefone, &nomePerfil, &naoLidas, &janela); err != nil {
		t.Fatalf("consultar conversa: %v", err)
	}

	if telefone != "5531988887777" {
		t.Errorf("telefone = %q", telefone)
	}
	if nomePerfil != "João do Mercado" {
		t.Errorf("nome de perfil = %q", nomePerfil)
	}
	if naoLidas != 1 {
		t.Errorf("não lidas = %d, quer 1", naoLidas)
	}
	// A resposta do cliente abre 24h de janela — é o que torna a réplica
	// gratuita e o que a mesa usa para saber se pode mandar texto livre.
	if janela == nil {
		t.Fatal("a janela de atendimento não foi aberta")
	}
	if diff := janela.Sub(quando); diff < 23*time.Hour || diff > 25*time.Hour {
		t.Errorf("janela expira em %v depois da mensagem, quer ~24h", diff)
	}

	var direcao, texto string
	if err := gdb.Raw(`SELECT direcao, coalesce(texto,'') FROM mensagens`).Row().Scan(&direcao, &texto); err != nil {
		t.Fatalf("consultar mensagem: %v", err)
	}
	if direcao != "entrada" || texto != "quero negociar minha dívida" {
		t.Errorf("mensagem gravada = %q / %q", direcao, texto)
	}
}

// A Meta reentrega o webhook quando a nossa resposta demora ou falha. Sem
// idempotência, a mesma fala apareceria duas vezes para o operador.
func TestWebhookReentregueNaoDuplicaMensagem(t *testing.T) {
	h, gdb := servidorComWebhook(t)
	corpo := eventoDeMensagem("5531988887777", "wamid.MESMO", "oi", time.Now())

	for i := range 3 {
		if rec := postarWebhook(t, h, corpo, assinar(corpo)); rec.Code != http.StatusOK {
			t.Fatalf("entrega %d: status %d", i+1, rec.Code)
		}
	}

	var mensagens, conversas, naoLidas int64
	_ = gdb.Raw(`SELECT count(*) FROM mensagens`).Scan(&mensagens).Error
	_ = gdb.Raw(`SELECT count(*) FROM conversas`).Scan(&conversas).Error
	_ = gdb.Raw(`SELECT nao_lidas FROM conversas LIMIT 1`).Scan(&naoLidas).Error

	if mensagens != 1 {
		t.Errorf("gravou %d mensagens para o mesmo wamid, quer 1", mensagens)
	}
	if conversas != 1 {
		t.Errorf("criou %d conversas para o mesmo telefone, quer 1", conversas)
	}
	if naoLidas != 1 {
		t.Errorf("contou %d não lidas, quer 1 — reentrega não pode inflar o contador", naoLidas)
	}
}

// O aviso de entrega chega pelo wamid que guardamos ao enviar, e precisa
// aparecer na linha da régua que o operador acompanha.
func TestWebhookDeStatusAtualizaODisparo(t *testing.T) {
	h, gdb := servidorComWebhook(t)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	divida := semearDivida(t, gdb, "11111111111", "CT-1", "", 1000, 40)
	if status, _ := enfileirar(t, h, cookie, divida); status != http.StatusCreated {
		t.Fatal("enfileirar falhou")
	}
	// Simula o envio: é o worker que preencheria isto ao entregar.
	if err := gdb.Exec(
		`UPDATE disparos SET status='enviado', referencia_externa='wamid.SAIDA1', enviado_em=now() WHERE divida_id = ?`,
		divida,
	).Error; err != nil {
		t.Fatalf("marcar enviado: %v", err)
	}

	agora := time.Now().UTC()
	statusMeta := func(estado string, quando time.Time) []byte {
		return []byte(fmt.Sprintf(`{
		  "object":"whatsapp_business_account",
		  "entry":[{"id":"1","changes":[{"field":"messages","value":{
		    "messaging_product":"whatsapp",
		    "metadata":{"phone_number_id":"1234567890"},
		    "statuses":[{"id":"wamid.SAIDA1","status":"%s","timestamp":"%d","recipient_id":"5531988887777"}]
		  }}]}]
		}`, estado, quando.Unix()))
	}

	for _, estado := range []string{"sent", "delivered", "read"} {
		corpo := statusMeta(estado, agora)
		if rec := postarWebhook(t, h, corpo, assinar(corpo)); rec.Code != http.StatusOK {
			t.Fatalf("status %s: código %d (%s)", estado, rec.Code, rec.Body.String())
		}
	}

	var entregue, lido *time.Time
	if err := gdb.Raw(
		`SELECT entregue_em, lido_em FROM disparos WHERE divida_id = ?`, divida,
	).Row().Scan(&entregue, &lido); err != nil {
		t.Fatalf("consultar disparo: %v", err)
	}
	if entregue == nil {
		t.Error("entregue_em ficou nulo depois do aviso de delivered")
	}
	if lido == nil {
		t.Error("lido_em ficou nulo depois do aviso de read")
	}
}

// O handshake de cadastro: a Meta chama com o token combinado e espera o
// desafio de volta em texto puro.
func TestWebhookVerificacaoDeCadastro(t *testing.T) {
	h, _ := servidorComWebhook(t)

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token="+verifyTokenDeTeste+"&hub.challenge=123456", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, quer 200", rec.Code)
	}
	if rec.Body.String() != "123456" {
		t.Errorf("corpo = %q, quer o desafio devolvido cru", rec.Body.String())
	}

	// Token errado não passa, e a resposta não diz o que estava errado.
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=errado&hub.challenge=123456", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("token errado: status = %d, quer 403", rec.Code)
	}
	if rec.Body.String() == "123456" {
		t.Error("devolveu o desafio mesmo com o token errado")
	}
}

// Sem segredo configurado a rota não existe: um webhook público que não
// confere assinatura aceitaria mensagem de cliente forjada.
func TestWebhookNaoExisteSemSegredo(t *testing.T) {
	h := servidorComBanco(t) // config sem as credenciais do WhatsApp

	corpo := eventoDeMensagem("5531988887777", "wamid.X", "oi", time.Now())
	rec := postarWebhook(t, h, corpo, assinar(corpo))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, quer 404 — a rota não deveria existir", rec.Code)
	}
}

// Corpo que não é o evento esperado devolve 200 de propósito: erro faria a
// Meta reentregar para sempre o mesmo payload indigesto.
func TestWebhookComCorpoEstranhoNaoPedeReentrega(t *testing.T) {
	h, _ := servidorComWebhook(t)

	corpo := []byte(`isto não é json`)
	if rec := postarWebhook(t, h, corpo, assinar(corpo)); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, quer 200", rec.Code)
	}
}

// A conversa é dado de devedor e segue o mesmo recorte do resto do sistema.
func TestConversaRespeitaOEscopoDeCarteira(t *testing.T) {
	h, gdb := servidorComWebhook(t)
	criarGerencia(t, h)
	cookieGerencia := logar(t, h, emailGerencia, senhaGerencia)

	// Cliente do Bruno responde.
	semearDivida(t, gdb, "33333333333", "CT-BRUNO", "BRUNO", 1000, 40)
	corpo := eventoDeMensagem("5531988887777", "wamid.BRUNO", "oi", time.Now())
	if rec := postarWebhook(t, h, corpo, assinar(corpo)); rec.Code != http.StatusOK {
		t.Fatalf("webhook: status %d", rec.Code)
	}

	cookieAna := criarOperador(t, h, cookieGerencia, "ana@arcom.com.br", "analista", "ANA")

	rec := chamar(t, h, http.MethodGet, "/api/v1/conversas", nil, cookieAna)
	if rec.Code != http.StatusOK {
		t.Fatalf("listar: status %d (%s)", rec.Code, rec.Body.String())
	}
	var lista struct {
		Itens []struct {
			ID string `json:"id"`
		} `json:"itens"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &lista); err != nil {
		t.Fatalf("resposta inesperada: %s", rec.Body.String())
	}
	if len(lista.Itens) != 0 {
		t.Errorf("analista viu %d conversa(s) de carteira alheia", len(lista.Itens))
	}

	// A gerência vê, e o telefone sai mascarado.
	rec = chamar(t, h, http.MethodGet, "/api/v1/conversas", nil, cookieGerencia)
	if rec.Code != http.StatusOK {
		t.Fatalf("listar como gerência: status %d", rec.Code)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("5531988887777")) {
		t.Errorf("telefone completo apareceu na listagem: %s", rec.Body.String())
	}
}
