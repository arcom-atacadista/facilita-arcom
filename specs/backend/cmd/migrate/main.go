// Comando separado que aplica as migrations do Postgres. Roda como um passo
// próprio no docker-compose (serviço "migrate", ver padroes/04-docker-deploy.md)
// antes do backend subir — nunca dentro do boot do servidor, porque múltiplas
// réplicas subindo juntas correriam pra aplicar migration ao mesmo tempo.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib" // registra o driver "pgx" pro database/sql

	"meu-projeto/internal/config"
	"meu-projeto/internal/db"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("configuração inválida", "erro", err)
		os.Exit(1)
	}

	if cfg.DatabaseURL == "" {
		log.Info("DATABASE_URL não configurada — projeto sem Postgres, nada a migrar")
		return
	}

	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Error("abrir conexão com o postgres", "erro", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	// Tolera a janela transitória em que o Postgres está de pé mas ainda não
	// aceita conexão (acontece de verdade na primeira subida com volume vazio
	// — ver comentário de EsperarPronto em internal/db/db.go). Sem isso, essa
	// janela derruba o migrate com um erro que se resolveria sozinho.
	if err := db.EsperarPronto(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return sqlDB.PingContext(ctx)
	}); err != nil {
		log.Error("postgres não respondeu ao ping", "erro", err)
		os.Exit(1)
	}

	goose.SetBaseFS(db.Migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Error("configurar dialeto do goose", "erro", err)
		os.Exit(1)
	}

	if err := goose.Up(sqlDB, "migrations"); err != nil {
		log.Error("aplicar migrations", "erro", err)
		os.Exit(1)
	}

	log.Info("migrations aplicadas com sucesso")
}
