package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
)

// EditionJob representa um trabalho individual de colheita/extração de edição do diário oficial.
type EditionJob struct {
	ID            int64
	EditionNumber string
	PdfURL        string
	PublishDate   time.Time
	Title         string
}

// Harvester implementa a estratégia ETL de Worker Pool + Rate Limiting + Stream Parsing.
type Harvester struct {
	client      *RondonopolisClient
	numWorkers  int
	reqInterval time.Duration
	logger      *slog.Logger
	httpClient  *http.Client
}

// NewHarvester instancia o Harvester com pool configurável de trabalhadores e rate limiting.
func NewHarvester(client *RondonopolisClient, logger *slog.Logger) *Harvester {
	return &Harvester{
		client:      client,
		numWorkers:  5,                      // Pool de 5 goroutines consumidoras em paralelo
		reqInterval: 200 * time.Millisecond, // Rate Limiting de 5 requisições por segundo
		logger:      logger,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// ExtractPDFTextStream efetua o download em stream do PDF da edição e extrai o texto bruto via pdftotext.
func ExtractPDFTextStream(ctx context.Context, httpClient *http.Client, pdfURL string) (string, error) {
	if pdfURL == "" {
		return "", fmt.Errorf("empty pdf url")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pdfURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad pdf status: %d", resp.StatusCode)
	}

	cmd := exec.CommandContext(ctx, "pdftotext", "-", "-")
	cmd.Stdin = resp.Body
	var outBuffer bytes.Buffer
	cmd.Stdout = &outBuffer

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext execution failed: %w", err)
	}

	return strings.TrimSpace(outBuffer.String()), nil
}

// HarvestExec(ctx) executa o ciclo completo de ETL (Discovery, Worker Pool, Rate Limiting, PDF Text Parsing).
func (h *Harvester) HarvestExec(ctx context.Context, query domain.SearchQuery) (*domain.SearchResult, error) {
	if h.client == nil {
		return nil, fmt.Errorf("harvester: client is nil")
	}

	// 1. DISCOVERY PHASE (Produtor): Busca lista de edições disponíveis
	res, err := h.client.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("harvester discovery: %w", err)
	}

	if res == nil || len(res.Items) == 0 {
		return &domain.SearchResult{Items: []domain.SearchResultItem{}, TotalCount: 0}, nil
	}

	jobsChan := make(chan EditionJob, len(res.Items))
	resultsChan := make(chan domain.SearchResultItem, len(res.Items))

	// Enfileira os trabalhos identificados no Discovery
	for _, item := range res.Items {
		jobsChan <- EditionJob{
			ID:            item.ExternalID,
			EditionNumber: item.TipoComunicacao,
			PdfURL:        item.Link,
			PublishDate:   item.AvailabilityDate,
			Title:         item.Texto,
		}
	}
	close(jobsChan)

	// 2. RATE LIMITER: Ticker que controla a cadência máxima de requisições por segundo
	limiter := time.NewTicker(h.reqInterval)
	defer limiter.Stop()

	// 3. WORKER POOL PHASE: Inicializa as goroutines consumidoras com parsing de PDF
	var wg sync.WaitGroup
	for w := 1; w <= h.numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for job := range jobsChan {
				select {
				case <-ctx.Done():
					return
				case <-limiter.C:
					extractedText, pdfErr := ExtractPDFTextStream(ctx, h.httpClient, job.PdfURL)

					finalText := job.Title
					hasExtracted := false
					if pdfErr == nil && len(extractedText) > 0 {
						finalText = extractedText
						hasExtracted = true
					}

					freeTextLower := strings.ToLower(query.FreeText)
					if freeTextLower != "" && !strings.Contains(strings.ToLower(finalText), freeTextLower) && !strings.Contains(strings.ToLower(job.EditionNumber), freeTextLower) {
						continue
					}

					snippet := finalText
					if len(snippet) > 1000 {
						snippet = snippet[:1000] + "..."
					}

					rawPayloadMap := map[string]interface{}{
						"edition_number":     job.EditionNumber,
						"pdf_url":            job.PdfURL,
						"publish_date":       job.PublishDate.Format(time.RFC3339),
						"pdf_text_extracted": hasExtracted,
						"extracted_snippet":  snippet,
						"source":             "DIORONDON-E PDF Parser Stream",
					}
					rawPayloadBytes, _ := json.Marshal(rawPayloadMap)

					itemResult := domain.SearchResultItem{
						ExternalID:       job.ID,
						Tribunal:         "Prefeitura Municipal de Rondonópolis / DIORONDON-E",
						Orgao:            "Secretaria Municipal de Administração, Gestão de Pessoas e Inovação",
						TipoComunicacao:  job.EditionNumber,
						Texto:            snippet,
						AvailabilityDate: job.PublishDate,
						Link:             job.PdfURL,
						RawPayload:       json.RawMessage(rawPayloadBytes),
					}
					resultsChan <- itemResult
				}
			}
		}(w)
	}

	// Aguarda a conclusão de todas as goroutines do pool
	wg.Wait()
	close(resultsChan)

	// 4. CHECKPOINTER & ACCUMULATOR: Coleta resultados idempotentes
	harvestedItems := make([]domain.SearchResultItem, 0, len(res.Items))
	for item := range resultsChan {
		harvestedItems = append(harvestedItems, item)
	}

	return &domain.SearchResult{
		Items:      harvestedItems,
		TotalCount: len(harvestedItems),
	}, nil
}
