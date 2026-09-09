// Package carteira mantém a carteira em atraso espelhando o dataset `debitos`
// do Gateway ARCOM.
//
// Antes desta etapa, cliente e dívida precisavam ser semeados à mão. Agora
// uma rodada diária cria o que entrou na faixa de 3 a 90 dias, atualiza o que
// mudou e fecha o que saiu de lá — porque dívida que some do Gateway foi
// quase certamente paga, e continuar cobrando quem pagou é pior do que
// deixar de cobrar quem deve.
package carteira

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"facilitaarcom/internal/cobranca"
	"facilitaarcom/internal/gatewayarcom"
)

const (
	// LoteDoGateway é quantos débitos pedir por chamada.
	LoteDoGateway = 500

	// MaxPaginas é o teto de segurança do laço de paginação: sem ele, um
	// cursor que não avança viraria laço infinito consumindo a API interna.
	MaxPaginas = 200

	// AtrasoMinimo e AtrasoMaximo são a régua da operação. Fora disso a
	// dívida não é assunto desta plataforma.
	AtrasoMinimo = 3
	AtrasoMaximo = 90
)

// Resultado é o que a rodada fez, e vira registro na tabela sincronizacoes.
type Resultado struct {
	Lidas       int
	Criadas     int
	Atualizadas int
	Fechadas    int
	Ignoradas   int
}

type Service struct {
	db         *gorm.DB
	gateway    *gatewayarcom.Cliente
	mapeamento gatewayarcom.Mapeamento
	log        *slog.Logger
	agora      func() time.Time
}

func NovoService(db *gorm.DB, gateway *gatewayarcom.Cliente, mapeamento gatewayarcom.Mapeamento, log *slog.Logger) *Service {
	return &Service{
		db: db, gateway: gateway, mapeamento: mapeamento, log: log,
		agora: func() time.Time { return time.Now().UTC() },
	}
}

var ErrSemGateway = errors.New("sincronização exige a credencial do Gateway ARCOM")

// Sincronizar roda uma rodada completa e devolve o que foi feito.
func (s *Service) Sincronizar(ctx context.Context) (Resultado, error) {
	if s.gateway == nil || !s.gateway.TemCredencial() {
		return Resultado{}, ErrSemGateway
	}

	inicio := s.agora()
	idRodada := uuid.New()
	if err := s.abrirRegistro(ctx, idRodada, inicio); err != nil {
		return Resultado{}, err
	}

	res, err := s.rodar(ctx, inicio)
	if err != nil {
		s.fecharRegistro(ctx, idRodada, res, err)
		return res, err
	}

	s.fecharRegistro(ctx, idRodada, res, nil)
	return res, nil
}

func (s *Service) rodar(ctx context.Context, inicio time.Time) (Resultado, error) {
	var res Resultado

	debitos, err := s.gateway.BuscarDebitos(ctx, "", LoteDoGateway)
	if err != nil {
		return res, fmt.Errorf("buscar débitos: %w", err)
	}
	res.Lidas = len(debitos)

	// Guarda contra rodada vazia: se o Gateway devolve zero linhas por
	// instabilidade e a gente fechasse tudo em seguida, a carteira inteira
	// sumiria de uma vez. Zero linhas nunca fecha nada.
	if len(debitos) == 0 {
		s.log.WarnContext(ctx, "o Gateway não devolveu nenhum débito — nada foi criado nem fechado")
		return res, nil
	}

	for _, d := range debitos {
		linha, err := gatewayarcom.Mapear(d, s.mapeamento)
		if err != nil {
			// Linha incompleta é registrada e pulada: uma dívida sem
			// vencimento não vira régua, e derrubar a rodada inteira por
			// causa dela seria pior.
			res.Ignoradas++
			s.log.DebugContext(ctx, "débito ignorado na sincronização", "erro", err)
			continue
		}

		venc, err := time.Parse(cobranca.FormatoData, linha.Vencimento)
		if err != nil {
			res.Ignoradas++
			continue
		}

		// A faixa é o recorte desta plataforma. O que está fora não entra —
		// nem o que ainda não venceu, nem o que já foi para o jurídico.
		dias := cobranca.DiasAtraso(venc, inicio)
		if dias < AtrasoMinimo || dias > AtrasoMaximo {
			res.Ignoradas++
			continue
		}

		criada, err := s.gravarLinha(ctx, linha, venc, inicio)
		if err != nil {
			return res, err
		}
		if criada {
			res.Criadas++
		} else {
			res.Atualizadas++
		}
	}

	// Sem telefone não há para onde disparar, e o dataset `debitos` não traz
	// nenhum. Esta etapa completa o cadastro a partir do histórico da Nines,
	// que é o único lugar do Gateway com o número do devedor.
	if err := s.completarTelefones(ctx); err != nil {
		// Não derruba a rodada: a carteira já entrou e vale por si. Sem
		// telefone o disparo é recusado com mensagem clara, na hora.
		s.log.WarnContext(ctx, "não consegui completar os telefones pelo histórico da Nines", "erro", err)
	}

	fechadas, err := s.fecharAusentes(ctx, inicio)
	if err != nil {
		return res, err
	}
	res.Fechadas = int(fechadas)

	return res, nil
}

