// Package infrastructure implementa os adapters de integração externa do módulo diario_oficial.
package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
	"github.com/yurythx/projeto-nova/internal/platform/metrics"
	"github.com/yurythx/projeto-nova/internal/platform/resilience"
)

const rondonopolisProviderLabel = "diario-oficial-rondonopolis"

// RondonopolisClient conecta com a API pública do Diário Oficial de Rondonópolis (DIORONDON-E).
type RondonopolisClient struct {
	baseURL string
	token   string
	client  *http.Client
	breaker *resilience.Breaker[*http.Response]
}

// NewRondonopolisClient constrói um cliente configurado para o Diário Oficial de Rondonópolis.
func NewRondonopolisClient(baseURL, token string, timeout time.Duration, logger *slog.Logger) *RondonopolisClient {
	return &RondonopolisClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client:  &http.Client{Timeout: timeout},
		breaker: resilience.New[*http.Response](resilience.Options{Name: rondonopolisProviderLabel, Logger: logger}),
	}
}

var _ domain.Client = (*RondonopolisClient)(nil)

type rondonopolisEdition struct {
	ID          int64  `json:"id"`
	Number      string `json:"number"`
	PublishDate string `json:"publish_date"`
	DocURL      string `json:"doc_url"`
	Content     string `json:"content"`
}

type rondonopolisResponse struct {
	Editions []rondonopolisEdition `json:"editions"`
}

func (c *RondonopolisClient) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	metrics.IntegrationRequestsTotal.WithLabelValues(rondonopolisProviderLabel).Inc()
	start := time.Now()

	if c.token != "" {
		req.Header.Set("Authorization", "Token "+c.token)
	}
	req.Header.Set("User-Agent", "Projeto-Nova/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.breaker.Execute(func() (*http.Response, error) {
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= http.StatusInternalServerError {
			resp.Body.Close()
			return nil, fmt.Errorf("rondonopolis diary responded with status %d", resp.StatusCode)
		}
		return resp, nil
	})
	metrics.IntegrationDuration.WithLabelValues(rondonopolisProviderLabel).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.IntegrationFailuresTotal.WithLabelValues(rondonopolisProviderLabel).Inc()
		if appErr, ok := apperrors.As(err); ok && appErr.Code == "CIRCUIT_OPEN" {
			return nil, appErr
		}
		return nil, apperrors.DependencyUnavailable(fmt.Sprintf("rondonopolis diary request failed: %v", err)).WithCode("INTEGRATION_UNAVAILABLE")
	}
	return resp, nil
}

func (c *RondonopolisClient) Check(ctx context.Context) (*domain.CheckResult, error) {
	if c.baseURL == "" || c.token == "" {
		return nil, apperrors.DependencyUnavailable("Rondonópolis Diário Oficial integration is not configured").WithCode("INTEGRATION_UNAVAILABLE")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.baseURL+"/", nil)
	if err != nil {
		return nil, fmt.Errorf("rondonopolis: build check request: %w", err)
	}

	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &domain.CheckResult{
		StatusCode: resp.StatusCode,
		Summary:    fmt.Sprintf("rondonopolis responded with HTTP %d", resp.StatusCode),
	}, nil
}

// parsePortalDate lê a "Data de Edição" da tabela do portal ("27/08/26" ou
// "27/08/2026"). Retorna o zero de time.Time (não uma data inventada) se não
// parsear — a montante isso vira NULL no banco.
func parsePortalDate(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"02/01/2006", "02/01/06", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		}
	}
	return time.Time{}
}

