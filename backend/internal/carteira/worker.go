package carteira

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

// Worker roda a sincronização da carteira uma vez por dia.
//
// De madrugada de propósito: é quando o Gateway está menos disputado e quando
// ninguém está olhando a carteira mudar debaixo do próprio filtro. O horário
// é UTC — 6h aqui é 3h em Brasília.
const HorarioPadrao = "0 6 * * *"

type Worker struct {
	svc     *Service
	cron    *cron.Cron
	horario string
	log     *slog.Logger
}

func NovoWorker(svc *Service, horario string, log *slog.Logger) *Worker {
	if horario == "" {
		horario = HorarioPadrao
	}
	return &Worker{
		svc:     svc,
		horario: horario,
		log:     log,
		cron:    cron.New(cron.WithLocation(time.UTC)),
	}
}

func (w *Worker) Iniciar() error {
	if _, err := w.cron.AddFunc(w.horario, w.rodada); err != nil {
		return err
	}
	w.cron.Start()
	w.log.Info("sincronização da carteira agendada", "horario_utc", w.horario)
	return nil
}

func (w *Worker) Parar(ctx context.Context) {
	parado := w.cron.Stop()
	select {
	case <-parado.Done():
	case <-ctx.Done():
		w.log.Warn("a sincronização da carteira não terminou a tempo do desligamento")
	}
}

func (w *Worker) rodada() {
	// Teto generoso: uma carteira grande pode levar minutos, mas nunca deve
	// atravessar até a rodada do dia seguinte.
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancelar()

	res, err := w.svc.Sincronizar(ctx)
	if err != nil {
		w.log.ErrorContext(ctx, "sincronização da carteira falhou", "erro", err)
		return
	}
	w.log.InfoContext(ctx, "carteira sincronizada",
		"lidas", res.Lidas, "criadas", res.Criadas, "atualizadas", res.Atualizadas,
		"fechadas", res.Fechadas, "ignoradas", res.Ignoradas)
}