// gravarLinha cria ou atualiza cliente e dívida. Devolve true quando a dívida
// é nova.
func (s *Service) gravarLinha(ctx context.Context, l gatewayarcom.LinhaCarteira, venc, agora time.Time) (bool, error) {
	var nova bool

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// O cliente é identificado pelo documento, que é único.
		cliente := cobranca.Cliente{
			ID:           uuid.New(),
			Nome:         l.NomeCliente,
			Documento:    l.Documento,
			CriadoEm:     agora,
			AtualizadoEm: agora,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "documento"}},
			DoUpdates: clause.Assignments(map[string]any{
				"nome":          l.NomeCliente,
				"atualizado_em": agora,
			}),
		}).Create(&cliente).Error; err != nil {
			return err
		}

		// O Create com ON CONFLICT não devolve o id da linha existente.
		var clienteID string
		if err := tx.Table("clientes").Select("id::text").
			Where("documento = ?", l.Documento).Limit(1).Scan(&clienteID).Error; err != nil {
			return err
		}
		id, err := uuid.Parse(clienteID)
		if err != nil {
			return fmt.Errorf("id de cliente inválido no banco: %w", err)
		}

		var responsavel, filial *string
		if l.ResponsavelCobranca != "" {
			r := l.ResponsavelCobranca
			responsavel = &r
		}
		if l.Filial != "" {
			f := l.Filial
			filial = &f
		}

		// ON CONFLICT DO UPDATE também reporta uma linha afetada, então
		// RowsAffected não distingue inserção de atualização. Sem esta
		// consulta, toda dívida entraria no relatório como "criada" e o
		// número da rodada mentiria.
		var existentes int64
		if err := tx.Model(&cobranca.Divida{}).
			Where("cliente_id = ? AND contrato = ?", id, l.Contrato).
			Count(&existentes).Error; err != nil {
			return err
		}
		nova = existentes == 0

		divida := cobranca.Divida{
			ID:                  uuid.New(),
			ClienteID:           id,
			Contrato:            l.Contrato,
			ValorOriginal:       l.Valor,
			ValorEncargos:       l.Encargos,
			Vencimento:          venc,
			Status:              cobranca.StatusDividaAberta,
			ResponsavelCobranca: responsavel,
			Filial:              filial,
			CriadoEm:            agora,
			AtualizadoEm:        agora,
		}

		erroUpsert := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "cliente_id"}, {Name: "contrato"}},
			DoUpdates: clause.Assignments(map[string]any{
				"valor_original":       l.Valor,
				"valor_encargos":       l.Encargos,
				"vencimento":           venc,
				"responsavel_cobranca": responsavel,
				"filial":               filial,
				"sincronizado_em":      agora,
				"atualizado_em":        agora,
				// status NÃO entra: uma dívida já negociada não pode voltar a
				// "aberto" só porque continua aparecendo no Gateway até a
				// primeira parcela compensar.
			}),
		}).Create(&divida).Error
		if erroUpsert != nil {
			return erroUpsert
		}

		// Marca origem e visto-por-último também na criação.
		return tx.Model(&cobranca.Divida{}).
			Where("cliente_id = ? AND contrato = ?", id, l.Contrato).
			Updates(map[string]any{"origem": "gateway", "sincronizado_em": agora}).Error
	})

	return nova, err
}

// fecharAusentes marca como quitada a dívida que veio do Gateway, está aberta
// e não apareceu nesta rodada.
//
// Só toca no que tem origem 'gateway': dívida lançada à mão por um operador
// não some por não estar lá.
func (s *Service) fecharAusentes(ctx context.Context, inicio time.Time) (int64, error) {
	res := s.db.WithContext(ctx).Model(&cobranca.Divida{}).
		Where("origem = ? AND status = ? AND (sincronizado_em IS NULL OR sincronizado_em < ?)",
			"gateway", cobranca.StatusDividaAberta, inicio).
		Updates(map[string]any{
			"status":        cobranca.StatusDividaQuitada,
			"atualizado_em": s.agora(),
		})
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected > 0 {
		s.log.InfoContext(ctx, "dívidas fechadas por terem saído do Gateway", "quantidade", res.RowsAffected)
	}
	return res.RowsAffected, nil
}

