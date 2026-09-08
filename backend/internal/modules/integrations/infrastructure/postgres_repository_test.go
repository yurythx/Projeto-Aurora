package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-aurora/internal/modules/integrations/domain"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping live integrations repository test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// A tabela integrations nasce vazia (baseline — ver migrations/000001):
// nenhuma migration semeia mais uma linha fixa, então estes testes
// semeiam e limpam a sua própria, em vez de depender de dado fixo do
// schema.
const testIntegrationKey = "test-provider"

func seedIntegration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO integrations (key, name, type, enabled, status)
		VALUES ($1, 'Test Provider', 'webhook', true, 'unknown')
		ON CONFLICT (key) DO UPDATE
			SET status = 'unknown', last_error = NULL, last_success_at = NULL`,
		testIntegrationKey)
	if err != nil {
		t.Fatalf("seed integration: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM integrations WHERE key = $1`, testIntegrationKey)
	})
}

func TestPostgresRepository_List_IncludesSeeded(t *testing.T) {
	pool := testPool(t)
	seedIntegration(t, pool)
	repo := NewPostgresRepository(pool)

	list, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	found := false
	for _, i := range list {
		if i.Key == testIntegrationKey {
			found = true
		}
	}
	if !found {
		t.Errorf("expected the seeded %q integration to be present", testIntegrationKey)
	}
}

func TestPostgresRepository_UpdateStatusTx_ReportsChange(t *testing.T) {
	pool := testPool(t)
	seedIntegration(t, pool)
	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	updated, changed, err := repo.UpdateStatusTx(ctx, tx, testIntegrationKey, true, nil)
	if err != nil {
		t.Fatalf("UpdateStatusTx: %v", err)
	}
	if !changed {
		t.Error("expected status to change from unknown to online")
	}
	if updated.Status != domain.StatusOnline {
		t.Errorf("Status = %s, want online", updated.Status)
	}
	if updated.LastSuccessAt == nil {
		t.Error("expected LastSuccessAt to be set on success")
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Mesmo resultado de novo: o status NÃO deve ser reportado como mudado.
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	_, changedAgain, err := repo.UpdateStatusTx(ctx, tx, testIntegrationKey, true, nil)
	if err != nil {
		t.Fatalf("UpdateStatusTx (2nd): %v", err)
	}
	if changedAgain {
		t.Error("expected no status change when the outcome repeats")
	}
	_ = tx.Rollback(ctx)

	// Uma falha transiciona online -> offline com um erro registrado.
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	errMsg := "connection timeout"
	updated, changed, err = repo.UpdateStatusTx(ctx, tx, testIntegrationKey, false, &errMsg)
	if err != nil {
		t.Fatalf("UpdateStatusTx (failure): %v", err)
	}
	if !changed {
		t.Error("expected status to change from online to offline")
	}
	if updated.Status != domain.StatusOffline {
		t.Errorf("Status = %s, want offline", updated.Status)
	}
	if updated.LastError == nil || *updated.LastError != errMsg {
		t.Errorf("LastError = %v, want %q", updated.LastError, errMsg)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}
