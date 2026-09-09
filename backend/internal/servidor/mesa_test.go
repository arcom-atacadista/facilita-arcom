package servidor_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"facilitaarcom/internal/config"
	"facilitaarcom/internal/servidor"
)

// metaFalsa é a Cloud API do lado do teste: guarda o corpo de cada envio para
// o teste conferir o que de fato viajaria, e responde como a Meta responde.
type metaFalsa struct {
	mu       sync.Mutex
	corpos   []map[string]any
	servidor *httptest.Server
}

func novaMetaFalsa(t *testing.T) *metaFalsa {
	t.Helper()
	m := &metaFalsa{}
	m.servidor = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var corpo map[string]any
		if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		m.corpos = append(m.corpos, corpo)
		m.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.RESPOSTA"}]}`))
	}))
	t.Cleanup(m.servidor.Close)
	return m
}

func (m *metaFalsa) enviados() []map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]map[string]any(nil), m.corpos...)
}

// servidorComMesa sobe o servidor com a Cloud API apontada para a Meta falsa,
// que é o que faz a mesa poder responder sem falar com a Meta de verdade.
func servidorComMesa(t *testing.T, meta *metaFalsa) (http.Handler, *gorm.DB) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	base := ""
	if meta != nil {
		base = meta.servidor.URL
	}
	cfg := &config.Config{
		Env: "development", Port: "3000", AppURL: "https://facilita.arcom.com.br",
		WhatsAppIDNumero:    "1234567890",
		WhatsAppToken:       "token-de-teste",
		WhatsAppAppSecret:   appSecretDeTeste,
		WhatsAppVerifyToken: verifyTokenDeTeste,
		WhatsAppBaseURL:     base,
	}
	gdb := bancoDeTeste(t)
	return servidor.Novo(cfg, log, gdb, nil), gdb
}

// conversaAberta faz o cliente escrever, que é o que abre a janela de 24h — o
// mesmo caminho da produção, em vez de inserir a conversa direto no banco.
func conversaAberta(t *testing.T, h http.Handler, telefone, wamid string) string {
	t.Helper()
	corpo := eventoDeMensagem(telefone, wamid, "quero negociar minha dívida", time.Now())
	if rec := postarWebhook(t, h, corpo, assinar(corpo)); rec.Code != http.StatusOK {
		t.Fatalf("webhook de entrada: status %d (%s)", rec.Code, rec.Body.String())
	}

	cookie := logar(t, h, emailGerencia, senhaGerencia)
	rec := chamar(t, h, http.MethodGet, "/api/v1/conversas", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("listar conversas: status %d (%s)", rec.Code, rec.Body.String())
	}

	var resposta struct {
		Itens []struct {
			ID           string `json:"id"`
			JanelaAberta bool   `json:"janelaAberta"`
		} `json:"itens"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("ler listagem: %v", err)
	}
	if len(resposta.Itens) != 1 {
		t.Fatalf("conversas = %d, quer 1", len(resposta.Itens))
	}
	if !resposta.Itens[0].JanelaAberta {
		t.Fatal("a janela devia estar aberta logo depois da mensagem do cliente")
	}
	return resposta.Itens[0].ID
}

func TestMesaRespondeDentroDaJanelaComTextoLivre(t *testing.T) {
	meta := novaMetaFalsa(t)
	h, _ := servidorComMesa(t, meta)
	criarGerencia(t, h)

	id := conversaAberta(t, h, "5531988887777", "wamid.CLIENTE1")
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	rec := chamar(t, h, http.MethodPost, "/api/v1/conversas/"+id+"/mensagens",
		map[string]string{"texto": "Bom dia! Consigo dividir em 3 vezes, fecha assim?"}, cookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("responder: status %d, quer 201 (%s)", rec.Code, rec.Body.String())
	}

	enviados := meta.enviados()
	if len(enviados) != 1 {
		t.Fatalf("envios para a Meta = %d, quer 1", len(enviados))
	}
	corpo := enviados[0]

	// Dentro da janela vai texto livre. Se fosse template, a mensagem seria
	// tarifada — o ponto inteiro da mesa é responder de graça.
	if corpo["type"] != "text" {
		t.Errorf("type = %v, quer \"text\"", corpo["type"])
	}
	if _, tem := corpo["template"]; tem {
		t.Error("o corpo levou \"template\" num envio de texto — a Meta recusa com 400")
	}
	texto, _ := corpo["text"].(map[string]any)
	if texto["body"] != "Bom dia! Consigo dividir em 3 vezes, fecha assim?" {
		t.Errorf("body = %v", texto["body"])
	}

	// A resposta tem que aparecer no histórico, senão o próximo analista a
	// abrir a conversa não sabe o que já foi dito.
	rec = chamar(t, h, http.MethodGet, "/api/v1/conversas/"+id+"/mensagens", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("histórico: status %d (%s)", rec.Code, rec.Body.String())
	}
	var hist struct {
		Itens []struct {
			Direcao string  `json:"direcao"`
			Texto   *string `json:"texto"`
			Status  *string `json:"status"`
		} `json:"itens"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &hist); err != nil {
		t.Fatalf("ler histórico: %v", err)
	}

	var saidas int
	for _, m := range hist.Itens {
		if m.Direcao == "saida" {
			saidas++
			if m.Status == nil || *m.Status != "enviada" {
				t.Errorf("status da saída = %v, quer \"enviada\"", m.Status)
			}
		}
	}
	if saidas != 1 {
		t.Errorf("mensagens de saída no histórico = %d, quer 1", saidas)
	}
}

