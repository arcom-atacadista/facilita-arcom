package disparo

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

// Worker roda a fila de disparo de minuto em minuto.
//
// É cron e não fila em Redis de propósito: a fila mora no Postgres (a tabela
// disparos), que é onde o registro precisa ficar de qualquer forma para
// auditoria. Acrescentar Redis só para agendar seria um serviço a mais
// rodando sem necessidade concreta — ver 05-regras-para-a-ia.md.
type Worker struct {
	svc  *Service
	cron *cron.Cron
	log  *slog.Logger
}

func NovoWorker(svc *Service, log *slog.Logger) *Worker {
	return &Worker{svc: svc, log: log, cron: cron.New(cron.WithLocation(time.UTC))}
}

// Iniciar agenda a rodada. Não bloqueia.
func (w *Worker) Iniciar() error {
	if _, err := w.cron.AddFunc("* * * * *", w.rodada); err != nil {
		return err
	}
	w.cron.Start()

	if w.svc.TemCanal() {
		w.log.Info("worker de disparo iniciado", "canal", w.svc.NomeDoCanal())
	} else {
		// Um aviso no boot, não a cada minuto: a operação precisa saber que a
		// fila enche e não escoa, sem inundar o log.
		w.log.Warn("worker de disparo iniciado SEM canal de envio — " +
			"as mensagens ficam na fila e nada é entregue. Ver internal/disparo/canal.go")
	}
	return nil
}

// Parar espera a rodada em andamento terminar — não corta um envio no meio.
func (w *Worker) Parar(ctx context.Context) {
	parado := w.cron.Stop()
	select {
	case <-parado.Done():
	case <-ctx.Done():
		w.log.Warn("worker de disparo não terminou a rodada a tempo do shutdown")
	}
}

func (w *Worker) rodada() {
	// Teto por rodada: uma rodada travada não pode se sobrepor à seguinte.
	ctx, cancelar := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancelar()

	enviados, err := w.svc.ProcessarFila(ctx)
	if err != nil {
		w.log.ErrorContext(ctx, "falha ao processar a fila de disparo", "erro", err)
		return
	}
	if enviados > 0 {
		w.log.InfoContext(ctx, "disparos entregues", "quantidade", enviados)
	}
}