// LoteDeTelefones é quantos disparos do histórico da Nines ler para
// reconstruir a base de telefones.
const LoteDeTelefones = 2000

// completarTelefones preenche o telefone de quem está sem, usando o número
// para onde a Nines já mandou mensagem.
//
// Só preenche o que está vazio: número corrigido à mão por um operador não
// pode ser sobrescrito por um registro antigo do histórico.
func (s *Service) completarTelefones(ctx context.Context) error {
	var faltando []struct {
		ID        string
		Documento string
	}
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id::text AS id, documento FROM clientes
		WHERE telefone IS NULL OR telefone = ''`).Scan(&faltando).Error; err != nil {
		return err
	}
	if len(faltando) == 0 {
		return nil
	}

	telefones, err := s.gateway.BuscarTelefonesDaNines(ctx, LoteDeTelefones)
	if err != nil {
		return err
	}
	if len(telefones) == 0 {
		return nil
	}

	preenchidos := 0
	for _, c := range faltando {
		tel, achou := telefones[c.Documento]
		if !achou {
			continue
		}
		if err := s.db.WithContext(ctx).Exec(
			`UPDATE clientes SET telefone = ?, atualizado_em = ? WHERE id = ?::uuid AND (telefone IS NULL OR telefone = '')`,
			tel, s.agora(), c.ID).Error; err != nil {
			return err
		}
		preenchidos++
	}

	if preenchidos > 0 {
		s.log.InfoContext(ctx, "telefones completados pelo histórico da Nines",
			"preenchidos", preenchidos, "sem_telefone", len(faltando)-preenchidos)
	}
	return nil
}

// --- registro das rodadas ---

func (s *Service) abrirRegistro(ctx context.Context, id uuid.UUID, inicio time.Time) error {
	return s.db.WithContext(ctx).Exec(
		`INSERT INTO sincronizacoes (id, iniciada_em, status) VALUES (?, ?, 'rodando')`,
		id, inicio).Error
}

func (s *Service) fecharRegistro(ctx context.Context, id uuid.UUID, r Resultado, falha error) {
	status, detalhe := "concluida", any(nil)
	if falha != nil {
		// A mensagem do erro fica no registro para a operação enxergar sem
		// abrir log de servidor. Erro do Gateway já vem sem corpo de resposta.
		status, detalhe = "falhou", falha.Error()
	}

	if err := s.db.WithContext(ctx).Exec(`
		UPDATE sincronizacoes
		SET terminada_em = ?, status = ?, lidas = ?, criadas = ?, atualizadas = ?, fechadas = ?, ignoradas = ?, erro_detalhe = ?
		WHERE id = ?`,
		s.agora(), status, r.Lidas, r.Criadas, r.Atualizadas, r.Fechadas, r.Ignoradas, detalhe, id,
	).Error; err != nil {
		s.log.ErrorContext(ctx, "falha ao registrar a sincronização", "erro", err)
	}
}

// UltimaSincronizacao alimenta a tela: "a carteira de hoje entrou?".
type UltimaSincronizacao struct {
	IniciadaEm  time.Time  `json:"iniciadaEm"`
	TerminadaEm *time.Time `json:"terminadaEm"`
	Status      string     `json:"status"`
	Lidas       int        `json:"lidas"`
	Criadas     int        `json:"criadas"`
	Atualizadas int        `json:"atualizadas"`
	Fechadas    int        `json:"fechadas"`
	Ignoradas   int        `json:"ignoradas"`
	ErroDetalhe *string    `json:"erroDetalhe"`
}

func (s *Service) Ultima(ctx context.Context) (*UltimaSincronizacao, error) {
	var u UltimaSincronizacao
	err := s.db.WithContext(ctx).Raw(`
		SELECT iniciada_em, terminada_em, status, lidas, criadas, atualizadas, fechadas, ignoradas, erro_detalhe
		FROM sincronizacoes ORDER BY iniciada_em DESC LIMIT 1`).Scan(&u).Error
	if err != nil {
		return nil, err
	}
	if u.Status == "" {
		return nil, nil
	}
	return &u, nil
}
