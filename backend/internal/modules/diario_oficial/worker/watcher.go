package worker

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/infrastructure"
)

// maxIngestRetries limita quantas vezes uma edição FAILED é reprocessada
// automaticamente antes de o watcher desistir dela.
const maxIngestRetries = 3

var editionDigitsRegex = regexp.MustCompile(`\d+`)

// sanitizeEditionNumber normaliza o rótulo cru da edição ("Edição Nº 6265",
// "6265 (PDF)", "6263-E") para a chave usada em
// diario_oficial_editions.edition_number (UNIQUE): dígitos iniciais mais um
// sufixo S/E de suplementar, se houver. Retorna "" quando não há dígitos.
func sanitizeEditionNumber(raw string) string {
	s := strings.ToUpper(strings.TrimSpace(raw))
	digits := editionDigitsRegex.FindString(s)
	if digits == "" {
		return ""
	}
	rest := strings.TrimLeft(s[strings.Index(s, digits)+len(digits):], "-/ ")
	if len(rest) > 0 && (rest[0] == 'S' || rest[0] == 'E') {
		return digits + string(rest[0])
	}
	return digits
}

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

	// 1. DESCOBERTA no portal. Falha aqui (portal fora do ar / lento) NÃO
	// pode impedir o processamento das edições já conhecidas — elas têm
	// pdf_url próprio e não dependem do portal.
	now := time.Now()
	newCount := 0
	searchResult, err := w.client.Search(ctx, domain.SearchQuery{})
	if err != nil {
		if w.logger != nil {
			w.logger.Warn("Vigia: portal indisponível na descoberta; segue processando pendentes", "err", err)
		}
		searchResult = &domain.SearchResult{}
	}

	for _, item := range searchResult.Items {
		edNumber := sanitizeEditionNumber(item.TipoComunicacao)
		if edNumber == "" || edNumber == "0" {
			edNumber = sanitizeEditionNumber(fmt.Sprintf("%d", item.ExternalID))
		}
		if edNumber == "" || edNumber == "0" {
			continue
		}

		existing, err := w.repo.GetEditionByNumber(ctx, edNumber)
		if err != nil {
			continue
		}

		switch {
		case existing == nil:
			// Nova edição encontrada no portal. Data desconhecida (portal sem
			// data parseável) fica zero -> NULL no banco, nunca uma data
			// inventada.
			ed := &domain.Edition{
				EditionNumber: edNumber,
				EditionDate:   item.AvailabilityDate,
				PdfURL:        item.Link,
				Status:        domain.EditionStatusPending,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if err := w.repo.SaveEdition(ctx, ed); err == nil {
				newCount++
			}
		case existing.EditionDate.IsZero() && !item.AvailabilityDate.IsZero():
			// Já existia sem data; o portal agora tem uma — preenche
			// (SaveEdition faz COALESCE, não sobrescreve data já boa).
			existing.EditionDate = item.AvailabilityDate
			existing.PdfURL = item.Link
			_ = w.repo.SaveEdition(ctx, existing)
		}
	}

	if w.logger != nil {
		w.logger.Info("Vigia concluiu verificação", "novas_edicoes_encontradas", newCount)
	}

	// 2. Reencaminha edições que falharam (até o limite de tentativas) — sem
	// isso, uma edição FAILED (ex.: um PDF que produziu bytes inválidos e já
	// foi corrigido por gazette.RepairEncoding) ficaria presa para sempre.
	if requeued, rErr := w.repo.RequeueFailedEditions(ctx, maxIngestRetries); rErr == nil && requeued > 0 && w.logger != nil {
		w.logger.Info("Vigia reencaminhou edições FAILED para nova tentativa", "count", requeued)
	}

	// 3. Processa edições pendentes através do Worker Pool
	pending, err := w.repo.GetPendingEditions(ctx, 20)
	if err != nil {
		if w.logger != nil {
			w.logger.Error("Vigia: falha ao listar edições pendentes", "err", err)
		}
		return
	}
	if len(pending) > 0 {
		if w.logger != nil {
			w.logger.Info("Disparando Worker Pool para edições pendentes", "count", len(pending))
		}
		_ = w.workerPool.ProcessBatch(ctx, pending)
	}
}
