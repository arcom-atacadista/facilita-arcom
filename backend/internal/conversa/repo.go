package conversa

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"facilitaarcom/internal/acesso"
)

type Repo struct{ db *gorm.DB }

func NovoRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

var ErrNaoEncontrado = gorm.ErrRecordNotFound

// AbrirOuAtualizar acha a conversa do telefone ou cria uma. O ON CONFLICT
// evita a corrida entre dois webhooks do mesmo número chegando juntos, que
// numa leitura-antes-de-gravar criaria duas conversas.
func (r *Repo) AbrirOuAtualizar(ctx context.Context, telefone string, nomePerfil *string, agora time.Time) (Conversa, error) {
	c := Conversa{
		ID:           uuid.New(),
		Telefone:     telefone,
		NomePerfil:   nomePerfil,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "telefone"}},
		DoUpdates: clause.Assignments(map[string]any{
			// coalesce: um webhook sem nome de perfil não pode apagar o que
			// já sabíamos sobre a pessoa.
			"nome_perfil":   gorm.Expr("coalesce(EXCLUDED.nome_perfil, conversas.nome_perfil)"),
			"atualizado_em": agora,
		}),
	}).Create(&c).Error
	if err != nil {
		return Conversa{}, err
	}

	// O Create com ON CONFLICT não devolve a linha existente, então relemos.
	return r.PorTelefone(ctx, telefone)
}

func (r *Repo) PorTelefone(ctx context.Context, telefone string) (Conversa, error) {
	var c Conversa
	err := r.db.WithContext(ctx).First(&c, "telefone = ?", telefone).Error
	return c, err
}

func (r *Repo) PorID(ctx context.Context, id uuid.UUID) (Conversa, error) {
	var c Conversa
	err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error
	return c, err
}

// LigarAoCliente casa a conversa com o cadastro quando reconhecemos o número.
func (r *Repo) LigarAoCliente(ctx context.Context, conversaID, clienteID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&Conversa{}).
		Where("id = ? AND cliente_id IS NULL", conversaID).
		Update("cliente_id", clienteID).Error
}

// RegistrarEntrada grava a mensagem do cliente e move o estado da conversa.
// Devolve falso quando a mensagem já existia — a Meta reentrega o webhook
// quando a nossa resposta demora, e sem isso a mesma fala apareceria duas
// vezes na tela do operador.
func (r *Repo) RegistrarEntrada(ctx context.Context, m *Mensagem, janelaExpiraEm *time.Time) (bool, error) {
	nova := true

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "wamid"}},
			DoNothing: true,
		}).Create(m)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			nova = false
			return nil
		}

		campos := map[string]any{
			"ultima_mensagem_em": m.OcorridaEm,
			"nao_lidas":          gorm.Expr("conversas.nao_lidas + 1"),
			"atualizado_em":      time.Now().UTC(),
		}
		if janelaExpiraEm != nil {
			campos["janela_expira_em"] = *janelaExpiraEm
		}
		return tx.Model(&Conversa{}).Where("id = ?", m.ConversaID).Updates(campos).Error
	})

	return nova, err
}

// RegistrarSaida guarda a mensagem que a plataforma mandou, para a conversa
// aparecer completa na mesa.
//
// Também avança ultima_mensagem_em, que é como a fila se ordena. Sem isso a
// coluna só andava com fala do cliente (RegistrarEntrada): uma conversa que
// acabou de receber resposta da mesa, ou a cobrança da régua, ficava parada no
// horário antigo e a tela mostrava "última mensagem" errada.
//
// greatest e não atribuição direta: aviso de entrega da Meta chega fora de
// ordem, e uma mensagem antiga registrada depois não pode puxar a conversa
// para trás no tempo.
func (r *Repo) RegistrarSaida(ctx context.Context, m *Mensagem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "wamid"}},
			DoNothing: true,
		}).Create(m)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // reentrega: já tínhamos esta mensagem
		}

		return tx.Model(&Conversa{}).Where("id = ?", m.ConversaID).Updates(map[string]any{
			"ultima_mensagem_em": gorm.Expr(
				"greatest(coalesce(conversas.ultima_mensagem_em, ?), ?)", m.OcorridaEm, m.OcorridaEm),
			"atualizado_em": time.Now().UTC(),
		}).Error
	})
}

