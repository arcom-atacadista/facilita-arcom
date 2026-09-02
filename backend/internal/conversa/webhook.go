package conversa

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"facilitaarcom/internal/problema"
)

// CaminhoWebhook é a rota que a Meta chama. Fica sob /api/ de propósito: o
// nginx já encaminha tudo sob esse prefixo para o backend, então o webhook
// funciona sem tocar na configuração de infraestrutura.
const CaminhoWebhook = "/webhooks/whatsapp"

// tamanhoMaximoEvento é o teto de leitura do corpo. A Meta agrupa eventos,
// mas nada perto disso; o limite existe para um corpo inesperado não comer a
// memória do backend.
const tamanhoMaximoEvento = 2 << 20 // 2 MB

type Webhook struct {
	svc         *Service
	appSecret   []byte
	verifyToken string
}

// NovoWebhook monta o handler. appSecret vazio desliga a rota inteira: um
// webhook público sem verificação de assinatura aceitaria evento forjado por
// qualquer um, e "mensagem do cliente" forjada é coisa séria numa operação de
// cobrança.
func NovoWebhook(svc *Service, appSecret, verifyToken string) *Webhook {
	return &Webhook{svc: svc, appSecret: []byte(appSecret), verifyToken: verifyToken}
}

func (w *Webhook) Configurado() bool { return len(w.appSecret) > 0 && w.verifyToken != "" }

func (h *Webhook) Rotas(r chi.Router) {
	r.Get(CaminhoWebhook, h.verificar)
	r.Post(CaminhoWebhook, h.receber)
}

// verificar responde o handshake que a Meta faz ao cadastrar o webhook: ela
// chama com um token combinado e espera o desafio de volta em texto puro.
func (h *Webhook) verificar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if q.Get("hub.mode") != "subscribe" ||
		subtle.ConstantTimeCompare([]byte(q.Get("hub.verify_token")), []byte(h.verifyToken)) != 1 {
		// Sem detalhe do que não bateu: quem chama sem o token combinado não
		// precisa saber se errou o modo ou o segredo.
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(q.Get("hub.challenge")))
}

func (h *Webhook) receber(w http.ResponseWriter, r *http.Request) {
	// O corpo cru é lido antes de qualquer coisa: a assinatura é sobre os
	// bytes exatos que a Meta mandou, e reserializar o JSON quebraria a
	// conferência.
	bruto, err := io.ReadAll(io.LimitReader(r.Body, tamanhoMaximoEvento))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !h.assinaturaConfere(r.Header.Get("X-Hub-Signature-256"), bruto) {
		h.svc.log.WarnContext(r.Context(), "webhook com assinatura inválida foi recusado",
			"tamanho_corpo", len(bruto))
		_ = problema.EscreverErro(w, problema.NaoAutenticado("Assinatura inválida."))
		return
	}

	var ev Evento
	if err := json.Unmarshal(bruto, &ev); err != nil {
		// Corpo que não é o evento esperado: registramos e devolvemos 200.
		// Devolver erro faria a Meta reentregar para sempre o mesmo payload
		// indigesto.
		h.svc.log.WarnContext(r.Context(), "webhook em formato inesperado", "erro", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.svc.Processar(r.Context(), ev); err != nil {
		// Falha nossa (banco fora do ar, por exemplo): 500 faz a Meta
		// reentregar, que é exatamente o que queremos aqui.
		h.svc.log.ErrorContext(r.Context(), "falha ao processar webhook", "erro", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// assinaturaConfere valida o X-Hub-Signature-256: HMAC-SHA256 do corpo cru
// com o segredo da aplicação. Comparação em tempo constante, para a diferença
// de tempo entre duas assinaturas erradas não entregar nada.
func (h *Webhook) assinaturaConfere(cabecalho string, corpo []byte) bool {
	if len(h.appSecret) == 0 {
		return false
	}

	assinatura, achou := strings.CutPrefix(cabecalho, "sha256=")
	if !achou {
		return false
	}

	recebida, err := hex.DecodeString(assinatura)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, h.appSecret)
	mac.Write(corpo)
	return hmac.Equal(recebida, mac.Sum(nil))
}
