package typesense

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type CollectionField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Facet    bool   `json:"facet,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	Index    *bool  `json:"index,omitempty"`
	Sort     bool   `json:"sort,omitempty"`
}

type CollectionSchema struct {
	Name                string            `json:"name"`
	Fields              []CollectionField `json:"fields"`
	DefaultSortingField string            `json:"default_sorting_field,omitempty"`
}

func (c *Client) EnsureCollections(ctx context.Context) error {
	articlesSchema := CollectionSchema{
		Name: "diorondon_articles",
		Fields: []CollectionField{
			{Name: "id", Type: "string"},
			{Name: "edition_number", Type: "int32", Facet: true},
			{Name: "edition_type", Type: "string", Facet: true},
			{Name: "publication_date", Type: "int64", Facet: true},
			{Name: "page_number", Type: "int32"},
			{Name: "contract_numbers", Type: "string[]", Facet: true},
			{Name: "cnpjs", Type: "string[]", Facet: true},
			{Name: "officials_named", Type: "string[]", Facet: true},
			{Name: "content", Type: "string"},
			// pdf_storage_url também é criado pelo frontend (typesense-client.ts);
			// mantido aqui para o schema ser idêntico independente de quem criar
			// a coleção primeiro (o schema é imutável após a criação).
			{Name: "pdf_storage_url", Type: "string", Optional: true},
		},
		DefaultSortingField: "publication_date",
	}

	personnelSchema := CollectionSchema{
		Name: "diorondon_personnel_acts",
		Fields: []CollectionField{
			{Name: "id", Type: "string"},
			{Name: "edition_number", Type: "int32", Facet: true},
			{Name: "edition_type", Type: "string", Facet: true},
			{Name: "publication_date", Type: "int64", Facet: true},
			{Name: "act_type", Type: "string", Facet: true},
			// person_name é ordenável: a busca sem termo lista servidores em
			// ordem alfabética (sort_by=person_name:asc). Sem sort:true o
			// Typesense rejeita a query inteira e o front cai no fallback.
			{Name: "person_name", Type: "string", Facet: true, Sort: true},
			{Name: "person_cpf", Type: "string", Facet: true, Optional: true},
			{Name: "person_matricula", Type: "string", Facet: true, Optional: true},
			{Name: "job_role", Type: "string", Facet: true},
			{Name: "secretaria", Type: "string", Facet: true},
			{Name: "das_level", Type: "string", Facet: true, Optional: true},
			{Name: "salary_value", Type: "float", Facet: true, Optional: true},
			{Name: "portaria_number", Type: "string", Facet: true, Optional: true},
			{Name: "full_act_text", Type: "string"},
			{Name: "pdf_page_number", Type: "int32"},
			{Name: "pdf_storage_url", Type: "string"},
			{Name: "confidence", Type: "string", Facet: true, Optional: true},
		},
		DefaultSortingField: "publication_date",
	}

	for _, schema := range []CollectionSchema{articlesSchema, personnelSchema} {
		if err := c.createCollectionIfNotExists(ctx, schema); err != nil {
			return fmt.Errorf("typesense: ensure collection %s: %w", schema.Name, err)
		}
	}
	return nil
}

// DropCollection remove a coleção inteira (documentos + schema). Um 404 é
// tratado como sucesso — o objetivo é "garantir que não existe".
func (c *Client) DropCollection(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%s/collections/%s", c.baseURL, name), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("typesense: drop collection %s: status %d", name, resp.StatusCode)
	}
	return nil
}

// RecreateCollections apaga e recria as duas coleções do DIORONDON. Usado pelo
// reindex explícito com --recreate para aplicar mudanças de schema (que o
// Typesense não permite alterar in-place).
func (c *Client) RecreateCollections(ctx context.Context) error {
	for _, name := range []string{"diorondon_personnel_acts", "diorondon_articles"} {
		if err := c.DropCollection(ctx, name); err != nil {
			return err
		}
	}
	return c.EnsureCollections(ctx)
}

func (c *Client) createCollectionIfNotExists(ctx context.Context, schema CollectionSchema) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/collections/%s", c.baseURL, schema.Name), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return nil // Já existe
	}
	if resp != nil {
		resp.Body.Close()
	}

	body, err := json.Marshal(schema)
	if err != nil {
		return err
	}

	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/collections", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return err
	}
	postReq.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)
	postReq.Header.Set("Content-Type", "application/json")

	postResp, err := c.httpClient.Do(postReq)
	if err != nil {
		return err
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusCreated && postResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create collection, status: %d", postResp.StatusCode)
	}
	return nil
}

// DeleteDocumentsByFilter remove todos os documentos que casam com filterBy
// (ex.: "edition_number:=6264") e devolve quantos foram removidos. Um 404
// (coleção ainda não existe) conta como zero removidos, não erro.
func (c *Client) DeleteDocumentsByFilter(ctx context.Context, collectionName, filterBy string) (int, error) {
	u := fmt.Sprintf("%s/collections/%s/documents?filter_by=%s", c.baseURL, collectionName, url.QueryEscape(filterBy))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("typesense delete by filter (%s) failed: %d %s", filterBy, resp.StatusCode, truncate(string(body), 300))
	}
	var res struct {
		NumDeleted int `json:"num_deleted"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)
	return res.NumDeleted, nil
}

