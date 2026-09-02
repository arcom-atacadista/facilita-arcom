// Package disparo é a régua de cobrança: monta a mensagem de cada faixa de
// atraso, enfileira e entrega por um Canal (ver canal.go).
package disparo

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	StatusNaFila    = "na_fila"
	StatusEnviado   = "enviado"
	StatusErro      = "erro"
	StatusCancelado = "cancelado"
)

type Disparo struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey"`
	DividaID          uuid.UUID  `gorm:"type:uuid;column:divida_id"`
	CampanhaID        *uuid.UUID `gorm:"type:uuid;column:campanha_id"`
	Telefone          string
	Mensagem          string
	Canal             string
	Status            string
	Tentativas        int
	ErroDetalhe       *string    `gorm:"column:erro_detalhe"`
	ReferenciaExterna *string    `gorm:"column:referencia_externa"`
	AgendadoPara      time.Time  `gorm:"column:agendado_para"`
	EnviadoEm         *time.Time `gorm:"column:enviado_em"`
	CriadoPor         *uuid.UUID `gorm:"type:uuid;column:criado_por"`
	CriadoEm          time.Time  `gorm:"column:criado_em"`
}

func (Disparo) TableName() string { return "disparos" }

type DisparoResposta struct {
	ID           uuid.UUID  `json:"id"`
	DividaID     uuid.UUID  `json:"dividaId"`
	Telefone     string     `json:"telefone"`
	Canal        string     `json:"canal"`
	Status       string     `json:"status"`
	Tentativas   int        `json:"tentativas"`
	ErroDetalhe  *string    `json:"erroDetalhe"`
	AgendadoPara time.Time  `json:"agendadoPara"`
	EnviadoEm    *time.Time `json:"enviadoEm"`
	CriadoEm     time.Time  `json:"criadoEm"`
}

// Responder devolve o telefone mascarado: a tela de histórico não precisa do
// número completo, e a lista é lida por operador de qualquer papel.
func Responder(d Disparo) DisparoResposta {
	return DisparoResposta{
		ID: d.ID, DividaID: d.DividaID,
		Telefone: mascararTelefone(d.Telefone), Canal: d.Canal,
		Status: d.Status, Tentativas: d.Tentativas, ErroDetalhe: d.ErroDetalhe,
		AgendadoPara: d.AgendadoPara, EnviadoEm: d.EnviadoEm, CriadoEm: d.CriadoEm,
	}
}

// EntradaEnfileirar pede o disparo de uma dívida da carteira de quem chama.
type EntradaEnfileirar struct {
	DividaID uuid.UUID `json:"dividaId" validate:"required"`
}

// --- montagem da mensagem ---

var naoDigito = regexp.MustCompile(`\D`)

// TelefoneComDDI normaliza para o formato que um provedor de WhatsApp espera:
// só dígitos, com o 55 na frente. Número que já vem com DDI é preservado.
func TelefoneComDDI(telefone string) string {
	so := naoDigito.ReplaceAllString(telefone, "")
	switch {
	case so == "":
		return ""
	// 10 ou 11 dígitos = número nacional (DDD + assinante), falta o DDI.
	case len(so) <= 11:
		return "55" + so
	default:
		return so
	}
}

// TelefoneValido recusa o que não tem cara de celular brasileiro. Mandar
// mensagem para número quebrado gasta cota do provedor e some no silêncio.
func TelefoneValido(comDDI string) bool {
	// 55 + DDD (2) + assinante (8 ou 9).
	return len(comDDI) == 12 || len(comDDI) == 13
}

// Variaveis são os valores que o template da campanha aceita.
type Variaveis struct {
	Nome     string
	Contrato string
	Dias     int
	Valor    float64
	Link     string
}

// MontarMensagem troca os marcadores do template. Só os marcadores conhecidos
// são substituídos; qualquer outro texto do template passa intacto — o
// template vem do banco, escrito pela coordenação, e não é interpretado como
// código.
func MontarMensagem(template string, v Variaveis) string {
	substituicoes := []string{
		"{nome}", primeiroNome(v.Nome),
		"{nome_completo}", v.Nome,
		"{contrato}", v.Contrato,
		"{dias}", fmt.Sprintf("%d", v.Dias),
		"{valor}", Moeda(v.Valor),
		"{link}", v.Link,
	}
	return strings.NewReplacer(substituicoes...).Replace(template)
}

func primeiroNome(nome string) string {
	if partes := strings.Fields(nome); len(partes) > 0 {
		return partes[0]
	}
	return nome
}

// Moeda formata em real brasileiro. Feito à mão porque a stdlib do Go não tem
// formatação por locale, e a alternativa (golang.org/x/text/message) não está
// no allowlist.
func Moeda(v float64) string {
	// math.Round direto sobre v*100, e não int64(Centavos(v)*100): arredondar
	// para duas casas e só depois multiplicar volta a cair no resto binário
	// (1.234.567,89 * 100 = 123456788,999... em float64), e a conversão para
	// inteiro truncaria um centavo a menos do que a tela mostra.
	centavos := int64(math.Round(v * 100))
	negativo := centavos < 0
	if negativo {
		centavos = -centavos
	}

	inteiros := fmt.Sprintf("%d", centavos/100)
	var comPontos strings.Builder
	for i, d := range inteiros {
		if i > 0 && (len(inteiros)-i)%3 == 0 {
			comPontos.WriteByte('.')
		}
		comPontos.WriteRune(d)
	}

	sinal := ""
	if negativo {
		sinal = "-"
	}
	return fmt.Sprintf("%sR$ %s,%02d", sinal, comPontos.String(), centavos%100)
}
