// Package cobranca é o núcleo do negócio: a carteira em atraso, a política
// comercial de cada faixa e os acordos fechados sobre ela.
//
// Cliente, dívida, política, acordo e parcela vivem no mesmo pacote porque
// são um agregado só — não existe acordo sem dívida, nem parcela sem acordo,
// e a regra de desconto lê a política junto da dívida. Separá-los em quatro
// pacotes só produziria import cruzado entre eles.
package cobranca

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// --- entidades ---

type Cliente struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Nome         string
	Documento    string
	Telefone     *string
	Email        *string
	CriadoEm     time.Time `gorm:"column:criado_em"`
	AtualizadoEm time.Time `gorm:"column:atualizado_em"`
}

func (Cliente) TableName() string { return "clientes" }

// LogValue mascara os dados pessoais: nome, CPF/CNPJ e telefone de devedor
// não vão pro log (03-backend.md, e é dado pessoal sob a LGPD).
func (c Cliente) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", c.ID.String()),
		slog.String("documento", mascararDocumento(c.Documento)),
	)
}

// mascararDocumento deixa só os dois últimos dígitos, o suficiente pra
// conferir um registro no suporte sem despejar o CPF no log.
func mascararDocumento(doc string) string {
	if len(doc) <= 2 {
		return "***"
	}
	return "***" + doc[len(doc)-2:]
}

const (
	StatusDividaAberta    = "aberto"
	StatusDividaNegociada = "negociado"
	StatusDividaQuitada   = "quitado"
	StatusDividaCancelada = "cancelado"
)

type Divida struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	ClienteID     uuid.UUID `gorm:"type:uuid;column:cliente_id"`
	Contrato      string
	ValorOriginal float64   `gorm:"column:valor_original"`
	Vencimento    time.Time `gorm:"type:date"`
	Status        string

	ResponsavelCobranca *string `gorm:"column:responsavel_cobranca"`
	Filial              *string

	TokenHash     []byte     `gorm:"column:token_hash"`
	TokenExpiraEm *time.Time `gorm:"column:token_expira_em"`

	CriadoEm     time.Time `gorm:"column:criado_em"`
	AtualizadoEm time.Time `gorm:"column:atualizado_em"`

	Cliente *Cliente `gorm:"foreignKey:ClienteID"`
}

func (Divida) TableName() string { return "dividas" }

type Politica struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	Nome              string
	FaixaMin          int       `gorm:"column:faixa_min"`
	FaixaMax          int       `gorm:"column:faixa_max"`
	DescontoAvista    float64   `gorm:"column:desconto_avista"`
	DescontoParcelado float64   `gorm:"column:desconto_parcelado"`
	MaxParcelas       int       `gorm:"column:max_parcelas"`
	EntradaMinimaPct  float64   `gorm:"column:entrada_minima_pct"`
	CriadoEm          time.Time `gorm:"column:criado_em"`
	AtualizadoEm      time.Time `gorm:"column:atualizado_em"`
}

func (Politica) TableName() string { return "politicas" }

type Campanha struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Nome         string
	FaixaMin     int `gorm:"column:faixa_min"`
	FaixaMax     int `gorm:"column:faixa_max"`
	Canal        string
	Template     string
	Ativo        bool
	CriadoEm     time.Time `gorm:"column:criado_em"`
	AtualizadoEm time.Time `gorm:"column:atualizado_em"`
}

func (Campanha) TableName() string { return "campanhas" }

const (
	StatusAcordoAtivo     = "ativo"
	StatusAcordoQuitado   = "quitado"
	StatusAcordoRompido   = "rompido"
	StatusAcordoCancelado = "cancelado"

	OrigemCliente  = "cliente"
	OrigemOperador = "operador"

	PagamentoPix    = "pix"
	PagamentoBoleto = "boleto"
)

type Acordo struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	DividaID      uuid.UUID `gorm:"type:uuid;column:divida_id"`
	TipoPagamento string    `gorm:"column:tipo_pagamento"`
	DescontoPct   float64   `gorm:"column:desconto_pct"`
	Entrada       float64
	Parcelas      int
	ValorTotal    float64 `gorm:"column:valor_total"`
	Status        string
	Origem        string
	CriadoPor     *uuid.UUID `gorm:"type:uuid;column:criado_por"`
	AprovadoPor   *uuid.UUID `gorm:"type:uuid;column:aprovado_por"`
	CriadoEm      time.Time  `gorm:"column:criado_em"`
	AtualizadoEm  time.Time  `gorm:"column:atualizado_em"`

	Lista []Parcela `gorm:"foreignKey:AcordoID"`
}

func (Acordo) TableName() string { return "acordos" }

type Parcela struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	AcordoID   uuid.UUID `gorm:"type:uuid;column:acordo_id"`
	Numero     int
	Valor      float64
	Vencimento time.Time `gorm:"type:date"`
	Pago       bool
	PagoEm     *time.Time `gorm:"column:pago_em"`
	CriadoEm   time.Time  `gorm:"column:criado_em"`
}

func (Parcela) TableName() string { return "parcelas" }

// --- DTOs de saída ---

