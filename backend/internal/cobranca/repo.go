package cobranca

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"facilitaarcom/internal/acesso"
)

type Repo struct{ db *gorm.DB }

func NovoRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

var ErrNaoEncontrado = gorm.ErrRecordNotFound

// FiltroDividas são os filtros de tela. Escopo NÃO entra aqui de propósito:
// o recorte por carteira é aplicado sempre, por aplicarEscopo, e não pode
// depender de um campo que a tela mandou (ou deixou de mandar).
type FiltroDividas struct {
	Faixa     Faixa
	Status    string
	Busca     string
	Pagina    int
	PorPagina int
}

// aplicarEscopo é a regra de LGPD do sistema: quem não é coordenação vê
// apenas a própria carteira, e quem não tem código de cobrança não vê nada.
// Falha fechado de propósito — no modelo Supabase anterior, qualquer usuário
// autenticado lia a base inteira de devedores.
func aplicarEscopo(q *gorm.DB, u acesso.Usuario) *gorm.DB {
	if u.Papel.VeCarteiraInteira() {
		return q
	}
	if u.CodigoCobranca == nil || *u.CodigoCobranca == "" {
		return q.Where("1 = 0")
	}
	return q.Where("dividas.responsavel_cobranca = ?", *u.CodigoCobranca)
}

