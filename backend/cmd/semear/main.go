// Comando de desenvolvimento que cria uma carteira fictícia para demonstrar o
// sistema antes de existir a chave do Gateway ARCOM.
//
// POR QUE ISTO EXISTE
//
// A carteira do Facilita ARCOM é espelho do Gateway: dívida entra pela
// sincronização (internal/carteira) e não há endpoint para criar dívida à mão,
// de propósito — inventar devedor pela API seria um caminho de escrita numa
// base que deve refletir o sistema de origem. Só que isso deixa a demonstração
// sem nada para disparar enquanto a chave do Gateway não é provisionada.
//
// Este comando preenche essa lacuna, e só ela: dados obviamente fictícios, num
// banco de desenvolvimento, apontando todos para um único telefone de teste.
//
// TRÊS TRAVAS, porque o estrago aqui seria mandar mensagem para gente real:
//
//  1. Recusa rodar com APP_ENV=production.
//  2. Recusa rodar se já existe dívida vinda do Gateway — carteira real não é
//     lugar de dado de brinquedo.
//  3. Exige TELEFONE_DEMO. Todo devedor fictício recebe esse mesmo número, que
//     é o seu; assim é impossível o teste alcançar um cliente de verdade.
//
// As dívidas entram com origem='manual'. A sincronização diária só encerra o
// que veio com origem='gateway', então a carteira de teste não é apagada
// quando o Gateway entrar no ar — some quando você derrubar o banco local.
//
//	DATABASE_URL=postgres://app@localhost:5432/app \
//	  TELEFONE_DEMO=34999998888 go run ./cmd/semear
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"

	_ "github.com/jackc/pgx/v5/stdlib" // registra o driver "pgx" pro database/sql

	"facilitaarcom/internal/config"
	"facilitaarcom/internal/disparo"
)

// CodigoCobrancaDemo é o responsável atribuído às dívidas fictícias. Para um
// analista enxergar a carteira de teste, o codigo_cobranca dele precisa ser
// este; coordenação pra cima vê tudo por papel e não precisa de nada.
const CodigoCobrancaDemo = "DEMO"

// devedor é uma linha da carteira fictícia. diasDeAtraso é relativo a hoje,
// para a dívida cair sempre na mesma faixa da régua independente de quando o
// comando roda.
type devedor struct {
	nome          string
	documento     string
	contrato      string
	valor         float64
	diasDeAtraso  int
	oQueDemonstra string
}

// A carteira cobre as três faixas da régua (3-30, 31-60, 61-90) e mistura CPF
// com CNPJ de propósito: é o que mostra o NomeDeTratamento funcionando.
var carteira = []devedor{
	{
		nome: "Mercado do João LTDA", documento: "11222333000181",
		contrato: "CTR-2026-0413", valor: 4820.50, diasDeAtraso: 12,
		oQueDemonstra: "PJ na faixa inicial — a mensagem chama de \"Mercado do João\", não de \"Mercado\"",
	},
	{
		nome: "Maria Aparecida da Silva", documento: "11122233396",
		contrato: "CTR-2026-0388", valor: 1240.00, diasDeAtraso: 27,
		oQueDemonstra: "PF na faixa inicial — a mensagem chama de \"Maria\"",
	},
	{
		nome: "Distribuidora Santa Rita ME", documento: "44555666000172",
		contrato: "CTR-2026-0291", valor: 18750.90, diasDeAtraso: 44,
		oQueDemonstra: "faixa intermediária, valor alto",
	},
	{
		nome: "José Carlos Ferreira", documento: "44455566677",
		contrato: "CTR-2026-0305", valor: 640.00, diasDeAtraso: 51,
		oQueDemonstra: "faixa intermediária, valor baixo — testa o mínimo de parcela",
	},
	{
		nome: "Padaria Pão Quente EIRELI", documento: "77888999000163",
		contrato: "CTR-2025-1180", valor: 9310.25, diasDeAtraso: 73,
		oQueDemonstra: "faixa final",
	},
	{
		// Foi este valor que expôs o arredondamento perdendo um centavo.
		nome: "Comercial Três Irmãos S/A", documento: "99000111000154",
		contrato: "CTR-2025-0940", valor: 1234567.89, diasDeAtraso: 88,
		oQueDemonstra: "faixa final, valor que já quebrou o arredondamento uma vez",
	},
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := executar(context.Background(), log); err != nil {
		log.Error("semear", "erro", err)
		os.Exit(1)
	}
}

