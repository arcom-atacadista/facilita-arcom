package acesso

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repo struct{ db *gorm.DB }

func NovoRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

var ErrNaoEncontrado = gorm.ErrRecordNotFound

// PorEmail busca ignorando maiúscula/minúscula — o índice único do banco é
// sobre lower(email), então a busca precisa casar com ele.
func (r *Repo) PorEmail(ctx context.Context, email string) (Usuario, error) {
	var u Usuario
	err := r.db.WithContext(ctx).
		Where("lower(email) = ?", strings.ToLower(strings.TrimSpace(email))).
		First(&u).Error
	return u, err
}

func (r *Repo) PorID(ctx context.Context, id uuid.UUID) (Usuario, error) {
	var u Usuario
	err := r.db.WithContext(ctx).First(&u, "id = ?", id).Error
	return u, err
}

func (r *Repo) Listar(ctx context.Context) ([]Usuario, error) {
	var us []Usuario
	err := r.db.WithContext(ctx).Order("nome").Find(&us).Error
	return us, err
}

func (r *Repo) Criar(ctx context.Context, u *Usuario) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *Repo) Salvar(ctx context.Context, u *Usuario) error {
	u.AtualizadoEm = time.Now().UTC()
	return r.db.WithContext(ctx).Save(u).Error
}

// Contar diz se já existe alguém cadastrado — o primeiro usuário do sistema
// nasce como gerência (mesma regra do modelo antigo).
func (r *Repo) Contar(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Usuario{}).Count(&n).Error
	return n, err
}

// --- sessões ---

func (r *Repo) CriarSessao(ctx context.Context, s *Sessao) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// UsuarioPorTokenHash devolve o dono de uma sessão viva. Junta usuário e
// sessão numa consulta só e já descarta sessão expirada e usuário inativo —
// desativar alguém mata o acesso na requisição seguinte, sem esperar o
// cookie vencer.
func (r *Repo) UsuarioPorTokenHash(ctx context.Context, hash []byte) (Usuario, Sessao, error) {
	var s Sessao
	if err := r.db.WithContext(ctx).
		Where("token_hash = ? AND expira_em > ?", hash, time.Now().UTC()).
		First(&s).Error; err != nil {
		return Usuario{}, Sessao{}, err
	}

	u, err := r.PorID(ctx, s.UsuarioID)
	if err != nil {
		return Usuario{}, Sessao{}, err
	}
	if !u.Ativo {
		return Usuario{}, Sessao{}, ErrNaoEncontrado
	}
	return u, s, nil
}

func (r *Repo) TocarSessao(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&Sessao{}).
		Where("id = ?", id).
		Update("ultimo_uso_em", time.Now().UTC()).Error
}

func (r *Repo) ApagarSessaoPorHash(ctx context.Context, hash []byte) error {
	return r.db.WithContext(ctx).Where("token_hash = ?", hash).Delete(&Sessao{}).Error
}

func (r *Repo) ApagarSessoesDoUsuario(ctx context.Context, usuarioID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("usuario_id = ?", usuarioID).Delete(&Sessao{}).Error
}

// LimparSessoesExpiradas roda no cron — sessão vencida não serve pra nada e
// só faz a tabela crescer.
func (r *Repo) LimparSessoesExpiradas(ctx context.Context) (int64, error) {
	res := r.db.WithContext(ctx).Where("expira_em < ?", time.Now().UTC()).Delete(&Sessao{})
	return res.RowsAffected, res.Error
}

// EhDuplicado reconhece a violação de índice único (e-mail já cadastrado)
// para o service devolver 409 em vez de 500.
func EhDuplicado(err error) bool { return errors.Is(err, gorm.ErrDuplicatedKey) }
