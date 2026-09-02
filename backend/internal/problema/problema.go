// Package problema é o contrato de erro da API — RFC 9457, ver
// system-design/padroes/09-contrato-api.md.
//
// Por que é um pacote próprio e não fica em internal/servidor: rotas.go
// importa cada pacote de feature para montar a árvore de rotas, e cada
// feature precisa dos construtores de erro. Se os construtores morassem em
// servidor, os dois se importariam e o Go recusaria o ciclo. Com o contrato
// aqui, a dependência é de mão única — feature -> problema <- servidor — e
// servidor.NaoEncontrado continua existindo como alias, do jeito que a skill
// novo-endpoint descreve.
package problema

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Problema é o corpo de toda resposta de erro (Content-Type:
// application/problem+json).
type Problema struct {
	Type   string            `json:"type"`
	Title  string            `json:"title"`
	Status int               `json:"status"`
	Detail string            `json:"detail"`
	Codigo string            `json:"codigo"`
	Campos map[string]string `json:"campos,omitempty"`
}

// ErroDominio é o erro que uma feature devolve quando já sabe o status e o
// código corretos. O wrapper H() em internal/servidor converte para HTTP.
type ErroDominio struct {
	Status int
	Titulo string
	Detail string
	Codigo string
	Campos map[string]string
}

func (e *ErroDominio) Error() string { return e.Detail }

func NaoEncontrado(detalhe string) error {
	return &ErroDominio{Status: http.StatusNotFound, Titulo: "Não encontrado", Detail: detalhe, Codigo: "nao_encontrado"}
}

func Conflito(detalhe string) error {
	return &ErroDominio{Status: http.StatusConflict, Titulo: "Conflito", Detail: detalhe, Codigo: "conflito"}
}

func SemPermissao() error {
	return &ErroDominio{Status: http.StatusForbidden, Titulo: "Sem permissão", Detail: "Você não tem acesso a este recurso.", Codigo: "sem_permissao"}
}

func NaoAutenticado(detalhe string) error {
	return &ErroDominio{Status: http.StatusUnauthorized, Titulo: "Não autenticado", Detail: detalhe, Codigo: "token_ausente"}
}

func SessaoExpirada() error {
	return &ErroDominio{Status: http.StatusUnauthorized, Titulo: "Sessão inválida", Detail: "Sua sessão expirou. Faça login novamente.", Codigo: "token_expirado"}
}

// Validacao recebe campo -> mensagem. Sempre 422.
func Validacao(campos map[string]string) error {
	return &ErroDominio{
		Status: http.StatusUnprocessableEntity,
		Titulo: "Dados inválidos",
		Detail: "Um ou mais campos estão inválidos.",
		Codigo: "validacao",
		Campos: campos,
	}
}

// UmCampo é o atalho para o caso comum de um único campo inválido.
func UmCampo(campo, mensagem string) error {
	return Validacao(map[string]string{campo: mensagem})
}

// Requisicao é o 400 de corpo malformado (JSON inválido, tipo errado) —
// distinto do 422, que é semântico.
func Requisicao(detalhe string) error {
	return &ErroDominio{Status: http.StatusBadRequest, Titulo: "Requisição inválida", Detail: detalhe, Codigo: "requisicao_invalida"}
}

// Escrever é o único lugar que serializa um Problema na resposta. Fica aqui
// e não em internal/servidor porque middleware de feature (o de sessão, por
// exemplo) também precisa recusar uma requisição no formato do contrato, e
// middleware não passa pelo wrapper H().
//
// Não loga nada de propósito: quem chama decide o que registrar. Erro 5xx
// tem que ser logado por quem o conhece, com o request_id em mãos.
func Escrever(w http.ResponseWriter, status int, titulo, detalhe, codigo string, campos map[string]string) error {
	p := Problema{
		Type:   "about:blank",
		Title:  titulo,
		Status: status,
		Detail: detalhe,
		Codigo: codigo,
		Campos: campos,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(p)
}

// EscreverErro serializa um ErroDominio já pronto. Se o erro não for de
// domínio, sai o genérico de 500 — nunca o texto do erro original.
func EscreverErro(w http.ResponseWriter, err error) error {
	var ed *ErroDominio
	if errors.As(err, &ed) {
		return Escrever(w, ed.Status, ed.Titulo, ed.Detail, ed.Codigo, ed.Campos)
	}
	return Escrever(w, http.StatusInternalServerError, "Erro interno", "Erro interno. Tente novamente.", "erro_interno", nil)
}
