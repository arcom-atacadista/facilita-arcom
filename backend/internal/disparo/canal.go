package disparo

import (
	"context"
	"errors"
	"log/slog"
)

// Canal é a porta de saída de mensagem do Facilita ARCOM.
//
// ─────────────────────────────────────────────────────────────────────────
// POR QUE ISTO É UMA INTERFACE SEM IMPLEMENTAÇÃO REAL
//
// Hoje o envio de WhatsApp para a carteira em atraso é feito pela Nines, e o
// Facilita ARCOM não tem por onde disparar:
//
//   - O Gateway ARCOM é somente leitura (todas as rotas são GET). O dataset
//     disparo-mensagem-nines conta o que a Nines já enviou; não envia.
//   - Não existe provedor de mensagem no allowlist da stack
//     (system-design/padroes/01-stack-permitida.md).
//   - A fila do RabbitMQ existe, mas 03-backend.md é explícito: nome de fila
//     e credencial vêm do time de infra, nunca inventados.
//
// Então a regra de 05-regras-para-a-ia.md se aplica: não inventar o mecanismo
// de acesso. Toda a camada de disparo está pronta — montagem da mensagem por
// campanha, fila em Postgres, trava anti-spam, worker com claim atômico. Só a
// última milha está aberta, atrás desta interface.
//
// Para ligar o envio de verdade, basta uma implementação nova neste pacote
// (nada mais muda):
//
//	type canalNines struct{ ... }
//	func (c *canalNines) Nome() string { return "nines" }
//	func (c *canalNines) Enviar(ctx context.Context, m Mensagem) (string, error) { ... }
//
// e passá-la em NovoService. As três perguntas que precisam de resposta antes:
// a Nines expõe API de envio? Se sim, qual URL/autenticação? Se não, existe
// fila da infra que um serviço interno consome — e qual o nome dela?
// ─────────────────────────────────────────────────────────────────────────
type Canal interface {
	// Nome identifica o canal nos registros (vai para disparos.canal).
	Nome() string

	// Enviar entrega a mensagem e devolve a referência que o provedor deu,
	// para conciliar depois com o histórico do Gateway (campo idReferencia do
	// dataset disparo-mensagem-nines).
	Enviar(ctx context.Context, m Mensagem) (referencia string, err error)
}

// Mensagem é o que sai pelo canal.
//
// Fora de uma janela de atendimento aberta, a Meta não aceita texto livre —
// só um template aprovado, pelo nome, com os valores em ordem para os
// marcadores {{1}}, {{2}}, ... Por isso Texto NÃO é o que viaja na chamada:
// ele é o registro do que foi montado, para o operador ver na tela e para a
// auditoria. Quem vai para a Meta é Template + Parametros.
type Mensagem struct {
	// Telefone já normalizado em dígitos com DDI (ver TelefoneComDDI).
	Telefone string

	// Texto montado a partir do template da campanha — registro, não payload.
	Texto string

	// Template é o nome como foi aprovado na Meta.
	Template string

	// Idioma no formato da Meta, ex.: pt_BR.
	Idioma string

	// Parametros são os valores na ordem dos marcadores do template.
	Parametros []string
}

// LogValue mascara o telefone: é dado pessoal de devedor e não vai cru pro
// log (03-backend.md).
func (m Mensagem) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("telefone", mascararTelefone(m.Telefone)),
		slog.Int("tamanho_texto", len(m.Texto)),
	)
}

func mascararTelefone(t string) string {
	if len(t) <= 4 {
		return "****"
	}
	return "****" + t[len(t)-4:]
}

// ErrSemCanal é o estado atual do sistema: existe fila, não existe saída.
// A fila NÃO é esvaziada nem marcada como enviada quando isto acontece — as
// mensagens ficam em na_fila esperando um canal. Marcar como enviado sem ter
// enviado seria a pior saída possível: a operação acharia que falou com o
// cliente.
var ErrSemCanal = errors.New("nenhum canal de envio configurado")
