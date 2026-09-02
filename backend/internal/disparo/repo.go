package disparo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"facilitaarcom/internal/acesso"
)

type Repo struct{ db *gorm.DB }

func NovoRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) Criar(ctx context.Context, d *Disparo) error {
	return r.db.WithContext(ctx).Create(d).Error
}

// HouveDisparoRecente é a trava anti-spam: mesmo devedor não recebe duas
// cobranças em poucos dias, mesmo que dois operadores peçam ou que a régua
// automática coincida com um pedido manual.
func (r *Repo) HouveDisparoRecente(ctx context.Context, dividaID uuid.UUID, desde time.Time) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Disparo{}).
		Where("divida_id = ? AND criado_em > ? AND status <> ?", dividaID, desde, StatusCancelado).
		Count(&n).Error
	return n > 0, err
}

// Reservar pega até `limite` disparos vencidos da fila e já os marca como em
// processamento, numa transação só.
//
// FOR UPDATE SKIP LOCKED é o que torna isso seguro com mais de uma réplica do
// backend: cada worker leva linhas diferentes em vez de disputar as mesmas.
// Sem isso, duas réplicas mandariam a mesma cobrança para o mesmo cliente.
func (r *Repo) Reservar(ctx context.Context, agora time.Time, limite int) ([]Disparo, error) {
	var reservados []Disparo

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ids []uuid.UUID
		if err := tx.Raw(`
			SELECT id FROM disparos
			WHERE status = ? AND agendado_para <= ?
			ORDER BY agendado_para
			LIMIT ?
			FOR UPDATE SKIP LOCKED`, StatusNaFila, agora, limite).Scan(&ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}

		// tentativas sobe já na reserva: se o processo morrer no meio do
		// envio, a mensagem não fica tentando para sempre.
		if err := tx.Model(&Disparo{}).Where("id IN ?", ids).
			UpdateColumn("tentativas", gorm.Expr("tentativas + 1")).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Order("agendado_para").Find(&reservados).Error
	})

	return reservados, err
}

func (r *Repo) MarcarEnviado(ctx context.Context, id uuid.UUID, referencia string, quando time.Time) error {
	campos := map[string]any{"status": StatusEnviado, "enviado_em": quando, "erro_detalhe": nil}
	if referencia != "" {
		campos["referencia_externa"] = referencia
	}
	return r.db.WithContext(ctx).Model(&Disparo{}).Where("id = ?", id).Updates(campos).Error
}

// MarcarErro devolve o disparo para a fila com um novo agendamento (backoff),
// ou o encerra como erro quando as tentativas acabaram.
func (r *Repo) MarcarErro(ctx context.Context, id uuid.UUID, detalhe string, proxima *time.Time) error {
	campos := map[string]any{"erro_detalhe": detalhe}
	if proxima != nil {
		campos["status"] = StatusNaFila
		campos["agendado_para"] = *proxima
	} else {
		campos["status"] = StatusErro
	}
	return r.db.WithContext(ctx).Model(&Disparo{}).Where("id = ?", id).Updates(campos).Error
}

func (r *Repo) Cancelar(ctx context.Context, id uuid.UUID, motivo string) error {
	return r.db.WithContext(ctx).Model(&Disparo{}).Where("id = ?", id).
		Updates(map[string]any{"status": StatusCancelado, "erro_detalhe": motivo}).Error
}

// CancelarVencidos descarta o que envelheceu na fila. Cobrar por uma dívida
// que pode já ter sido paga há semanas é pior do que não cobrar — e é o que
// aconteceria no dia em que um canal fosse ligado com a fila cheia de
// mensagens antigas.
func (r *Repo) CancelarVencidos(ctx context.Context, limite time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Model(&Disparo{}).
		Where("status = ? AND agendado_para < ?", StatusNaFila, limite).
		Updates(map[string]any{
			"status":       StatusCancelado,
			"erro_detalhe": "cancelado por tempo na fila — a mensagem ficou velha demais para ser enviada",
		})
	return res.RowsAffected, res.Error
}

// Listar respeita o mesmo recorte de carteira da cobrança: histórico de
// disparo é dado de devedor.
func (r *Repo) Listar(ctx context.Context, u acesso.Usuario, pagina, porPagina int) ([]Disparo, int64, error) {
	q := r.db.WithContext(ctx).Model(&Disparo{}).Joins("JOIN dividas ON dividas.id = disparos.divida_id")

	if !u.Papel.VeCarteiraInteira() {
		if u.CodigoCobranca == nil || *u.CodigoCobranca == "" {
			q = q.Where("1 = 0")
		} else {
			q = q.Where("dividas.responsavel_cobranca = ?", *u.CodigoCobranca)
		}
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var disparos []Disparo
	err := q.Order("disparos.criado_em DESC").
		Limit(porPagina).Offset((pagina - 1) * porPagina).
		Find(&disparos).Error
	return disparos, total, err
}

func (r *Repo) ContarPorStatus(ctx context.Context) (map[string]int64, error) {
	var linhas []struct {
		Status string
		Total  int64
	}
	if err := r.db.WithContext(ctx).Model(&Disparo{}).
		Select("status, count(*) AS total").Group("status").Scan(&linhas).Error; err != nil {
		return nil, err
	}

	out := make(map[string]int64, len(linhas))
	for _, l := range linhas {
		out[l.Status] = l.Total
	}
	return out, nil
}
