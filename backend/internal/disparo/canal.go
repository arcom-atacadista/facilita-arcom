package disparo

import (
	"context"
	"errors"
	"log/slog"
)

// Canal é a porta de saída de mensagem do Facilita ARCOM.
//
// A implementação real é canalMeta (canal_meta.go), que fala direto com a
// Cloud API do WhatsApp — sem intermediário, porque a Meta atende empresa
// integrada direto e um BSP só acrescentaria markup por mensagem.
//
// Continua sendo interface por dois motivos práticos, não por abstração:
//
//   - Canal nulo é um estado legítimo e frequente. Sem as credenciais da Meta
//     configuradas, o sistema sobe, a régua monta a mensagem e a fila
//     enfileira — e nada é marcado como enviado. Ver ErrSemCanal.
//   - O teste do serviço e do worker exercita a fila inteira sem chamar a
//     Meta, o que é o que permite rodar a suíte sem credencial.
//
// Uma saída nova (outro provedor, fila interna da infra) é uma implementação
// nova neste pacote passada em NovoService; nada mais muda.
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

	// Livre manda Texto como corpo da mensagem, em vez de um template.
	//
	// A Meta só aceita isso dentro de uma janela de atendimento aberta — as
	// 24 horas que começam quando o cliente escreve. É o que a mesa de
	// atendimento usa para responder, e nesse caso a mensagem não é tarifada.
	//
	// Precisa ser explícito, e não inferido de "Template vazio", porque a
	// régua também preenche Texto: uma campanha que perdesse o template_meta
	// viraria texto livre em silêncio e a Meta recusaria o disparo inteiro
	// fora da janela. Quem confere se a janela está aberta é quem chama.
	Livre bool
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

// TextoLivre adapta o canal para a mesa de atendimento, que responde o cliente
// com texto livre dentro da janela de 24 horas.
//
// Mora aqui, e não no pacote conversa, porque este é o pacote que fala com a
// Meta — assim a mesa usa o mesmo canal da régua sem que um pacote passe a
// depender do outro, pelo mesmo motivo do Historico acima.
type TextoLivre struct{ canal Canal }

// NovoTextoLivre devolve nil quando não há canal, para o chamador não guardar
// uma interface não-nula com ponteiro nulo dentro — o clássico jeito de fazer
// um `if saida != nil` mentir.
func NovoTextoLivre(canal Canal) *TextoLivre {
	if canal == nil {
		return nil
	}
	return &TextoLivre{canal: canal}
}

// EnviarTextoLivre entrega a resposta e devolve a referência da Meta. Não
// confere a janela de atendimento: isso é regra da mesa, e é lá que fica.
func (t *TextoLivre) EnviarTextoLivre(ctx context.Context, telefone, texto string) (string, error) {
	return t.canal.Enviar(ctx, Mensagem{Telefone: telefone, Texto: texto, Livre: true})
}
