package disparo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// canalMeta entrega a mensagem pela Cloud API do WhatsApp, direto na Meta —
// sem intermediário, que é o caminho mais barato e o que a plataforma já
// comporta (ver canal.go).
//
// O que a Meta aceita fora de uma janela de atendimento aberta é apenas um
// template previamente aprovado, referenciado pelo nome, com os valores em
// ordem para os marcadores {{1}}, {{2}}, ... O texto que montamos serve de
// registro; o que viaja é o nome do template mais os parâmetros.
type canalMeta struct {
	http      *http.Client
	baseURL   string
	versaoAPI string
	idNumero  string
	token     string
}

// URLBaseMeta é fixa. Nunca montada com input de cliente — é o que fecha SSRF
// nesta superfície (ver skill seguranca).
const URLBaseMeta = "https://graph.facebook.com"

// ConfigMeta são as credenciais que o time de TI provisiona junto com a conta
// WhatsApp Business da ARCOM.
type ConfigMeta struct {
	// IDNumero é o phone number id da conta (não é o telefone em si).
	IDNumero string
	// Token é o access token permanente da aplicação.
	Token string
	// VersaoAPI no formato vNN.N. Vazio usa o padrão abaixo.
	VersaoAPI string
	// BaseURL só é preenchida em teste, para apontar a um servidor falso.
	BaseURL string
}

const versaoPadraoMeta = "v25.0"

// NovoCanalMeta devolve o canal pronto para uso. Credencial faltando é erro
// de configuração e aparece no boot, não no meio de um disparo.
func NovoCanalMeta(cfg ConfigMeta) (Canal, error) {
	if cfg.IDNumero == "" || cfg.Token == "" {
		return nil, errors.New("canal Meta exige WHATSAPP_PHONE_NUMBER_ID e WHATSAPP_ACCESS_TOKEN")
	}

	versao := cfg.VersaoAPI
	if versao == "" {
		versao = versaoPadraoMeta
	}
	base := cfg.BaseURL
	if base == "" {
		base = URLBaseMeta
	}

	return &canalMeta{
		// Timeout sempre presente: sem ele uma chamada pendurada segura a
		// rodada inteira do worker.
		http:      &http.Client{Timeout: 20 * time.Second},
		baseURL:   strings.TrimRight(base, "/"),
		versaoAPI: versao,
		idNumero:  cfg.IDNumero,
		token:     cfg.Token,
	}, nil
}

func (c *canalMeta) Nome() string { return "whatsapp" }

// --- corpo da requisição ---

type corpoEnvio struct {
	MessagingProduct string        `json:"messaging_product"`
	RecipientType    string        `json:"recipient_type"`
	To               string        `json:"to"`
	Type             string        `json:"type"`
	Template         corpoTemplate `json:"template"`
}

type corpoTemplate struct {
	Name       string            `json:"name"`
	Language   corpoIdioma       `json:"language"`
	Components []corpoComponente `json:"components,omitempty"`
}

type corpoIdioma struct {
	Code string `json:"code"`
}

type corpoComponente struct {
	Type       string           `json:"type"`
	Parameters []corpoParametro `json:"parameters"`
}

type corpoParametro struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type respostaEnvio struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

type respostaErro struct {
	Erro struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code"`
		Subcode int    `json:"error_subcode"`
		Detalhe string `json:"error_data.details"`
	} `json:"error"`
}

// ErroMeta é uma recusa vinda da Meta. Permanente separa o que adianta tentar
// de novo (limite de taxa, instabilidade) do que só se resolve com
// intervenção humana (template não aprovado, número inválido, token expirado)
// — tentar 3x um template reprovado só gasta tempo e polui o log.
type ErroMeta struct {
	Status     int
	Codigo     int
	Subcodigo  int
	Mensagem   string
	Permanente bool
}

func (e *ErroMeta) Error() string {
	return fmt.Sprintf("meta recusou o envio (http %d, código %d)", e.Status, e.Codigo)
}