type DividaResposta struct {
	ID                  uuid.UUID       `json:"id"`
	Contrato            string          `json:"contrato"`
	ValorOriginal       float64         `json:"valorOriginal"`
	Vencimento          string          `json:"vencimento"`
	DiasAtraso          int             `json:"diasAtraso"`
	Faixa               string          `json:"faixa"`
	Status              string          `json:"status"`
	Filial              *string         `json:"filial"`
	ResponsavelCobranca *string         `json:"responsavelCobranca"`
	Cliente             ClienteResposta `json:"cliente"`
	TemLinkAtivo        bool            `json:"temLinkAtivo"`
}

type ClienteResposta struct {
	ID        uuid.UUID `json:"id"`
	Nome      string    `json:"nome"`
	Documento string    `json:"documento"`
	Telefone  *string   `json:"telefone"`
	Email     *string   `json:"email"`
}

type PoliticaResposta struct {
	ID                uuid.UUID `json:"id"`
	Nome              string    `json:"nome"`
	FaixaMin          int       `json:"faixaMin"`
	FaixaMax          int       `json:"faixaMax"`
	DescontoAvista    float64   `json:"descontoAvista"`
	DescontoParcelado float64   `json:"descontoParcelado"`
	MaxParcelas       int       `json:"maxParcelas"`
	EntradaMinimaPct  float64   `json:"entradaMinimaPct"`
}

type AcordoResposta struct {
	ID            uuid.UUID         `json:"id"`
	DividaID      uuid.UUID         `json:"dividaId"`
	TipoPagamento string            `json:"tipoPagamento"`
	DescontoPct   float64           `json:"descontoPct"`
	Entrada       float64           `json:"entrada"`
	Parcelas      int               `json:"parcelas"`
	ValorTotal    float64           `json:"valorTotal"`
	Status        string            `json:"status"`
	Origem        string            `json:"origem"`
	CriadoEm      time.Time         `json:"criadoEm"`
	Lista         []ParcelaResposta `json:"lista"`
}

type ParcelaResposta struct {
	Numero     int     `json:"numero"`
	Valor      float64 `json:"valor"`
	Vencimento string  `json:"vencimento"`
	Pago       bool    `json:"pago"`
}

// --- DTOs de entrada ---

type EntradaAtualizarPolitica struct {
	Nome              *string  `json:"nome" validate:"omitempty,min=2,max=120"`
	DescontoAvista    *float64 `json:"descontoAvista" validate:"omitempty,gte=0,lte=100"`
	DescontoParcelado *float64 `json:"descontoParcelado" validate:"omitempty,gte=0,lte=100"`
	MaxParcelas       *int     `json:"maxParcelas" validate:"omitempty,gte=1,lte=24"`
	EntradaMinimaPct  *float64 `json:"entradaMinimaPct" validate:"omitempty,gte=0,lte=100"`
}

// EntradaFecharAcordo é o acordo lançado por um operador na mesa (o cliente
// que fecha sozinho passa pelo pacote negociacao, que não aceita desconto
// arbitrário — usa só o da política).
type EntradaFecharAcordo struct {
	DividaID      uuid.UUID `json:"dividaId" validate:"required"`
	TipoPagamento string    `json:"tipoPagamento" validate:"required,oneof=pix boleto"`
	DescontoPct   float64   `json:"descontoPct" validate:"gte=0,lte=100"`
	Parcelas      int       `json:"parcelas" validate:"required,gte=1,lte=24"`
}

func RespostaDeCliente(c Cliente) ClienteResposta {
	return ClienteResposta{ID: c.ID, Nome: c.Nome, Documento: c.Documento, Telefone: c.Telefone, Email: c.Email}
}

func RespostaDePolitica(p Politica) PoliticaResposta {
	return PoliticaResposta{
		ID: p.ID, Nome: p.Nome, FaixaMin: p.FaixaMin, FaixaMax: p.FaixaMax,
		DescontoAvista: p.DescontoAvista, DescontoParcelado: p.DescontoParcelado,
		MaxParcelas: p.MaxParcelas, EntradaMinimaPct: p.EntradaMinimaPct,
	}
}

func RespostaDeAcordo(a Acordo) AcordoResposta {
	lista := make([]ParcelaResposta, 0, len(a.Lista))
	for _, p := range a.Lista {
		lista = append(lista, ParcelaResposta{
			Numero: p.Numero, Valor: p.Valor,
			Vencimento: p.Vencimento.Format(FormatoData), Pago: p.Pago,
		})
	}
	return AcordoResposta{
		ID: a.ID, DividaID: a.DividaID, TipoPagamento: a.TipoPagamento,
		DescontoPct: a.DescontoPct, Entrada: a.Entrada, Parcelas: a.Parcelas,
		ValorTotal: a.ValorTotal, Status: a.Status, Origem: a.Origem,
		CriadoEm: a.CriadoEm, Lista: lista,
	}
}

func RespostaDeDivida(d Divida, agora time.Time) DividaResposta {
	dias := DiasAtraso(d.Vencimento, agora)
	var cliente ClienteResposta
	if d.Cliente != nil {
		cliente = RespostaDeCliente(*d.Cliente)
	}
	return DividaResposta{
		ID: d.ID, Contrato: d.Contrato, ValorOriginal: d.ValorOriginal,
		Vencimento: d.Vencimento.Format(FormatoData),
		DiasAtraso: dias, Faixa: string(FaixaDe(dias)), Status: d.Status,
		Filial: d.Filial, ResponsavelCobranca: d.ResponsavelCobranca,
		Cliente:      cliente,
		TemLinkAtivo: d.TokenExpiraEm != nil && d.TokenExpiraEm.After(agora),
	}
}
