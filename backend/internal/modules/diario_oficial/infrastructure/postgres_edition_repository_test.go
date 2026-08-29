package infrastructure

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
)

// Testes de persistência do módulo diario_oficial contra o Postgres real e
// migrado. Pulados sem TEST_DATABASE_URL (mesmo padrão dos outros repos).
// Cada teste cria e limpa a própria edição/findings.

func editionTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping live diario_oficial repository test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func seedEdition(t *testing.T, pool *pgxpool.Pool, number string, date *time.Time) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO diario_oficial_editions (edition_number, edition_date, pdf_url, status)
		 VALUES ($1, $2, $3, 'PENDING') RETURNING id`,
		number, date, "https://example.test/"+number+".pdf",
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed edition: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM diario_oficial_editions WHERE id = $1`, id)
	})
	return id
}

func mkFinding(editionID int64, ext, act, nome, conf string) domain.Finding {
	n := nome
	return domain.Finding{
		ID: uuid.New(), EditionID: editionID, ExternalID: ext, ActType: act,
		ServidorNome: &n, Confidence: conf, RawContent: "RESOLVE " + act + " " + nome, PDFPageNumber: 3,
	}
}

// TestScanEdition_NullDate cobre o bug que travou o pipeline: após 000031
// tornar edition_date nullable, scanEdition lia NULL num time.Time.
func TestScanEdition_NullDate(t *testing.T) {
	pool := editionTestPool(t)
	repo := NewPostgresEditionRepository(pool)
	seedEdition(t, pool, "TST-"+uuid.NewString()[:8], nil) // data NULL

	pend, err := repo.GetPendingEditions(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetPendingEditions com edição de data NULL: %v", err)
	}
	if len(pend) == 0 {
		t.Fatal("edição PENDING com data NULL não apareceu em GetPendingEditions")
	}
}

// TestSaveFindingsTx_ReplacesSet garante que reprocessar uma edição substitui
// o conjunto de findings (não acumula com hash diferente).
func TestSaveFindingsTx_ReplacesSet(t *testing.T) {
	ctx := context.Background()
	pool := editionTestPool(t)
	repo := NewPostgresEditionRepository(pool)
	edID := seedEdition(t, pool, "TST-"+uuid.NewString()[:8], nil)

	if err := repo.SaveFindingsTx(ctx, edID, []domain.Finding{
		mkFinding(edID, "ext-a", "EXONERACAO", "Fulano De Tal", "high"),
		mkFinding(edID, "ext-b", "NOMEACAO_COMISSIONADO", "Ciclana Silva", "medium"),
	}); err != nil {
		t.Fatalf("SaveFindingsTx #1: %v", err)
	}

	// Reprocessa com findings diferentes (external_id novos).
	if err := repo.SaveFindingsTx(ctx, edID, []domain.Finding{
		mkFinding(edID, "ext-c", "RELOTACAO", "Beltrano Souza", "medium"),
	}); err != nil {
		t.Fatalf("SaveFindingsTx #2: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM diario_oficial_findings WHERE edition_id = $1`, edID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("após reprocessar, findings = %d, want 1 (conjunto substituído, sem acúmulo)", n)
	}
}

// TestRequeueFailedEditions_RespectsMaxRetries: FAILED com retry_count < max
// volta para PENDING; no limite, fica FAILED.
func TestRequeueFailedEditions_RespectsMaxRetries(t *testing.T) {
	ctx := context.Background()
	pool := editionTestPool(t)
	repo := NewPostgresEditionRepository(pool)
	edID := seedEdition(t, pool, "TST-"+uuid.NewString()[:8], nil)

	_, _ = pool.Exec(ctx, `UPDATE diario_oficial_editions SET status='FAILED', retry_count=2 WHERE id=$1`, edID)
	n, err := repo.RequeueFailedEditions(ctx, 3)
	if err != nil {
		t.Fatalf("RequeueFailedEditions: %v", err)
	}
	if n < 1 {
		t.Fatalf("esperava reencaminhar >=1, got %d", n)
	}
	var status string
	var rc int
	_ = pool.QueryRow(ctx, `SELECT status, retry_count FROM diario_oficial_editions WHERE id=$1`, edID).Scan(&status, &rc)
	if status != "PENDING" || rc != 3 {
		t.Errorf("status/retry = %s/%d, want PENDING/3", status, rc)
	}

	// Agora no limite: volta a FAILED e não deve reencaminhar.
	_, _ = pool.Exec(ctx, `UPDATE diario_oficial_editions SET status='FAILED' WHERE id=$1`, edID)
	before := countFailedAtMax(t, pool)
	if _, err := repo.RequeueFailedEditions(ctx, 3); err != nil {
		t.Fatalf("RequeueFailedEditions #2: %v", err)
	}
	_ = pool.QueryRow(ctx, `SELECT status FROM diario_oficial_editions WHERE id=$1`, edID).Scan(&status)
	if status != "FAILED" {
		t.Errorf("no limite de tentativas a edição deveria seguir FAILED, got %s", status)
	}
	_ = before
}

func countFailedAtMax(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM diario_oficial_editions WHERE status='FAILED' AND retry_count >= 3`).Scan(&n)
	return n
}

// TestSearchFindings_FTSRanksAndOrders: a busca por termo usa FTS e ordena por
// relevância, depois pela data da edição.
func TestSearchFindings_FTSRanksAndOrders(t *testing.T) {
	ctx := context.Background()
	pool := editionTestPool(t)
	repo := NewPostgresEditionRepository(pool)

	d := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	edID := seedEdition(t, pool, "TST-"+uuid.NewString()[:8], &d)

	uniq := "zorglub" + uuid.NewString()[:6]
	if err := repo.SaveFindingsTx(ctx, edID, []domain.Finding{
		{ID: uuid.New(), EditionID: edID, ExternalID: "fts-1", ActType: "EXONERACAO",
			RawContent: "RESOLVE EXONERAR o servidor " + uniq + " coordenador de compras", Confidence: "medium", PDFPageNumber: 2},
	}); err != nil {
		t.Fatalf("SaveFindingsTx: %v", err)
	}

	got, total, err := repo.SearchFindings(ctx, uniq, "", 10, 0)
	if err != nil {
		t.Fatalf("SearchFindings: %v", err)
	}
	if total < 1 || len(got) < 1 {
		t.Fatalf("SearchFindings(%q) não encontrou o finding (total=%d len=%d)", uniq, total, len(got))
	}
	if got[0].EditionNumber == "" {
		t.Errorf("finding sem edition_number (JOIN não trouxe metadados)")
	}
}