// codigosPermanentes são recusas que nenhuma tentativa futura resolve.
//
//	131026 — número não existe no WhatsApp ou não pode receber
//	132000 a 132015 — problemas do template (não existe, não aprovado,
//	                  contagem de parâmetros errada, idioma indisponível)
//	131047 — fora da janela e sem template válido
//	190    — token inválido ou expirado
func permanente(status, codigo int) bool {
	switch {
	case codigo == 190, codigo == 131026, codigo == 131047, codigo == 131051:
		return true
	case codigo >= 132000 && codigo <= 132015:
		return true
	// 4xx que não seja limite de taxa é problema da requisição, não do momento.
	case status >= 400 && status < 500 && status != http.StatusTooManyRequests:
		return true
	}
	return false
}

// A Meta recusa parâmetro de template com quebra de linha, tabulação ou mais
// de quatro espaços seguidos. Nossos valores não deveriam ter nada disso, mas
// um nome cadastrado com espaço duplo no ERP chegaria aqui — e a recusa viria
// como erro genérico, difícil de rastrear.
var espacosDemais = regexp.MustCompile(`\s{2,}`)

func limparParametro(v string) string {
	v = strings.ReplaceAll(v, "\n", " ")
	v = strings.ReplaceAll(v, "\t", " ")
	return strings.TrimSpace(espacosDemais.ReplaceAllString(v, " "))
}

func (c *canalMeta) Enviar(ctx context.Context, m Mensagem) (string, error) {
	if m.Template == "" {
		return "", errors.New("disparo sem template aprovado — a Meta não aceita texto livre fora da janela de atendimento")
	}

	idioma := m.Idioma
	if idioma == "" {
		idioma = "pt_BR"
	}

	parametros := make([]corpoParametro, 0, len(m.Parametros))
	for _, valor := range m.Parametros {
		parametros = append(parametros, corpoParametro{Type: "text", Text: limparParametro(valor)})
	}

	corpo := corpoEnvio{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               m.Telefone,
		Type:             "template",
		Template: corpoTemplate{
			Name:     m.Template,
			Language: corpoIdioma{Code: idioma},
		},
	}
	if len(parametros) > 0 {
		corpo.Template.Components = []corpoComponente{{Type: "body", Parameters: parametros}}
	}

	bruto, err := json.Marshal(corpo)
	if err != nil {
		return "", fmt.Errorf("serializar envio: %w", err)
	}

	url := fmt.Sprintf("%s/%s/%s/messages", c.baseURL, c.versaoAPI, c.idNumero)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bruto))
	if err != nil {
		return "", fmt.Errorf("montar requisição de envio: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		// A URL não entra na mensagem: ela vai para o log e carrega o id do
		// número da conta.
		return "", fmt.Errorf("chamar a Meta: %w", err)
	}
	defer resp.Body.Close()

	corpoResp, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("ler resposta da Meta: %w", err)
	}

	if resp.StatusCode >= 400 {
		var falha respostaErro
		_ = json.Unmarshal(corpoResp, &falha)
		return "", &ErroMeta{
			Status:     resp.StatusCode,
			Codigo:     falha.Erro.Code,
			Subcodigo:  falha.Erro.Subcode,
			Mensagem:   falha.Erro.Message,
			Permanente: permanente(resp.StatusCode, falha.Erro.Code),
		}
	}

	var ok respostaEnvio
	if err := json.Unmarshal(corpoResp, &ok); err != nil {
		return "", fmt.Errorf("resposta da Meta em formato inesperado: %w", err)
	}
	if len(ok.Messages) == 0 || ok.Messages[0].ID == "" {
		// Sem o wamid não dá para conciliar a entrega depois; melhor falhar
		// alto do que gravar o disparo como enviado sem rastro.
		return "", errors.New("a Meta aceitou o envio mas não devolveu o id da mensagem")
	}
	return ok.Messages[0].ID, nil
}
