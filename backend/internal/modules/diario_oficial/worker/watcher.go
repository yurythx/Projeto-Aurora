package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/infrastructure"
)

type Watcher struct {
	repo       domain.EditionRepository
	client     *infrastructure.RondonopolisClient
	workerPool *SyncWorkerPool
	interval   time.Duration
	logger     *slog.Logger
}

func NewWatcher(
	repo domain.EditionRepository,
	client *infrastructure.RondonopolisClient,
	workerPool *SyncWorkerPool,
	interval time.Duration,
	logger *slog.Logger,
) *Watcher {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	return &Watcher{
		repo:       repo,
		client:     client,
		workerPool: workerPool,
		interval:   interval,
		logger:     logger,
	}
}

// Start inicia o loop continuo do vigia em background.
func (w *Watcher) Start(ctx context.Context) {
	if w.logger != nil {
		w.logger.Info("Iniciando Vigia do Diário Oficial de Rondonópolis", "interval", w.interval)
	}

	// Executa uma sincronização inicial imediata ao arrancar
	w.sync(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if w.logger != nil {
				w.logger.Info("Encerrando Vigia do Diário Oficial")
			}
			return
		case <-ticker.C:
			w.sync(ctx)
		}
	}
}

func (w *Watcher) sync(ctx context.Context) {
	if w.logger != nil {
		w.logger.Info("Vigia verificando edições recém-publicadas no portal oficial...")
	}

	// 1. Consulta edições no portal oficial via Search do Client
	searchResult, err := w.client.Search(ctx, domain.SearchQuery{})
	if err != nil {
		if w.logger != nil {
			w.logger.Error("Vigia falhou ao consultar catálogo de edições", "err", err)
		}
		return
	}

	now := time.Now()
	newCount := 0

	for _, item := range searchResult.Items {
		edNumber := strings.TrimSpace(strings.TrimPrefix(item.TipoComunicacao, "Edição Nº "))
		if edNumber == "" || edNumber == "0" {
			edNumber = fmt.Sprintf("%d", item.ExternalID)
		}
		if edNumber == "" || edNumber == "0" {
			continue
		}

		existing, err := w.repo.GetEditionByNumber(ctx, edNumber)
		if err != nil {
			continue
		}

		if existing == nil {
			// Nova edição encontrada no portal!
			ed := &domain.Edition{
				EditionNumber: edNumber,
				EditionDate:   item.AvailabilityDate,
				PdfURL:        item.Link,
				Status:        domain.EditionStatusPending,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if ed.EditionDate.IsZero() {
				ed.EditionDate = now
			}
			if err := w.repo.SaveEdition(ctx, ed); err == nil {
				newCount++
			}
		}
	}

	if w.logger != nil {
		w.logger.Info("Vigia concluiu verificação", "novas_edicoes_encontradas", newCount)
	}

	// 2. Processa edições pendentes através do Worker Pool
	pending, err := w.repo.GetPendingEditions(ctx, 20)
	if err == nil && len(pending) > 0 {
		if w.logger != nil {
			w.logger.Info("Disparando Worker Pool para edições pendentes", "count", len(pending))
		}
		_ = w.workerPool.ProcessBatch(ctx, pending)
	}
}