func (c *RondonopolisClient) Search(ctx context.Context, query domain.SearchQuery) (*domain.SearchResult, error) {
	if c.baseURL == "" {
		return nil, apperrors.DependencyUnavailable("Rondonópolis Diário Oficial integration is not configured").WithCode("INTEGRATION_UNAVAILABLE")
	}

	searchURL := c.baseURL
	if !strings.HasSuffix(searchURL, "/") {
		searchURL += "/"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("rondonopolis: build search request: %w", err)
	}

	resp, err := c.do(ctx, req)
	if err != nil {
		// Se c.baseURL /api/v1/diary/ falhar ou expirar, tentar fall back direto no portal diário oficial HTML
		portalReq, pErr := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.rondonopolis.mt.gov.br/diario-oficial/", nil)
		if pErr == nil {
			resp, err = c.client.Do(portalReq)
		}
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rondonopolis: read body: %w", err)
	}

	items := make([]domain.SearchResultItem, 0)
	freeTextLower := strings.ToLower(query.FreeText)

	// 1. Tenta decodificar como JSON estruturado da API
	var rResp rondonopolisResponse
	if jErr := json.Unmarshal(bodyBytes, &rResp); jErr == nil && len(rResp.Editions) > 0 {
		for _, ed := range rResp.Editions {
			pubTime, pErr := time.Parse(time.RFC3339, ed.PublishDate)
			if pErr != nil {
				pubTime, pErr = time.Parse("2006-01-02T15:04:05", ed.PublishDate)
				if pErr != nil {
					pubTime = time.Now()
				}
			}

			if query.Since != nil && pubTime.Before(*query.Since) {
				continue
			}

			if freeTextLower != "" && !strings.Contains(strings.ToLower(ed.Content), freeTextLower) && !strings.Contains(strings.ToLower(ed.Number), freeTextLower) {
				continue
			}

			rawPayload, _ := json.Marshal(ed)
			snippet := ed.Content
			if len(snippet) > 1000 {
				snippet = snippet[:1000] + "..."
			}

			items = append(items, domain.SearchResultItem{
				ExternalID:       ed.ID,
				Tribunal:         "Prefeitura Municipal de Rondonópolis / DIORONDON-E",
				Orgao:            "Secretaria Municipal de Administração, Gestão de Pessoas e Inovação",
				TipoComunicacao:  fmt.Sprintf("Edição Nº %s", ed.Number),
				Texto:            snippet,
				AvailabilityDate: pubTime,
				Link:             ed.DocURL,
				RawPayload:       rawPayload,
			})

			if len(items) >= 100 {
				break
			}
		}
	}

	// 2. Se JSON não retornou edições (ex.: resposta HTML do portal), faz parsing da TABELA oficial.
	// Estrutura real: <tr><th scope="row">6265</th><td ...>27/08/26</td><td ...><a href="...pdf" title="Baixar edição n° 6265 ...">
	if len(items) == 0 {
		rowRe := regexp.MustCompile(`(?is)<th[^>]*scope="row"[^>]*>\s*(\d+)\s*</th>\s*<td[^>]*>\s*([\d/]{6,10})\s*</td>\s*<td[^>]*>\s*<a\s+href="([^"]+\.pdf)"`)
		matches := rowRe.FindAllStringSubmatch(string(bodyBytes), -1)

		// Fallback: layout mudou e a linha não casou — pega ao menos o <a> (sem data).
		if len(matches) == 0 {
			aRe := regexp.MustCompile(`(?i)<a\s+href="([^"]+\.pdf)"\s+title="Baixar edição n°\s*(\d+)`)
			for _, m := range aRe.FindAllStringSubmatch(string(bodyBytes), -1) {
				matches = append(matches, []string{m[0], m[2], "", m[1]}) // [_, ednum, data-vazia, href]
			}
		}

		seenEditions := make(map[string]bool)
		for _, match := range matches {
			edNumber := match[1]
			dateStr := strings.TrimSpace(match[2])
			pdfPath := match[3]

			if !strings.HasPrefix(pdfPath, "http") {
				if strings.HasPrefix(pdfPath, "/") {
					pdfPath = "https://www.rondonopolis.mt.gov.br" + pdfPath
				} else {
					pdfPath = "https://www.rondonopolis.mt.gov.br/" + pdfPath
				}
			}

			if seenEditions[edNumber] {
				continue
			}
			seenEditions[edNumber] = true

			if freeTextLower != "" && !strings.Contains(strings.ToLower(edNumber), freeTextLower) && !strings.Contains(strings.ToLower(pdfPath), freeTextLower) {
				continue
			}

			var idVal int64
			if n, pErr := strconv.ParseInt(edNumber, 10, 64); pErr == nil {
				idVal = n
			} else {
				continue // sem número de edição não há como deduplicar
			}

			// Data REAL da tabela (dd/mm/yy ou dd/mm/yyyy). Se não parsear,
			// zero — o watcher grava NULL, nunca uma data inventada.
			pubDate := parsePortalDate(dateStr)
			if query.Since != nil && !pubDate.IsZero() && pubDate.Before(*query.Since) {
				continue
			}

			rawMap := map[string]interface{}{
				"edition_number": edNumber,
				"edition_date":   dateStr,
				"pdf_url":        pdfPath,
				"parsed_at":      time.Now().Format(time.RFC3339),
				"source":         "DIORONDON-E Portal Oficial (tabela)",
			}
			rawPayload, _ := json.Marshal(rawMap)

			items = append(items, domain.SearchResultItem{
				ExternalID:       idVal,
				Tribunal:         "Prefeitura Municipal de Rondonópolis / DIORONDON-E",
				Orgao:            "Secretaria Municipal de Administração, Gestão de Pessoas e Inovação",
				TipoComunicacao:  fmt.Sprintf("Edição Nº %s", edNumber),
				Texto:            fmt.Sprintf("Diário Oficial de Rondonópolis - Edição Nº %s (%s).", edNumber, dateStr),
				AvailabilityDate: pubDate,
				Link:             pdfPath,
				RawPayload:       rawPayload,
			})

			if len(items) >= 30 {
				break
			}
		}
	}

	// 3. Extração de SNIPPET do PDF — só quando há termo de busca (o feed que
	// mostra trechos). A descoberta do watcher (query vazia) pula isto: baixar
	// e rodar pdftotext em 5 PDFs a cada Search estourava o timeout do portal
	// e ainda sobrescrevia o edition_date do RawPayload.
	if freeTextLower != "" {
		maxExtract := 5
		if len(items) < maxExtract {
			maxExtract = len(items)
		}
		for i := 0; i < maxExtract; i++ {
			pdfURL := items[i].Link
			if pdfURL == "" || !strings.Contains(pdfURL, ".pdf") {
				continue
			}
			pdfText, pdfErr := ExtractPDFTextStream(ctx, c.client, pdfURL)
			if pdfErr != nil || len(pdfText) == 0 {
				continue
			}
			snippet := pdfText
			if len(snippet) > 1000 {
				snippet = snippet[:1000] + "..."
			}
			items[i].Texto = snippet
		}
	}

	return &domain.SearchResult{
		Items:      items,
		TotalCount: len(items),
	}, nil
}
