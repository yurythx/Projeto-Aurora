package infrastructure

import (
	"context"
	"net/http"
	"net/http/httptest"
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
