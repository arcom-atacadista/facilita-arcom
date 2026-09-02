package servidor

import (
	"errors"
	"net/http"

	"facilitaarcom/internal/problema"
)

// O contrato de erro em si mora em internal/problema (ver o comentário de
// pacote lá para o motivo). Estes aliases mantêm servidor.NaoEncontrado(...)
// e servidor.ErroDominio válidos, como a skill novo-endpoint descreve.
type (
	Problema    = problema.Problema
	ErroDominio = problema.ErroDominio
)

var (
	NaoEncontrado  = problema.NaoEncontrado
	Conflito       = problema.Conflito
	SemPermissao   = problema.SemPermissao
	NaoAutenticado = problema.NaoAutenticado
	SessaoExpirada = problema.SessaoExpirada
	Validacao      = problema.Validacao
	UmCampo        = problema.UmCampo
	Requisicao     = problema.Requisicao
)

// apiHandler é um handler que pode devolver erro — elimina o
// `if err != nil { w.WriteHeader(500); return }` repetido em cada handler,
// que é onde é fácil vazar detalhe interno sem querer.
// Alias (=) e não tipo próprio: rotas.go passa s.H para os pacotes de
// feature montarem seus handlers, e um tipo nomeado aqui tornaria essa
// passagem incompatível com a assinatura que a feature declara.
type apiHandler = func(w http.ResponseWriter, r *http.Request) error

// H adapta um apiHandler para http.HandlerFunc, convertendo erro em Problema
// pelo único ponto de conversão do backend. Nenhum handler escreve JSON de
// erro na mão.
func (s *Servidor) H(fn apiHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			s.tratarErro(w, r, err)
		}
	}
}

func (s *Servidor) tratarErro(w http.ResponseWriter, r *http.Request, err error) {
	var ed *ErroDominio
	if errors.As(err, &ed) {
		s.escreverProblema(w, r, ed.Status, ed.Titulo, ed.Detail, ed.Codigo, ed.Campos)
		return
	}

	// Erro não mapeado: loga tudo no servidor, devolve só o genérico pro
	// cliente. Nunca err.Error() no corpo — pode conter nome de tabela,
	// driver do banco ou caminho de arquivo.
	s.log.ErrorContext(r.Context(), "erro interno não tratado", "erro", err, "caminho", r.URL.Path, "metodo", r.Method)
	s.escreverProblema(w, r, http.StatusInternalServerError, "Erro interno", "Erro interno. Tente novamente.", "erro_interno", nil)
}

func (s *Servidor) escreverProblema(w http.ResponseWriter, r *http.Request, status int, titulo, detalhe, codigo string, campos map[string]string) {
	if err := problema.Escrever(w, status, titulo, detalhe, codigo, campos); err != nil {
		s.log.ErrorContext(r.Context(), "falha ao codificar problema", "erro", err)
	}
}
