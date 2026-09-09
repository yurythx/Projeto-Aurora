package app

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/yurythx/projeto-aurora/internal/platform/audit"
	"github.com/yurythx/projeto-aurora/internal/platform/httpserver"
	"github.com/yurythx/projeto-aurora/internal/platform/idempotency"
	"github.com/yurythx/projeto-aurora/internal/platform/jobs"
	"github.com/yurythx/projeto-aurora/internal/platform/lgpd"
	"github.com/yurythx/projeto-aurora/internal/platform/outbox"
	"github.com/yurythx/projeto-aurora/internal/platform/ratelimit"
)

// Worker roda os processadores de segundo plano da infraestrutura (publisher do outbox,
// limpezas periódicas de rate limit e idempotência, sweepers de jobs).
type Worker struct {
	deps       *Dependencies
	processors []processor
}

type processor func(ctx context.Context) error

// NewWorker constrói o runner do worker para o Projeto Aurora.
func NewWorker(deps *Dependencies) (*Worker, error) {
	outboxPublisher := outbox.NewPublisher(deps.DB, deps.Publisher, deps.Logger)

	staleHandlers := map[string]jobs.StaleJobHandler{}

	return &Worker{
		deps: deps,
		processors: []processor{
			supervised("outbox_publisher", deps.Logger, outboxPublisher.Run),
			supervised("rate_limit_cleanup", deps.Logger, ratelimit.Cleanup(deps.DB)),
			supervised("idempotency_cleanup", deps.Logger, idempotency.Cleanup(deps.DB)),
			supervised("jobs_stale_sweeper", deps.Logger, jobs.SweepStale(deps.DB, staleHandlers, deps.Config.Jobs.StaleAfter, deps.Logger)),
			// LGPD art. 18, VI — processa as solicitações de exclusão
			// (anonimização) do titular (F3.2).
			supervised("lgpd_erasure", deps.Logger, lgpd.ErasureProcessor(deps.DB, deps.Logger)),
			// Cópia WORM diária da trilha de auditoria para o object
			// storage — bucket DEDICADO com object-lock quando disponível,
			// cadeia de SHA-256 sempre (F2.6).
			supervised("audit_worm_export", deps.Logger, audit.WORMExporter(deps.DB, deps.Storage, deps.Config.AuditWORM.Bucket, deps.Config.AuditWORM.RetentionDays, deps.Logger)),
		},
	}, nil
}

func supervised(name string, logger *slog.Logger, fn processor) processor {
	return func(ctx context.Context) error {
		backoff := time.Second
		const maxBackoff = 30 * time.Second

		for {
			err := fn(ctx)
			if ctx.Err() != nil {
				return nil
			}
			if err != nil {
				logger.Error("processor exited unexpectedly, restarting", slog.String("processor", name), slog.Any("error", err), slog.Duration("retry_in", backoff))
			} else {
				logger.Warn("processor returned without error before shutdown, restarting", slog.String("processor", name))
			}

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
			}
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
		}
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if len(w.processors) == 0 {
		<-ctx.Done()
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(w.processors))

	for _, p := range w.processors {
		wg.Add(1)
		go func(p processor) {
			defer wg.Done()
			if err := p(ctx); err != nil {
				errCh <- err
			}
		}(p)
	}

	<-ctx.Done()
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) RunMetricsServer(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/health", httpserver.HealthHandler())
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:              w.deps.Config.Worker.MetricsAddr(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		w.deps.Logger.Info("worker metrics listener starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
