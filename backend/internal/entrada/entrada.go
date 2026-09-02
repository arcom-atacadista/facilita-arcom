// Package entrada decodifica e valida o corpo de uma requisição. Existe para
// que nenhum handler repita o par "decodifica JSON + roda o validator", que é
// exatamente onde é fácil esquecer o DisallowUnknownFields e abrir mass
// assignment (ver system-design/padroes/03-backend.md).
package entrada

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"

	"facilitaarcom/internal/problema"
)

// validador é reutilizado entre requisições: validator.New() compila cache de
// reflexão por struct, criar um por request joga esse cache fora. É seguro
// para uso concorrente.
var validador = validator.New(validator.WithRequiredStructEnabled())

// Decodificar lê o corpo JSON em destino e valida as tags do validator.
// Devolve erro já no formato do contrato: 400 para corpo malformado, 422 com
// o mapa de campos para validação.
func Decodificar(r *http.Request, destino any) error {
	dec := json.NewDecoder(r.Body)
	// Campo desconhecido é recusado em vez de ignorado: é o que impede um
	// cliente de mandar {"papel":"gerencia"} num endpoint que só deveria
	// aceitar {"nome":...} e ter sorte de o DTO ter esse campo.
	dec.DisallowUnknownFields()

	if err := dec.Decode(destino); err != nil {
		return erroDeCorpo(err)
	}

	// Um segundo objeto depois do primeiro é corpo malformado, não "sobra".
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return problema.Requisicao("O corpo da requisição deve conter um único objeto JSON.")
	}

	if err := validador.Struct(destino); err != nil {
		var invalidas validator.ValidationErrors
		if errors.As(err, &invalidas) {
			return problema.Validacao(mensagens(invalidas))
		}
		return err
	}
	return nil
}

func erroDeCorpo(err error) error {
	var maxBytes *http.MaxBytesError
	if errors.As(err, &maxBytes) {
		return problema.Requisicao("Corpo da requisição grande demais.")
	}

	var desconhecido *json.UnmarshalTypeError
	if errors.As(err, &desconhecido) {
		return problema.UmCampo(desconhecido.Field, "Tipo de dado inválido para este campo.")
	}

	if errors.Is(err, io.EOF) {
		return problema.Requisicao("O corpo da requisição está vazio.")
	}

	// Mensagem do encoding/json pode citar posição/estrutura interna do
	// payload; não vai pro cliente.
	if strings.Contains(err.Error(), "unknown field") {
		return problema.Requisicao("O corpo da requisição contém campos não reconhecidos.")
	}
	return problema.Requisicao("O corpo da requisição não é um JSON válido.")
}

// mensagens traduz o erro do validator para o mapa campo -> texto em
// português que o contrato de API pede.
func mensagens(erros validator.ValidationErrors) map[string]string {
	out := make(map[string]string, len(erros))
	for _, e := range erros {
		out[nomeDoCampo(e)] = textoDaRegra(e)
	}
	return out
}

// nomeDoCampo devolve o nome como o frontend o conhece (o do JSON), não o do
// campo Go — senão a tela não consegue casar o erro com o input.
func nomeDoCampo(e validator.FieldError) string {
	nome := e.Field()
	return strings.ToLower(nome[:1]) + nome[1:]
}

func textoDaRegra(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "Campo obrigatório."
	case "email":
		return "Informe um e-mail válido."
	case "min":
		return fmt.Sprintf("Deve ter no mínimo %s caracteres.", e.Param())
	case "max":
		return fmt.Sprintf("Deve ter no máximo %s caracteres.", e.Param())
	case "gte":
		return fmt.Sprintf("Deve ser maior ou igual a %s.", e.Param())
	case "lte":
		return fmt.Sprintf("Deve ser menor ou igual a %s.", e.Param())
	case "oneof":
		return "Valor não permitido para este campo."
	default:
		return "Valor inválido."
	}
}
