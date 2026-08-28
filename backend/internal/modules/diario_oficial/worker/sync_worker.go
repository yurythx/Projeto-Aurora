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
)

type EditionJob struct {
	Edition domain.Edition
}

type SyncWorkerPool struct {
	repo        domain.EditionRepository
	parser      *application.PDFParser
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
		reqInterval: 200 * time.Millisecond, // 5 req/s rate limiter
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		logger:      logger,
	}
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

func (p *SyncWorkerPool) processJob(ctx context.Context, job EditionJob) {
	ed := job.Edition
	if p.logger != nil {
		p.logger.Info("Iniciando ingestão da edição", "edition_number", ed.EditionNumber, "url", ed.PdfURL)
	}

	// 1. Marca status como PROCESSING
	_ = p.repo.UpdateEditionStatus(ctx, ed.ID, domain.EditionStatusProcessing, 0, nil)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ed.PdfURL, nil)
	if err != nil {
		errMsg := err.Error()
		_ = p.repo.UpdateEditionStatus(ctx, ed.ID, domain.EditionStatusFailed, 0, &errMsg)
		return
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		errMsg := err.Error()
		_ = p.repo.UpdateEditionStatus(ctx, ed.ID, domain.EditionStatusFailed, 0, &errMsg)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("bad status code: %d", resp.StatusCode)
		_ = p.repo.UpdateEditionStatus(ctx, ed.ID, domain.EditionStatusFailed, 0, &errMsg)
		return
	}

	// 2. Extração em Stream via pdftotext
	pdfText, err := p.parser.ExtractTextStream(ctx, resp.Body)
	if err != nil {
		errMsg := err.Error()
		_ = p.repo.UpdateEditionStatus(ctx, ed.ID, domain.EditionStatusFailed, 0, &errMsg)
		return
	}

	// 3. Parsing de atos de pessoal e contratos
	findings := p.parser.ParseTextFindings(pdfText, ed.EditionNumber, ed.ID)

	// 4. Persistência em Transação Atômica no PostgreSQL
	if err := p.repo.SaveFindingsTx(ctx, ed.ID, findings); err != nil {
		errMsg := err.Error()
		_ = p.repo.UpdateEditionStatus(ctx, ed.ID, domain.EditionStatusFailed, 0, &errMsg)
		return
	}

	if p.logger != nil {
		p.logger.Info("Ingestão concluída com sucesso", "edition_number", ed.EditionNumber, "findings_count", len(findings))
	}
}