func (c *Client) IndexDocument(ctx context.Context, collectionName string, doc interface{}) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/collections/%s/documents?action=upsert", c.baseURL, collectionName), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("indexing document failed with status %d", resp.StatusCode)
	}
	return nil
}

// ImportDocuments faz upsert em lote via endpoint JSONL /documents/import.
// Muito mais rápido que IndexDocument num loop para o reindexador (centenas
// de docs). Erros por documento vêm no corpo JSONL da resposta: se algum
// documento falhar, retorna erro com a primeira linha problemática.
func (c *Client) ImportDocuments(ctx context.Context, collectionName string, docs []interface{}) error {
	if len(docs) == 0 {
		return nil
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, d := range docs {
		if err := enc.Encode(d); err != nil {
			return fmt.Errorf("typesense import: encode doc: %w", err)
		}
	}

	url := fmt.Sprintf("%s/collections/%s/documents/import?action=upsert", c.baseURL, collectionName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("typesense import failed with status %d: %s", resp.StatusCode, truncate(string(body), 500))
	}
	// A resposta é JSONL: uma linha {"success":true} por doc. Procura falhas.
	for _, line := range bytes.Split(body, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var res struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		if err := json.Unmarshal(line, &res); err == nil && !res.Success {
			return fmt.Errorf("typesense import: at least one document failed: %s", res.Error)
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// APIKey representa uma chave da API do Typesense. O campo Value só vem
// preenchido na resposta de criação; em listagens vem só ValuePrefix.
type APIKey struct {
	ID          int      `json:"id"`
	Value       string   `json:"value,omitempty"`
	ValuePrefix string   `json:"value_prefix,omitempty"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
	Collections []string `json:"collections"`
}

// CreateSearchOnlyKey cria uma chave restrita a `documents:search` nas coleções
// dadas e devolve seu valor completo (só disponível neste momento). Serve para
// entregar ao navegador sem expor a chave admin.
func (c *Client) CreateSearchOnlyKey(ctx context.Context, description string, collections []string) (APIKey, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"description": description,
		"actions":     []string{"documents:search"},
		"collections": collections,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/keys", bytes.NewReader(body))
	if err != nil {
		return APIKey{}, err
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return APIKey{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return APIKey{}, fmt.Errorf("typesense create key failed: %d %s", resp.StatusCode, truncate(string(b), 300))
	}
	var k APIKey
	if err := json.NewDecoder(resp.Body).Decode(&k); err != nil {
		return APIKey{}, err
	}
	return k, nil
}

// ListKeys devolve as chaves existentes (sem o valor completo, só o prefixo).
func (c *Client) ListKeys(ctx context.Context) ([]APIKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/keys", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("typesense list keys failed: %d %s", resp.StatusCode, truncate(string(b), 300))
	}
	var out struct {
		Keys []APIKey `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Keys, nil
}

type SearchParams struct {
	Q        string
	QueryBy  string
	FilterBy string
	FacetBy  string
	Page     int
	PerPage  int
	SortBy   string
}

type SearchResponse struct {
	Found       int          `json:"found"`
	Page        int          `json:"page"`
	Hits        []SearchHit  `json:"hits"`
	FacetCounts []FacetCount `json:"facet_counts"`
}

type SearchHit struct {
	Document   map[string]interface{} `json:"document"`
	Highlights []SearchHighlight      `json:"highlights"`
}

type SearchHighlight struct {
	Field    string   `json:"field"`
	Snippet  string   `json:"snippet"`
	Snippets []string `json:"snippets"`
}

type FacetCount struct {
	FieldName string       `json:"field_name"`
	Counts    []FacetValue `json:"counts"`
}

type FacetValue struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

func (c *Client) Search(ctx context.Context, collectionName string, params SearchParams) (*SearchResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/collections/%s/documents/search", c.baseURL, collectionName), nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	if params.Q != "" {
		q.Add("q", params.Q)
	} else {
		q.Add("q", "*")
	}
	if params.QueryBy != "" {
		q.Add("query_by", params.QueryBy)
	}
	if params.FilterBy != "" {
		q.Add("filter_by", params.FilterBy)
	}
	if params.FacetBy != "" {
		q.Add("facet_by", params.FacetBy)
	}
	if params.Page > 0 {
		q.Add("page", fmt.Sprintf("%d", params.Page))
	}
	if params.PerPage > 0 {
		q.Add("per_page", fmt.Sprintf("%d", params.PerPage))
	}
	if params.SortBy != "" {
		q.Add("sort_by", params.SortBy)
	}

	req.URL.RawQuery = q.Encode()
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed with status %d", resp.StatusCode)
	}

	var res SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}
