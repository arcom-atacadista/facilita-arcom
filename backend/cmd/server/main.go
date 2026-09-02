// Entrada do backend. Monta as dependências, sobe um http.Server com timeouts
// e desliga de forma graciosa quando recebe SIGTERM/SIGINT — sem isso, um
// `docker compose stop`/deploy novo corta requisição no meio.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"facilitaarcom/internal/config"
	"facilitaarcom/internal/db"
	"facilitaarcom/internal/servidor"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := executar(log); err != nil {
		log.Error("servidor encerrado com erro", "erro", err)
		os.Exit(1)
	}
}

func executar(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	gdb, err := db.AbrirPostgres(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	rdb, err := db.AbrirRedis(cfg.RedisURL)
	if err != nil {
		return err
	}

	handler := servidor.Novo(cfg, log, gdb, rdb)

	srv := &http.Server{
		Addr:    "0.0.0.0:" + cfg.Port,
		Handler: handler,
		// Timeouts contra slowloris e cliente lento — sem isso uma conexão
		// parada em aberto consome uma goroutine pra sempre.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	erros := make(chan error, 1)
	go func() {
		log.Info("subindo servidor", "port", cfg.Port, "ambiente", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erros <- err
			return
		}
		erros <- nil
	}()

	select {
	case err := <-erros:
		return err
	case <-ctx.Done():
	}

	log.Info("sinal de encerramento recebido, drenando requisições em andamento")
	desligarCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(desligarCtx); err != nil {
		return err
	}
	log.Info("servidor encerrado com sucesso")
	return nil
}