func TestMesaRecusaRespostaForaDaJanela(t *testing.T) {
	meta := novaMetaFalsa(t)
	h, gdb := servidorComMesa(t, meta)
	criarGerencia(t, h)

	id := conversaAberta(t, h, "5531988886666", "wamid.CLIENTE2")

	// Envelhece a janela: é o que o tempo faria passadas 24 horas.
	if err := gdb.Exec(
		`UPDATE conversas SET janela_expira_em = now() - interval '1 minute' WHERE id = ?::uuid`,
		id).Error; err != nil {
		t.Fatalf("envelhecer a janela: %v", err)
	}

	cookie := logar(t, h, emailGerencia, senhaGerencia)
	rec := chamar(t, h, http.MethodPost, "/api/v1/conversas/"+id+"/mensagens",
		map[string]string{"texto": "ainda dá pra negociar?"}, cookie)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, quer 409 (%s)", rec.Code, rec.Body.String())
	}

	var p struct{ Codigo string }
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("ler problema: %v", err)
	}
	// O código próprio é o que deixa a tela explicar que o caminho agora é a
	// régua, em vez de mostrar "conflito" genérico.
	if p.Codigo != "janela_fechada" {
		t.Errorf("codigo = %q, quer \"janela_fechada\"", p.Codigo)
	}

	// E, principalmente: nada foi enviado. Recusar depois de mandar seria
	// cobrar a tarifa e mentir para o analista.
	if enviados := meta.enviados(); len(enviados) != 0 {
		t.Errorf("envios para a Meta = %d, quer 0", len(enviados))
	}
}

func TestMesaSemCanalConfiguradoFicaEmModoLeitura(t *testing.T) {
	// Sem WHATSAPP_BASE_URL apontado e sem credencial, é o estado do projeto
	// hoje: dá para ler o que o cliente escreveu, não dá para responder.
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		Env: "development", Port: "3000", AppURL: "https://facilita.arcom.com.br",
		WhatsAppAppSecret:   appSecretDeTeste,
		WhatsAppVerifyToken: verifyTokenDeTeste,
	}
	gdb := bancoDeTeste(t)
	h := servidor.Novo(cfg, log, gdb, nil)
	criarGerencia(t, h)

	id := conversaAberta(t, h, "5531988885555", "wamid.CLIENTE3")
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	rec := chamar(t, h, http.MethodPost, "/api/v1/conversas/"+id+"/mensagens",
		map[string]string{"texto": "oi"}, cookie)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, quer 409 (%s)", rec.Code, rec.Body.String())
	}

	var p struct{ Codigo string }
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("ler problema: %v", err)
	}
	if p.Codigo != "sem_canal" {
		t.Errorf("codigo = %q, quer \"sem_canal\"", p.Codigo)
	}
}

func TestMesaRecusaTextoVazioOuLongoDemais(t *testing.T) {
	meta := novaMetaFalsa(t)
	h, _ := servidorComMesa(t, meta)
	criarGerencia(t, h)

	id := conversaAberta(t, h, "5531988884444", "wamid.CLIENTE4")
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	longo := make([]byte, 4097)
	for i := range longo {
		longo[i] = 'a'
	}

	casos := map[string]string{
		"vazio":         "",
		"só espaço":     "   ",
		"acima do teto": string(longo),
	}
	for nome, texto := range casos {
		t.Run(nome, func(t *testing.T) {
			rec := chamar(t, h, http.MethodPost, "/api/v1/conversas/"+id+"/mensagens",
				map[string]string{"texto": texto}, cookie)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, quer 422 (%s)", rec.Code, rec.Body.String())
			}
		})
	}

	if enviados := meta.enviados(); len(enviados) != 0 {
		t.Errorf("envios para a Meta = %d, quer 0 — validação não gasta chamada", len(enviados))
	}
}

func TestMesaNaoAceitaAutorNoCorpo(t *testing.T) {
	// Campo desconhecido é recusado: sem isso, um corpo com "usuarioId" abriria
	// a porta para assinar mensagem no nome de outro analista.
	meta := novaMetaFalsa(t)
	h, _ := servidorComMesa(t, meta)
	criarGerencia(t, h)

	id := conversaAberta(t, h, "5531988883333", "wamid.CLIENTE5")
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	rec := chamar(t, h, http.MethodPost, "/api/v1/conversas/"+id+"/mensagens",
		map[string]any{"texto": "oi", "usuarioId": "00000000-0000-0000-0000-000000000001"}, cookie)
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, quer 400 ou 422 (%s)", rec.Code, rec.Body.String())
	}
	if enviados := meta.enviados(); len(enviados) != 0 {
		t.Errorf("envios = %d, quer 0", len(enviados))
	}
}

func TestMesaExigeSessao(t *testing.T) {
	meta := novaMetaFalsa(t)
	h, _ := servidorComMesa(t, meta)
	criarGerencia(t, h)

	id := conversaAberta(t, h, "5531988882222", "wamid.CLIENTE6")

	rec := chamar(t, h, http.MethodPost, "/api/v1/conversas/"+id+"/mensagens",
		map[string]string{"texto": "oi"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, quer 401 (%s)", rec.Code, rec.Body.String())
	}
}

func TestMesaResponde404ParaConversaInexistente(t *testing.T) {
	meta := novaMetaFalsa(t)
	h, _ := servidorComMesa(t, meta)
	criarGerencia(t, h)
	cookie := logar(t, h, emailGerencia, senhaGerencia)

	rec := chamar(t, h, http.MethodPost,
		"/api/v1/conversas/00000000-0000-0000-0000-0000000000ff/mensagens",
		map[string]string{"texto": "oi"}, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, quer 404 (%s)", rec.Code, rec.Body.String())
	}
}
