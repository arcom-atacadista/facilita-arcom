// Package conversa é a entrada de mensagem: o que o cliente responde ao
// WhatsApp da ARCOM, e o que a Meta informa sobre a entrega do que saiu.
//
// É a base da mesa de atendimento e o que controla a janela de 24 horas —
// dentro dela a resposta ao cliente é texto livre e não é tarifada; fora
// dela, só template pago.
package conversa

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const (
	DirecaoEntrada = "entrada"
	DirecaoSaida   = "saida"

	StatusEnviada  = "enviada"
	StatusEntregue = "entregue"
	StatusLida     = "lida"
	StatusFalhou   = "falhou"
)

type Conversa struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	Telefone   string
	ClienteID  *uuid.UUID `gorm:"type:uuid;column:cliente_id"`
	NomePerfil *string    `gorm:"column:nome_perfil"`

	// JanelaExpiraEm é o fim da janela de atendimento, como a Meta informa.
	JanelaExpiraEm   *time.Time `gorm:"column:janela_expira_em"`
	UltimaMensagemEm *time.Time `gorm:"column:ultima_mensagem_em"`
	NaoLidas         int        `gorm:"column:nao_lidas"`

	CriadoEm     time.Time `gorm:"column:criado_em"`
	AtualizadoEm time.Time `gorm:"column:atualizado_em"`
}

func (Conversa) TableName() string { return "conversas" }

// LogValue mascara o telefone: é dado pessoal e não vai cru para o log.
func (c Conversa) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", c.ID.String()),
		slog.String("telefone", MascararTelefone(c.Telefone)),
	)
}

// JanelaAberta diz se dá para responder com texto livre, sem custo.
func (c Conversa) JanelaAberta(agora time.Time) bool {
	return c.JanelaExpiraEm != nil && c.JanelaExpiraEm.After(agora)
}

type Mensagem struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	ConversaID uuid.UUID `gorm:"type:uuid;column:conversa_id"`
	Direcao    string
	Wamid      *string
	Tipo       string
	Texto      *string
	Status     *string
	ErroCodigo *int      `gorm:"column:erro_codigo"`
	OcorridaEm time.Time `gorm:"column:ocorrida_em"`
	CriadoEm   time.Time `gorm:"column:criado_em"`
}

func (Mensagem) TableName() string { return "mensagens" }

func MascararTelefone(t string) string {
	if len(t) <= 4 {
		return "****"
	}
	return "****" + t[len(t)-4:]
}

// --- o que a Meta manda no webhook ---
//
// Só os campos que a plataforma usa. O payload da Meta é bem maior, e
// repassar tudo seria guardar dado que ninguém pediu.

type Evento struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
}

type Change struct {
	Field string `json:"field"`
	Value Value  `json:"value"`
}

type Value struct {
	MessagingProduct string       `json:"messaging_product"`
	Metadata         Metadata     `json:"metadata"`
	Contacts         []Contato    `json:"contacts"`
	Messages         []MsgMeta    `json:"messages"`
	Statuses         []StatusMeta `json:"statuses"`
}

type Metadata struct {
	PhoneNumberID string `json:"phone_number_id"`
}

type Contato struct {
	WaID    string `json:"wa_id"`
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
}

type MsgMeta struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      struct {
		Body string `json:"body"`
	} `json:"text"`
	Button struct {
		Text string `json:"text"`
	} `json:"button"`
}

type StatusMeta struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp"`
	RecipientID  string `json:"recipient_id"`
	Conversation struct {
		ID                  string `json:"id"`
		ExpirationTimestamp string `json:"expiration_timestamp"`
	} `json:"conversation"`
	Errors []struct {
		Code    int    `json:"code"`
		Title   string `json:"title"`
		Message string `json:"message"`
	} `json:"errors"`
}
