package infrastructure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
)

func TestRondonopolisClient_Check_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token test-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := NewRondonopolisClient(ts.URL, "test-token", 5*time.Second, nil)
	result, err := client.Check(context.Background())
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", result.StatusCode, http.StatusOK)
	}
}

func TestRondonopolisClient_Search_Success(t *testing.T) {
	jsonResp := `{
		"editions": [
			{
				"id": 101,
				"number": "6000",
				"publish_date": "2026-08-25T10:00:00Z",
				"doc_url": "http://example.com/pdf1.pdf",
				"content": "Portaria Exonerar a pedido Joao da Silva"
			}
		]
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token test-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonResp))
	}))
	defer ts.Close()

	client := NewRondonopolisClient(ts.URL, "test-token", 5*time.Second, nil)
	result, err := client.Search(context.Background(), domain.SearchQuery{FreeText: "Joao"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("Items count = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.ExternalID != 101 {
		t.Errorf("ExternalID = %d, want 101", item.ExternalID)
	}
	if item.TipoComunicacao != "Edição Nº 6000" {
		t.Errorf("TipoComunicacao = %q, want Edição Nº 6000", item.TipoComunicacao)
	}
}

func TestParsePortalDate(t *testing.T) {
	cases := map[string]string{
		"27/08/26":   "2026-08-27",
		"27/08/2026": "2026-08-27",
		"01/01/25":   "2025-01-01",
		"2026-08-27": "2026-08-27",
	}
	for in, want := range cases {
		got := parsePortalDate(in)
		if got.Format("2006-01-02") != want {
			t.Errorf("parsePortalDate(%q) = %s, want %s", in, got.Format("2006-01-02"), want)
		}
	}
	// Lixo -> zero (NUNCA uma data inventada).
	for _, bad := range []string{"", "n/d", "hoje", "32/13/99", "27-08-2026"} {
		if !parsePortalDate(bad).IsZero() {
			t.Errorf("parsePortalDate(%q) devia ser zero, got %v", bad, parsePortalDate(bad))
		}
	}
}

// TestRondonopolisClient_Search_HTMLTable garante que a data REAL da tabela do
// portal é extraída (e não uma data decrementada inventada).
func TestRondonopolisClient_Search_HTMLTable(t *testing.T) {
	html := `<table class="table align-middle"><thead><tr><th>Edição</th><th>Data de Edição</th><th>Baixar</th></tr></thead><tbody>
	<tr><th scope="row">6265</th><td class="text-center">27/08/26</td><td class="text-center"><a href="/media/docs/edicoes/2026/x.pdf" title="Baixar edição n° 6265 (PDF)"></a></td></tr>
	<tr><th scope="row">6264</th><td class="text-center">26/08/26</td><td class="text-center"><a href="/media/docs/edicoes/2026/y.pdf" title="Baixar edição n° 6264 (PDF)"></a></td></tr>
	</tbody></table>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer ts.Close()

	client := NewRondonopolisClient(ts.URL, "", 5*time.Second, nil)
	res, err := client.Search(context.Background(), domain.SearchQuery{})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("Items = %d, want 2", len(res.Items))
	}
	byNum := map[int64]domain.SearchResultItem{}
	for _, it := range res.Items {
		byNum[it.ExternalID] = it
	}
	if got := byNum[6265].AvailabilityDate.Format("2006-01-02"); got != "2026-08-27" {
		t.Errorf("6265 date = %s, want 2026-08-27", got)
	}
	if !strings.HasSuffix(byNum[6265].Link, "/media/docs/edicoes/2026/x.pdf") {
		t.Errorf("6265 link = %q", byNum[6265].Link)
	}
	if got := byNum[6264].AvailabilityDate.Format("2006-01-02"); got != "2026-08-26" {
		t.Errorf("6264 date = %s, want 2026-08-26", got)
	}
}
