// Package db monta as conexões de infraestrutura (Postgres via gorm, Redis via
// go-redis). Fica isolado num pacote próprio porque é o único lugar do
// backend que conhece o driver — o resto do código depende só de *gorm.DB /
// *redis.Client.
package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// tentativasConexao/intervaloConexao toleram a janela transitória em que o
// Postgres está de pé mas ainda não aceita conexão — acontece de verdade na
// primeira subida com volume vazio: a imagem oficial sobe, roda os scripts de
// inicialização, desliga e sobe de novo pra valer, tudo em menos de 1
// segundo. Sem retry, um `cmd/migrate` (ou o próprio backend) que conecta uma
// vez só cai nessa janela com um erro transitório (Postgres "the database
// system is starting up") que se resolveria sozinho no segundo seguinte —
// incidente real que já aconteceu com esse exato padrão. 15 tentativas de 1s
// = até 15s de tolerância.
const (
	tentativasConexao = 15
	intervaloConexao  = time.Second
)

// EsperarPronto repete `tentar` até dar certo ou as tentativas acabarem — use
// antes de rodar migrations (cmd/migrate) ou de considerar uma conexão
// pronta. Ver comentário das constantes acima.
func EsperarPronto(tentar func() error) error {
	var ultimoErro error
	for tentativa := 1; tentativa <= tentativasConexao; tentativa++ {
		err := tentar()
		if err == nil {
			return nil
		}
		if ultimoErro == nil { // primeira falha: aviso único, não spam a cada tentativa
			slog.Warn("postgres ainda não aceita conexão, tentando de novo", "erro", err, "tentativas_restantes", tentativasConexao-tentativa)
		}
		ultimoErro = err
		if tentativa < tentativasConexao {
			time.Sleep(intervaloConexao)
		}
	}
	return ultimoErro
}

// AbrirPostgres conecta no Postgres. DatabaseURL vazia retorna (nil, nil) —
// projeto sem Postgres (padroes/07-frontend-primeiro.md) não deve falhar por
// isso.
func AbrirPostgres(databaseURL string) (*gorm.DB, error) {
	if databaseURL == "" {
		return nil, nil
	}

	var gdb *gorm.DB
	err := EsperarPronto(func() error {
		var err error
		gdb, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{
			Logger:         gormlogger.Default.LogMode(gormlogger.Warn),
			NowFunc:        func() time.Time { return time.Now().UTC() }, // datas sempre em UTC (ver 09-contrato-api.md)
			TranslateError: true,
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("conectar no postgres: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("obter *sql.DB do gorm: %w", err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return gdb, nil
}

// AbrirRedis conecta no Redis. RedisURL vazia retorna (nil, nil) pelo mesmo
// motivo do Postgres. Sem EsperarPronto de propósito: a imagem oficial do
// Redis não tem a dança de "sobe, roda init, desliga, sobe de novo" que a do
// Postgres tem no primeiro start — não existe a mesma janela transitória pra
// tolerar.
func AbrirRedis(redisURL string) (*redis.Client, error) {
	if redisURL == "" {
		return nil, nil
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("REDIS_URL inválida: %w", err)
	}

	cliente := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cliente.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("conectar no redis: %w", err)
	}

	return cliente, nil
}

// Pronto verifica se as conexões configuradas respondem — usado pelo
// readiness (GET /api/ready), nunca pelo liveness. Timeout curto: se o banco
// está lento, readiness falha rápido em vez de travar a checagem.
func Pronto(ctx context.Context, gdb *gorm.DB, rdb *redis.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if gdb != nil {
		sqlDB, err := gdb.DB()
		if err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
	}

	if rdb != nil {
		if err := rdb.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis: %w", err)
		}
	}

	return nil
}
