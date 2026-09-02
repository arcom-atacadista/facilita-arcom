// Package acesso é a identidade do Facilita ARCOM: quem entra, com que papel
// e até que desconto pode conceder sozinho. Substitui o Supabase Auth + as
// tabelas profiles/user_roles do modelo anterior.
package acesso

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Papel é o nível do usuário na operação de cobrança. A ordem importa:
// Nivel() usa a posição para comparar "pelo menos coordenação".
type Papel string

const (
	PapelAprendiz    Papel = "aprendiz"
	PapelAnalista    Papel = "analista"
	PapelCoordenacao Papel = "coordenacao"
	PapelGerencia    Papel = "gerencia"
)

// Equipe separa quem atende a carteira interna de quem atende as filiais.
type Equipe string

const (
	EquipeInterno Equipe = "interno"
	EquipeFiliais Equipe = "filiais"
)

var niveis = map[Papel]int{
	PapelAprendiz:    1,
	PapelAnalista:    2,
	PapelCoordenacao: 3,
	PapelGerencia:    4,
}

// AlcadaPadrao é o teto de desconto que cada papel recebe ao ser criado, em
// pontos percentuais. Depois disso a alçada vive na coluna do usuário e pode
// ser ajustada caso a caso pela coordenação.
var AlcadaPadrao = map[Papel]float64{
	PapelAprendiz:    20,
	PapelAnalista:    60,
	PapelCoordenacao: 100,
	PapelGerencia:    100,
}

// Valido diz se a string veio de fora com um papel que existe — usado antes
// de gravar, porque o ENUM do banco recusaria com erro 500 em vez de 422.
func (p Papel) Valido() bool { _, ok := niveis[p]; return ok }

func (p Papel) Nivel() int { return niveis[p] }

// PeloMenos compara hierarquia: gerência satisfaz "pelo menos coordenação".
func (p Papel) PeloMenos(minimo Papel) bool { return p.Nivel() >= minimo.Nivel() }

// VeCarteiraInteira separa quem enxerga a base toda de devedores de quem só
// enxerga a própria carteira. É a regra que fecha o buraco de LGPD do modelo
// antigo, onde qualquer autenticado lia todos os clientes.
func (p Papel) VeCarteiraInteira() bool { return p.PeloMenos(PapelCoordenacao) }

func (e Equipe) Valido() bool { return e == EquipeInterno || e == EquipeFiliais }

// Usuario é a conta de quem opera a cobrança.
type Usuario struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Nome         string
	Email        string
	SenhaHash    string `gorm:"column:senha_hash"`
	Papel        Papel  `gorm:"type:papel_usuario"`
	Equipe       Equipe `gorm:"type:equipe_usuario"`
	Ativo        bool
	AlcadaMaxima float64 `gorm:"column:alcada_maxima"`
	// Casa com dividas.responsavel_cobranca. Vazio = sem carteira própria;
	// quem não vê a carteira inteira e não tem código não vê dívida nenhuma.
	CodigoCobranca *string   `gorm:"column:codigo_cobranca"`
	CriadoEm       time.Time `gorm:"column:criado_em"`
	AtualizadoEm   time.Time `gorm:"column:atualizado_em"`
}

func (Usuario) TableName() string { return "usuarios" }

// LogValue evita que hash de senha e e-mail caiam no log quando alguém logar
// o usuário inteiro por engano — exigência de 03-backend.md.
func (u Usuario) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", u.ID.String()),
		slog.String("papel", string(u.Papel)),
	)
}

// Sessao é o registro do token opaco que foi pro cookie. O banco guarda só o
// SHA-256; o texto em claro existe uma vez, na resposta do login.
type Sessao struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	UsuarioID   uuid.UUID `gorm:"type:uuid;column:usuario_id"`
	TokenHash   []byte    `gorm:"column:token_hash"`
	CriadaEm    time.Time `gorm:"column:criada_em"`
	ExpiraEm    time.Time `gorm:"column:expira_em"`
	UltimoUsoEm time.Time `gorm:"column:ultimo_uso_em"`
}

func (Sessao) TableName() string { return "sessoes" }

// --- DTOs de entrada ---

type EntradaLogin struct {
	Email string `json:"email" validate:"required,email,max=255"`
	Senha string `json:"senha" validate:"required,min=8,max=200"`
}

func (e EntradaLogin) LogValue() slog.Value {
	return slog.StringValue("[credenciais omitidas]")
}

type EntradaCriarUsuario struct {
	Nome           string  `json:"nome" validate:"required,min=2,max=120"`
	Email          string  `json:"email" validate:"required,email,max=255"`
	Senha          string  `json:"senha" validate:"required,min=10,max=200"`
	Papel          Papel   `json:"papel" validate:"required"`
	Equipe         Equipe  `json:"equipe" validate:"required"`
	CodigoCobranca *string `json:"codigoCobranca" validate:"omitempty,max=60"`
}

func (e EntradaCriarUsuario) LogValue() slog.Value {
	return slog.GroupValue(slog.String("nome", e.Nome), slog.String("papel", string(e.Papel)))
}

type EntradaAtualizarUsuario struct {
	Nome           *string  `json:"nome" validate:"omitempty,min=2,max=120"`
	Papel          *Papel   `json:"papel"`
	Equipe         *Equipe  `json:"equipe"`
	Ativo          *bool    `json:"ativo"`
	AlcadaMaxima   *float64 `json:"alcadaMaxima" validate:"omitempty,gte=0,lte=100"`
	CodigoCobranca *string  `json:"codigoCobranca" validate:"omitempty,max=60"`
}

// --- DTOs de saída ---

// UsuarioResposta é o que sai na API. Nunca inclui senha_hash — por isso é um
// tipo à parte, e não o Usuario com `json:"-"` (fácil de quebrar sem notar).
type UsuarioResposta struct {
	ID             uuid.UUID `json:"id"`
	Nome           string    `json:"nome"`
	Email          string    `json:"email"`
	Papel          Papel     `json:"papel"`
	Equipe         Equipe    `json:"equipe"`
	Ativo          bool      `json:"ativo"`
	AlcadaMaxima   float64   `json:"alcadaMaxima"`
	CodigoCobranca *string   `json:"codigoCobranca"`
	CriadoEm       time.Time `json:"criadoEm"`
}

func Responder(u Usuario) UsuarioResposta {
	return UsuarioResposta{
		ID:             u.ID,
		Nome:           u.Nome,
		Email:          u.Email,
		Papel:          u.Papel,
		Equipe:         u.Equipe,
		Ativo:          u.Ativo,
		AlcadaMaxima:   u.AlcadaMaxima,
		CodigoCobranca: u.CodigoCobranca,
		CriadoEm:       u.CriadoEm,
	}
}