func (r *Repo) ListarDividas(ctx context.Context, u acesso.Usuario, f FiltroDividas, agora time.Time) ([]Divida, int64, error) {
	q := r.db.WithContext(ctx).Model(&Divida{}).Joins("JOIN clientes ON clientes.id = dividas.cliente_id")
	q = aplicarEscopo(q, u)

	if f.Status != "" {
		q = q.Where("dividas.status = ?", f.Status)
	}

	// A faixa é derivada do vencimento, então vira intervalo de datas em vez
	// de coluna — assim o índice de vencimento é usado e não precisamos de
	// coluna calculada que envelhece todo dia.
	if min, max, ok := limitesDaFaixa(f.Faixa); ok {
		hoje := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, time.UTC)
		q = q.Where("dividas.vencimento BETWEEN ? AND ?",
			hoje.AddDate(0, 0, -max).Format(FormatoData),
			hoje.AddDate(0, 0, -min).Format(FormatoData))
	}

	if f.Busca != "" {
		// ILIKE com o termo como parâmetro (?), nunca concatenado — o % faz
		// parte do valor, não do SQL.
		termo := "%" + f.Busca + "%"
		q = q.Where("clientes.nome ILIKE ? OR clientes.documento ILIKE ? OR dividas.contrato ILIKE ?", termo, termo, termo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var dividas []Divida
	err := q.Preload("Cliente").
		Order("dividas.vencimento ASC").
		Limit(f.PorPagina).Offset((f.Pagina - 1) * f.PorPagina).
		Find(&dividas).Error
	return dividas, total, err
}

func limitesDaFaixa(f Faixa) (min, max int, ok bool) {
	switch f {
	case Faixa3a30:
		return 3, 30, true
	case Faixa31a60:
		return 31, 60, true
	case Faixa61a90:
		return 61, 90, true
	default:
		return 0, 0, false
	}
}

// DividaDoUsuario carrega uma dívida já respeitando o escopo — é o que
// impede trocar o id na URL pra ver a carteira de outro analista (IDOR).
func (r *Repo) DividaDoUsuario(ctx context.Context, u acesso.Usuario, id uuid.UUID) (Divida, error) {
	var d Divida
	q := r.db.WithContext(ctx).Model(&Divida{}).Where("dividas.id = ?", id)
	err := aplicarEscopo(q, u).Preload("Cliente").First(&d).Error
	return d, err
}

// DividaPorID não aplica escopo — só o fluxo público de negociação usa, onde
// quem autoriza é a posse do token, não uma sessão.
func (r *Repo) DividaPorID(ctx context.Context, id uuid.UUID) (Divida, error) {
	var d Divida
	err := r.db.WithContext(ctx).Preload("Cliente").First(&d, "id = ?", id).Error
	return d, err
}

func (r *Repo) DividaPorTokenHash(ctx context.Context, hash []byte, agora time.Time) (Divida, error) {
	var d Divida
	err := r.db.WithContext(ctx).Preload("Cliente").
		Where("token_hash = ? AND token_expira_em > ?", hash, agora).
		First(&d).Error
	return d, err
}

func (r *Repo) SalvarDivida(ctx context.Context, d *Divida) error {
	d.AtualizadoEm = time.Now().UTC()
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *Repo) DefinirToken(ctx context.Context, id uuid.UUID, hash []byte, expiraEm time.Time) error {
	return r.db.WithContext(ctx).Model(&Divida{}).Where("id = ?", id).
		Updates(map[string]any{
			"token_hash":      hash,
			"token_expira_em": expiraEm,
			"atualizado_em":   time.Now().UTC(),
		}).Error
}

// --- políticas ---

func (r *Repo) ListarPoliticas(ctx context.Context) ([]Politica, error) {
	var ps []Politica
	err := r.db.WithContext(ctx).Order("faixa_min").Find(&ps).Error
	return ps, err
}

// PoliticaDaFaixa acha a política que cobre um dia de atraso. O banco garante
// que no máximo uma cobre cada dia (constraint politicas_faixa_sem_sobreposicao),
// então não há ambiguidade de qual vem primeiro.
func (r *Repo) PoliticaDaFaixa(ctx context.Context, dias int) (*Politica, error) {
	var p Politica
	err := r.db.WithContext(ctx).
		Where("faixa_min <= ? AND faixa_max >= ?", dias, dias).
		First(&p).Error
	if err != nil {
		if errors.Is(err, ErrNaoEncontrado) {
			return nil, nil // fora da régua: sem desconto, não é erro
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repo) PoliticaPorID(ctx context.Context, id uuid.UUID) (Politica, error) {
	var p Politica
	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	return p, err
}

func (r *Repo) SalvarPolitica(ctx context.Context, p *Politica) error {
	p.AtualizadoEm = time.Now().UTC()
	return r.db.WithContext(ctx).Save(p).Error
}

// --- campanhas ---

func (r *Repo) CampanhaDaFaixa(ctx context.Context, dias int) (*Campanha, error) {
	var c Campanha
	err := r.db.WithContext(ctx).
		Where("ativo AND faixa_min <= ? AND faixa_max >= ?", dias, dias).
		Order("faixa_min").First(&c).Error
	if err != nil {
		if errors.Is(err, ErrNaoEncontrado) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repo) ListarCampanhas(ctx context.Context) ([]Campanha, error) {
	var cs []Campanha
	err := r.db.WithContext(ctx).Order("faixa_min").Find(&cs).Error
	return cs, err
}

// --- acordos ---

func (r *Repo) AcordoAtivoDaDivida(ctx context.Context, dividaID uuid.UUID) (*Acordo, error) {
	var a Acordo
	err := r.db.WithContext(ctx).Preload("Lista", func(db *gorm.DB) *gorm.DB {
		return db.Order("parcelas.numero")
	}).Where("divida_id = ? AND status = ?", dividaID, StatusAcordoAtivo).First(&a).Error
	if err != nil {
		if errors.Is(err, ErrNaoEncontrado) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// GravarAcordo grava acordo, parcelas e a virada de status da dívida numa
// transação só: sem isso, uma falha no meio deixaria acordo sem parcela, ou
// dívida marcada como negociada sem acordo nenhum.
func (r *Repo) GravarAcordo(ctx context.Context, a *Acordo, parcelas []Parcela) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(a).Error; err != nil {
			return err
		}
		if len(parcelas) > 0 {
			if err := tx.Create(&parcelas).Error; err != nil {
				return err
			}
		}
		return tx.Model(&Divida{}).Where("id = ?", a.DividaID).
			Updates(map[string]any{"status": StatusDividaNegociada, "atualizado_em": time.Now().UTC()}).Error
	})
}

func (r *Repo) ListarAcordos(ctx context.Context, u acesso.Usuario, pagina, porPagina int) ([]Acordo, int64, error) {
	q := r.db.WithContext(ctx).Model(&Acordo{}).Joins("JOIN dividas ON dividas.id = acordos.divida_id")
	q = aplicarEscopo(q, u)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var acordos []Acordo
	err := q.Preload("Lista", func(db *gorm.DB) *gorm.DB { return db.Order("parcelas.numero") }).
		Order("acordos.criado_em DESC").
		Limit(porPagina).Offset((pagina - 1) * porPagina).
		Find(&acordos).Error
	return acordos, total, err
}

// EhDuplicado reconhece a violação do índice único parcial de acordo ativo —
// é o que transforma a corrida de dois aceites simultâneos num 409 limpo.
func EhDuplicado(err error) bool { return errors.Is(err, gorm.ErrDuplicatedKey) }