// AtualizarStatusDaSaida aplica o aviso de entrega da Meta.
//
// O status só avança: a Meta não garante ordem de entrega dos webhooks, e um
// aviso de "entregue" chegando depois de "lida" não pode fazer a mensagem
// regredir na tela.
var ordemDoStatus = map[string]int{
	StatusEnviada: 1, StatusEntregue: 2, StatusLida: 3, StatusFalhou: 4,
}

func (r *Repo) AtualizarStatusDaSaida(ctx context.Context, wamid, status string, erroCodigo *int) error {
	novo := ordemDoStatus[status]
	if novo == 0 {
		return nil // status que não acompanhamos
	}

	campos := map[string]any{"status": status}
	if erroCodigo != nil {
		campos["erro_codigo"] = *erroCodigo
	}

	return r.db.WithContext(ctx).Model(&Mensagem{}).
		Where("wamid = ? AND (status IS NULL OR ?::int > ?)",
			wamid, novo, gorm.Expr("(CASE status WHEN 'enviada' THEN 1 WHEN 'entregue' THEN 2 WHEN 'lida' THEN 3 WHEN 'falhou' THEN 4 ELSE 0 END)")).
		Updates(campos).Error
}

// MarcarEntregaDoDisparo leva o aviso da Meta para a linha da régua, que é
// onde o operador acompanha a cobrança.
func (r *Repo) MarcarEntregaDoDisparo(ctx context.Context, wamid, status string, quando time.Time, detalhe *string) error {
	campos := map[string]any{}
	switch status {
	case StatusEntregue:
		campos["entregue_em"] = quando
	case StatusLida:
		campos["lido_em"] = quando
		// Quem leu, recebeu — mesmo que o aviso de entrega tenha se perdido.
		campos["entregue_em"] = gorm.Expr("coalesce(entregue_em, ?)", quando)
	case StatusFalhou:
		campos["status"] = "erro"
		if detalhe != nil {
			campos["erro_detalhe"] = *detalhe
		}
	default:
		return nil
	}

	return r.db.WithContext(ctx).Table("disparos").
		Where("referencia_externa = ?", wamid).
		Updates(campos).Error
}

// ClientePorTelefone reconhece o número na carteira. Compara só os dígitos
// finais porque o cadastro guarda o número sem DDI e a Meta manda com ele.
func (r *Repo) ClientePorTelefone(ctx context.Context, telefoneComDDI string) (uuid.UUID, error) {
	// Os 8 últimos dígitos bastam para casar sem depender de DDD ou do nono
	// dígito, e o filtro por sufixo é seguro contra número curto.
	if len(telefoneComDDI) < 8 {
		return uuid.Nil, ErrNaoEncontrado
	}
	sufixo := telefoneComDDI[len(telefoneComDDI)-8:]

	// Escaneado como texto e convertido depois: uuid.UUID é [16]byte, e o
	// driver tentaria encaixar a string do Postgres byte a byte.
	var bruto *string
	err := r.db.WithContext(ctx).Table("clientes").
		Select("id::text").
		Where("regexp_replace(coalesce(telefone,''), '\\D', '', 'g') LIKE ?", "%"+sufixo).
		Limit(1).Scan(&bruto).Error
	if err != nil {
		return uuid.Nil, err
	}
	if bruto == nil || *bruto == "" {
		return uuid.Nil, ErrNaoEncontrado
	}

	id, err := uuid.Parse(*bruto)
	if err != nil {
		return uuid.Nil, fmt.Errorf("id de cliente inválido no banco: %w", err)
	}
	return id, nil
}

// --- leitura pela mesa ---