func executar(ctx context.Context, log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuração inválida: %w", err)
	}

	// Trava 1: produção não recebe dado fictício, em hipótese nenhuma.
	if cfg.Env == "production" {
		return errors.New("APP_ENV=production — este comando é só de desenvolvimento")
	}
	if cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL não configurada")
	}

	// Trava 3: sem telefone de teste não há como garantir que a mensagem não
	// alcance um cliente real, então o comando simplesmente não roda.
	telefone := disparo.TelefoneComDDI(os.Getenv("TELEFONE_DEMO"))
	if !disparo.TelefoneValido(telefone) {
		return errors.New("defina TELEFONE_DEMO com o seu número (ex.: 34999998888) — " +
			"todo devedor fictício recebe esse número, e é isso que impede o teste de alcançar cliente real")
	}

	bd, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("abrir conexão: %w", err)
	}
	defer bd.Close()

	if err := bd.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres não respondeu: %w", err)
	}

	// Trava 2: se o Gateway já alimentou esta base, ela é carteira de verdade.
	var doGateway int
	if err := bd.QueryRowContext(ctx,
		`SELECT count(*) FROM dividas WHERE origem = 'gateway'`).Scan(&doGateway); err != nil {
		return fmt.Errorf("verificar origem das dívidas: %w", err)
	}
	if doGateway > 0 {
		return fmt.Errorf("este banco já tem %d dívida(s) vindas do Gateway — "+
			"é carteira real, não vou misturar dado fictício", doGateway)
	}

	hoje := time.Now().UTC().Truncate(24 * time.Hour)

	for _, d := range carteira {
		clienteID, err := upsertCliente(ctx, bd, d, telefone)
		if err != nil {
			return fmt.Errorf("cliente %s: %w", d.nome, err)
		}

		vencimento := hoje.AddDate(0, 0, -d.diasDeAtraso)
		if err := upsertDivida(ctx, bd, clienteID, d, vencimento); err != nil {
			return fmt.Errorf("dívida %s: %w", d.contrato, err)
		}

		log.Info("devedor pronto",
			"nome", d.nome,
			"contrato", d.contrato,
			"dias_de_atraso", d.diasDeAtraso,
			"demonstra", d.oQueDemonstra)
	}

	log.Info("carteira de exemplo criada",
		"devedores", len(carteira),
		"telefone_de_todos", telefone,
		"responsavel_cobranca", CodigoCobrancaDemo)
	log.Info("para um analista ver esta carteira, o codigo_cobranca dele precisa ser " +
		CodigoCobrancaDemo + "; conta de coordenação ou gerência já vê tudo")

	return nil
}

// upsertCliente devolve o id do cliente, criando-o se ainda não existe. O
// documento é a chave natural — rodar o comando duas vezes não duplica nada.
func upsertCliente(ctx context.Context, bd *sql.DB, d devedor, telefone string) (uuid.UUID, error) {
	var id uuid.UUID
	err := bd.QueryRowContext(ctx, `
		INSERT INTO clientes (id, nome, documento, telefone)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (documento) DO UPDATE
		  SET nome = EXCLUDED.nome, telefone = EXCLUDED.telefone, atualizado_em = now()
		RETURNING id::text
	`, uuid.New(), d.nome, d.documento, telefone).Scan(&id)
	return id, err
}

// upsertDivida cria a dívida em aberto. status fica de fora do UPDATE de
// propósito: se você já negociou essa dívida na demonstração, rodar o comando
// de novo não deve reabrir o acordo — é a mesma regra da sincronização real.
func upsertDivida(ctx context.Context, bd *sql.DB, clienteID uuid.UUID, d devedor, vencimento time.Time) error {
	_, err := bd.ExecContext(ctx, `
		INSERT INTO dividas
		  (id, cliente_id, contrato, valor_original, vencimento, status, responsavel_cobranca, origem)
		VALUES ($1, $2, $3, $4, $5, 'aberto', $6, 'manual')
		ON CONFLICT (cliente_id, contrato) DO UPDATE
		  SET valor_original = EXCLUDED.valor_original,
		      vencimento     = EXCLUDED.vencimento,
		      atualizado_em  = now()
	`, uuid.New(), clienteID, d.contrato, d.valor, vencimento, CodigoCobrancaDemo)
	return err
}
