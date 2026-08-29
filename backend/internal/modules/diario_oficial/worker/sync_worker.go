package worker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/application"
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
	"github.com/yurythx/projeto-nova/internal/platform/metrics"
	"github.com/yurythx/projeto-nova/pkg/typesense"
)

type EditionJob struct {
	Edition domain.Edition
}

// SyncWorkerPool processa edições do Diário Oficial em paralelo:
// 1. Faz download em stream do PDF (sem disco)
// 2. Extrai texto via pdftotext
// 3. Parseia atos de pessoal e contratos (gazette.ParsePersonnelActs)
// 4. Persiste findings atomicamente no PostgreSQL
// 5. Indexa atos de pessoal no Typesense (best-effort — nunca bloqueia a ingestão principal)
type SyncWorkerPool struct {
	repo        domain.EditionRepository
	parser      *application.PDFParser
	tsClient    *typesense.Client // nil quando Typesense não está configurado
	reindexer   *Reindexer        // nil quando Typesense não está configurado
	numWorkers  int
	reqInterval time.Duration
	httpClient  *http.Client
	logger      *slog.Logger
}

func NewSyncWorkerPool(repo domain.EditionRepository, numWorkers int, logger *slog.Logger) *SyncWorkerPool {
	if numWorkers <= 0 {
		numWorkers = 5
	}
	return &SyncWorkerPool{
		repo:        repo,
		parser:      application.NewPDFParser(),
		numWorkers:  numWorkers,
		reqInterval: 200 * time.Millisecond, // rate limiter: 5 req/s
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		logger:      logger,
	}
}

// WithTypesenseClient injeta o cliente Typesense para indexação de atos de pessoal.
// Segue o mesmo padrão de injeção opcional que Service.WithRondonopolisClient usa —
// nada explode se nil, só a indexação é pulada.
func (p *SyncWorkerPool) WithTypesenseClient(ts *typesense.Client) *SyncWorkerPool {
	p.tsClient = ts
	if ts != nil {
		p.reindexer = NewReindexer(p.repo, ts, p.logger)
	}
	return p
}

// ProcessBatch executa o processamento concorrente do Worker Pool para a lista de edições passadas.
func (p *SyncWorkerPool) ProcessBatch(ctx context.Context, editions []domain.Edition) error {
	if len(editions) == 0 {
		return nil
	}

	jobsChan := make(chan EditionJob, len(editions))
	for _, ed := range editions {
		jobsChan <- EditionJob{Edition: ed}
	}
	close(jobsChan)

	limiter := time.NewTicker(p.reqInterval)
	defer limiter.Stop()

	var wg sync.WaitGroup
	for w := 1; w <= p.numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for job := range jobsChan {
				select {
				case <-ctx.Done():
					return
				case <-limiter.C:
					p.processJob(ctx, job)
				}
			}
		}(w)
	}

	wg.Wait()
	return nil
}

// failEdition marca a edição como FAILED, registra a métrica e, quando já
// esgotou as tentativas, contabiliza como abandonada.
func (p *SyncWorkerPool) failEdition(ctx context.Context, ed domain.Edition, reason string) {
	_ = p.repo.UpdateEditionStatus(ctx, ed.ID, domain.EditionStatusFailed, 0, &reason)
	metrics.DiarioEditionsIngestedTotal.WithLabelValues("failed").Inc()
	if ed.RetryCount+1 >= maxIngestRetries {
		metrics.DiarioEditionAbandonedTotal.Inc()
		if p.logger != nil {
			p.logger.Error("Diário: edição abandonada após esgotar tentativas",
				"edition_number", ed.EditionNumber, "retry_count", ed.RetryCount, "reason", reason)
		}
	}
}

func (p *SyncWorkerPool) processJob(ctx context.Context, job EditionJob) {
	ed := job.Edition
	if p.logger != nil {
		p.logger.Info("Iniciando ingestão da edição",
			"edition_number", ed.EditionNumber,
			"url", ed.PdfURL,
		)
	}

	// 1. Marca status como PROCESSING
	_ = p.repo.UpdateEditionStatus(ctx, ed.ID, domain.EditionStatusProcessing, 0, nil)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ed.PdfURL, nil)
	if err != nil {
		p.failEdition(ctx, ed, err.Error())
		return
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.failEdition(ctx, ed, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.failEdition(ctx, ed, fmt.Sprintf("bad status code: %d", resp.StatusCode))
		return
	}

	// 2. Extração em Stream via pdftotext (zero-disk)
	pdfText, err := p.parser.ExtractTextStream(ctx, resp.Body)
	if err != nil {
		p.failEdition(ctx, ed, err.Error())
		return
	}

	// 3. Parsing de atos de pessoal e contratos (findings para PostgreSQL)
	findings := p.parser.ParseTextFindings(pdfText, ed.EditionNumber, ed.ID)

	// 4. Persistência Atômica no PostgreSQL
	if err := p.repo.SaveFindingsTx(ctx, ed.ID, findings); err != nil {
		p.failEdition(ctx, ed, err.Error())
		return
	}
	for _, f := range findings {
		conf := f.Confidence
		if conf == "" {
			conf = "medium"
		}
		metrics.DiarioFindingsExtractedTotal.WithLabelValues(f.ActType, conf).Inc()
	}
	metrics.DiarioEditionsIngestedTotal.WithLabelValues("completed").Inc()

	// 5. Indexação no Typesense (best-effort — nunca bloqueia nem falha a
	// ingestão principal). Reindexa a partir dos findings recém-persistidos
	// no PostgreSQL (a fonte da verdade), não re-rodando o parser de blocos.
	if p.reindexer != nil {
		if n, err := p.reindexer.ReindexEdition(ctx, ed.ID); err != nil {
			if p.logger != nil {
				p.logger.Warn("Typesense: falha ao reindexar edição (best-effort)",
					"edition_number", ed.EditionNumber, "error", err)
			}
		} else if p.logger != nil {
			p.logger.Info("Typesense: edição reindexada", "edition_number", ed.EditionNumber, "docs", n)
		}
	}

	recordsCount := len(findings)
	if p.logger != nil {
		p.logger.Info("Ingestão concluída com sucesso",
			"edition_number", ed.EditionNumber,
			"findings_count", recordsCount,
		)
	}

	_ = p.repo.UpdateEditionStatus(ctx, ed.ID, domain.EditionStatusCompleted, recordsCount, nil)
}