// ListarConversas respeita o mesmo recorte de carteira do resto do sistema:
// conversa é dado de devedor. Conversa de número desconhecido só aparece para
// quem enxerga a carteira inteira — não é de ninguém ainda.
func (r *Repo) ListarConversas(ctx context.Context, u acesso.Usuario, pagina, porPagina int) ([]Conversa, int64, error) {
	q := r.db.WithContext(ctx).Model(&Conversa{})

	if !u.Papel.VeCarteiraInteira() {
		if u.CodigoCobranca == nil || *u.CodigoCobranca == "" {
			q = q.Where("1 = 0")
		} else {
			q = q.Where(`conversas.cliente_id IN (
				SELECT d.cliente_id FROM dividas d WHERE d.responsavel_cobranca = ?
			)`, *u.CodigoCobranca)
		}
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var conversas []Conversa
	err := q.Order("ultima_mensagem_em DESC NULLS LAST").
		Limit(porPagina).Offset((pagina - 1) * porPagina).
		Find(&conversas).Error
	return conversas, total, err
}

// ResumoDaFila é o que a mesa precisa por conversa além do que a tabela
// conversas guarda: o nome do cliente na carteira e a última fala.
//
// Vem numa consulta só, por conversa em lote, e não por linha na tela — a fila
// tem 20 itens por página e uma consulta por item seriam 40 idas ao banco a
// cada refresh.
type ResumoDaFila struct {
	ConversaID    uuid.UUID `gorm:"column:conversa_id"`
	NomeCliente   *string   `gorm:"column:nome_cliente"`
	Documento     *string   `gorm:"column:documento"`
	UltimoTexto   *string   `gorm:"column:ultimo_texto"`
	UltimaDirecao *string   `gorm:"column:ultima_direcao"`
}

// ResumoDaFilaPor devolve o resumo das conversas pedidas, indexado por id.
// Lista vazia devolve mapa vazio sem tocar no banco — `IN ()` é erro de sintaxe
// no Postgres.
func (r *Repo) ResumoDaFilaPor(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]ResumoDaFila, error) {
	porID := make(map[uuid.UUID]ResumoDaFila, len(ids))
	if len(ids) == 0 {
		return porID, nil
	}

	var linhas []ResumoDaFila
	// DISTINCT ON pega a mensagem mais recente de cada conversa numa varredura
	// só; o LEFT JOIN mantém a conversa de número que ainda não casou com
	// nenhum cliente da carteira.
	err := r.db.WithContext(ctx).Raw(`
		SELECT c.id           AS conversa_id,
		       cl.nome        AS nome_cliente,
		       cl.documento   AS documento,
		       m.texto        AS ultimo_texto,
		       m.direcao      AS ultima_direcao
		  FROM conversas c
		  LEFT JOIN clientes cl ON cl.id = c.cliente_id
		  LEFT JOIN (
		        SELECT DISTINCT ON (conversa_id) conversa_id, texto, direcao
		          FROM mensagens
		         ORDER BY conversa_id, ocorrida_em DESC
		  ) m ON m.conversa_id = c.id
		 WHERE c.id IN ?`, ids).Scan(&linhas).Error
	if err != nil {
		return nil, err
	}

	for _, l := range linhas {
		porID[l.ConversaID] = l
	}
	return porID, nil
}

func (r *Repo) MensagensDaConversa(ctx context.Context, conversaID uuid.UUID, limite int) ([]Mensagem, error) {
	var ms []Mensagem
	err := r.db.WithContext(ctx).
		Where("conversa_id = ?", conversaID).
		Order("ocorrida_em DESC").Limit(limite).
		Find(&ms).Error
	// Devolvidas em ordem cronológica, que é como a mesa mostra.
	for i, j := 0, len(ms)-1; i < j; i, j = i+1, j-1 {
		ms[i], ms[j] = ms[j], ms[i]
	}
	return ms, err
}

func (r *Repo) ZerarNaoLidas(ctx context.Context, conversaID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&Conversa{}).
		Where("id = ?", conversaID).Update("nao_lidas", 0).Error
}

func EhNaoEncontrado(err error) bool { return errors.Is(err, ErrNaoEncontrado) }
