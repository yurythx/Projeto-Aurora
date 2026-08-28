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

	// 2. Se JSON não retornou edições (ex.: resposta HTML do portal), faz parsing via expressão regular no HTML oficial
	if len(items) == 0 {
		re := regexp.MustCompile(`<a\s+href="([^"]+\.pdf)"\s+title="Baixar edição n°\s*(\d+)([^"]*)"`)
		matches := re.FindAllStringSubmatch(string(bodyBytes), -1)

		now := time.Now()
		seenEditions := make(map[string]bool)

		for idx, match := range matches {
			if len(match) < 3 {
				continue
			}
			pdfPath := match[1]
			edNumber := match[2]
			extraTitle := strings.TrimSpace(match[3])

			if !strings.HasPrefix(pdfPath, "http") {
				if strings.HasPrefix(pdfPath, "/") {
					pdfPath = "https://www.rondonopolis.mt.gov.br" + pdfPath
				} else {
					pdfPath = "https://www.rondonopolis.mt.gov.br/" + pdfPath
				}
			}

			editionKey := edNumber + "-" + pdfPath
			if seenEditions[editionKey] {
				continue
			}
			seenEditions[editionKey] = true

			if freeTextLower != "" && !strings.Contains(strings.ToLower(edNumber), freeTextLower) && !strings.Contains(strings.ToLower(extraTitle), freeTextLower) && !strings.Contains(strings.ToLower(pdfPath), freeTextLower) {
				continue
			}

			idVal := int64(6263 - idx)
			if n, pErr := strconv.ParseInt(edNumber, 10, 64); pErr == nil {
				idVal = n
			}

			pubDate := now.AddDate(0, 0, -idx)
			titleLabel := fmt.Sprintf("Edição Nº %s %s", edNumber, extraTitle)

			rawMap := map[string]interface{}{
				"edition_number": edNumber,
				"extra_title":    extraTitle,
				"pdf_url":        pdfPath,
				"parsed_at":      now.Format(time.RFC3339),
				"source":         "DIORONDON-E Portal Oficial",
			}
			rawPayload, _ := json.Marshal(rawMap)

			items = append(items, domain.SearchResultItem{
				ExternalID:       idVal,
				Tribunal:         "Prefeitura Municipal de Rondonópolis / DIORONDON-E",
				Orgao:            "Secretaria Municipal de Administração, Gestão de Pessoas e Inovação",
				TipoComunicacao:  strings.TrimSpace(titleLabel),
				Texto:            fmt.Sprintf("Publicação oficial do Diário Oficial de Rondonópolis - Edição Nº %s %s. Disponível para download e auditoria.", edNumber, extraTitle),
				AvailabilityDate: pubDate,
				Link:             pdfPath,
				RawPayload:       rawPayload,
			})

			if len(items) >= 20 {
				break
			}
		}
	}

	// 3. PARSING EM STREAM DE CONTEÚDO PDF (Extrai texto real do PDF via pdftotext)
	maxExtract := 5
	if len(items) < maxExtract {
		maxExtract = len(items)
	}

	for i := 0; i < maxExtract; i++ {
		pdfURL := items[i].Link
		if pdfURL != "" && strings.Contains(pdfURL, ".pdf") {
			pdfText, pdfErr := ExtractPDFTextStream(ctx, c.client, pdfURL)
			if pdfErr == nil && len(pdfText) > 0 {
				snippet := pdfText
				if len(snippet) > 1000 {
					snippet = snippet[:1000] + "..."
				}
				items[i].Texto = snippet

				rawMap := map[string]interface{}{
					"edition_number":     items[i].TipoComunicacao,
					"pdf_url":            pdfURL,
					"pdf_text_extracted": true,
					"extracted_snippet":  snippet,
					"full_text_length":   len(pdfText),
					"source":             "DIORONDON-E PDF Stream Extractor",
				}
				rawPayloadBytes, _ := json.Marshal(rawMap)
				items[i].RawPayload = json.RawMessage(rawPayloadBytes)
			}
		}
	}

	return &domain.SearchResult{
		Items:      items,
		TotalCount: len(items),
	}, nil
}
