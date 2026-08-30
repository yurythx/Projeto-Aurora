package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
)

type mockEditionRepo struct {
	editions map[string]*domain.Edition
	findings []domain.Finding

	pdfHashes        map[int64]string // editionID -> hash gravado
	saveFindingsCall int
	setHashCall      int
}

func newMockRepo() *mockEditionRepo {
	return &mockEditionRepo{
		editions:  make(map[string]*domain.Edition),
		findings:  make([]domain.Finding, 0),
		pdfHashes: make(map[int64]string),
	}
}

func (m *mockEditionRepo) SaveEdition(ctx context.Context, ed *domain.Edition) error {
	ed.ID = int64(len(m.editions) + 1)
	m.editions[ed.EditionNumber] = ed
	return nil
}

func (m *mockEditionRepo) GetEditionByNumber(ctx context.Context, editionNumber string) (*domain.Edition, error) {
	if ed, ok := m.editions[editionNumber]; ok {
		return ed, nil
	}
	return nil, nil
}

func (m *mockEditionRepo) GetPendingEditions(ctx context.Context, limit int) ([]domain.Edition, error) {
	out := make([]domain.Edition, 0)
	for _, ed := range m.editions {
		if ed.Status == domain.EditionStatusPending {
			out = append(out, *ed)
		}
	}
	return out, nil
}

func (m *mockEditionRepo) ListAllEditions(ctx context.Context, limit int) ([]domain.Edition, error) {
	out := make([]domain.Edition, 0)
	for _, ed := range m.editions {
		out = append(out, *ed)
	}
	return out, nil
}

func (m *mockEditionRepo) UpdateEditionStatus(ctx context.Context, editionID int64, status domain.EditionStatus, recordsCount int, errMsg *string) error {
	for _, ed := range m.editions {
		if ed.ID == editionID {
			ed.Status = status
			ed.RecordsCount = recordsCount
			ed.ErrorMessage = errMsg
			return nil
		}
	}
	return fmt.Errorf("edition not found")
}

func (m *mockEditionRepo) SetEditionPDFHash(ctx context.Context, editionID int64, sha256Hex string) error {
	m.setHashCall++
	m.pdfHashes[editionID] = sha256Hex
	for _, ed := range m.editions {
		if ed.ID == editionID {
			ed.PDFSHA256 = sha256Hex
		}
	}
	return nil
}

func (m *mockEditionRepo) SaveFindingsTx(ctx context.Context, editionID int64, findings []domain.Finding) error {
	m.saveFindingsCall++
	m.findings = append(m.findings, findings...)
	return m.UpdateEditionStatus(ctx, editionID, domain.EditionStatusCompleted, len(findings), nil)
}

func (m *mockEditionRepo) SearchFindings(ctx context.Context, query string, actType string, limit, offset int) ([]domain.Finding, int, error) {
	return m.findings, len(m.findings), nil
}

func (m *mockEditionRepo) ListFindingsForIndex(ctx context.Context, afterID uuid.UUID, limit int) ([]domain.Finding, error) {
	if afterID != uuid.Nil {
		return nil, nil // mock: entrega tudo numa página só
	}
	return m.findings, nil
}

func (m *mockEditionRepo) RequeueFailedEditions(ctx context.Context, maxRetries int) (int, error) {
	n := 0
	for _, ed := range m.editions {
		if ed.Status == domain.EditionStatusFailed && ed.RetryCount < maxRetries {
			ed.Status = domain.EditionStatusPending
			ed.RetryCount++
			n++
		}
	}
	return n, nil
}

func (m *mockEditionRepo) ListFindingsByEdition(ctx context.Context, editionID int64) ([]domain.Finding, error) {
	out := make([]domain.Finding, 0)
	for _, f := range m.findings {
		if f.EditionID == editionID {
			out = append(out, f)
		}
	}
	return out, nil
}

func (m *mockEditionRepo) ListFindingsForReview(ctx context.Context, limit int) ([]domain.Finding, error) {
	out := make([]domain.Finding, 0)
	for _, f := range m.findings {
		if f.Confidence == "low" {
			out = append(out, f)
		}
	}
	return out, nil
}

func (m *mockEditionRepo) GetFindingByID(ctx context.Context, id uuid.UUID) (*domain.Finding, error) {
	for i := range m.findings {
		if m.findings[i].ID == id {
			f := m.findings[i]
			return &f, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockEditionRepo) UpdateFindingReview(ctx context.Context, id uuid.UUID, in domain.FindingReviewInput, reviewedBy *uuid.UUID) (*domain.Finding, error) {
	for i := range m.findings {
		if m.findings[i].ID == id {
			m.findings[i].ActType = in.ActType
			m.findings[i].Confidence = in.Confidence
			f := m.findings[i]
			return &f, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockEditionRepo) AcknowledgeFinding(ctx context.Context, id uuid.UUID, reviewedBy *uuid.UUID, note string) error {
	return nil
}

func (m *mockEditionRepo) DeleteFinding(ctx context.Context, id uuid.UUID) error {
	for i := range m.findings {
		if m.findings[i].ID == id {
			m.findings = append(m.findings[:i], m.findings[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("not found")
}

func TestSyncWorkerPool_MockExecution(t *testing.T) {
	repo := newMockRepo()
	ed := &domain.Edition{
		EditionNumber: "6263",
		EditionDate:   time.Now(),
		PdfURL:        "https://www.rondonopolis.mt.gov.br/",
		Status:        domain.EditionStatusPending,
	}
	_ = repo.SaveEdition(context.Background(), ed)

	workerPool := NewSyncWorkerPool(repo, 2, nil)
	pending, _ := repo.GetPendingEditions(context.Background(), 10)

	if len(pending) != 1 {
		t.Fatalf("expected 1 pending edition, got %d", len(pending))
	}

	// Execução isolada do mock de pool
	err := workerPool.ProcessBatch(context.Background(), pending)
	if err != nil {
		t.Fatalf("unexpected error running worker pool: %v", err)
	}

	updated, _ := repo.GetEditionByNumber(context.Background(), "6263")
	if updated == nil {
		t.Fatal("expected edition to exist")
	}
}

// PDF mínimo (poppler reconstrói o xref e extrai texto vazio, saindo com 0).
const minimalPDF = "%PDF-1.1\n" +
	"1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n" +
	"2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n" +
	"3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 300 144]>>endobj\n" +
	"trailer<</Size 4/Root 1 0 R>>\n%%EOF\n"

// TestSyncWorkerPool_BinaryHashIdempotency cobre CLAUDE.md §1: o worker calcula
// o SHA-256 dos bytes do PDF e, numa reprocessagem com o MESMO binário, pula
// pdftotext-parser-persist e só reconfirma COMPLETED.
func TestSyncWorkerPool_BinaryHashIdempotency(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext ausente; pulando teste de ingestão real")
	}

	pdfBytes := []byte(minimalPDF)
	sum := sha256.Sum256(pdfBytes)
	wantHash := hex.EncodeToString(sum[:])

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer srv.Close()

	repo := newMockRepo()
	ed := &domain.Edition{EditionNumber: "9001", PdfURL: srv.URL, Status: domain.EditionStatusPending}
	_ = repo.SaveEdition(context.Background(), ed)

	pool := NewSyncWorkerPool(repo, 1, nil)

	// 1ª passagem: edição nova, sem hash → ingestão completa + hash gravado.
	pending, _ := repo.GetPendingEditions(context.Background(), 10)
	if err := pool.ProcessBatch(context.Background(), pending); err != nil {
		t.Fatalf("1ª passagem: %v", err)
	}
	if repo.saveFindingsCall != 1 {
		t.Fatalf("1ª passagem: SaveFindingsTx chamado %d× (esperado 1)", repo.saveFindingsCall)
	}
	if got := repo.pdfHashes[ed.ID]; got != wantHash {
		t.Fatalf("hash gravado = %q, esperado %q", got, wantHash)
	}
	stored, _ := repo.GetEditionByNumber(context.Background(), "9001")
	if stored.Status != domain.EditionStatusCompleted {
		t.Fatalf("status = %s, esperado COMPLETED", stored.Status)
	}

	// 2ª passagem: mesmo binário, hash já armazenado → atalho de idempotência.
	stored.Status = domain.EditionStatusPending // simula re-enfileiramento
	pending2, _ := repo.GetPendingEditions(context.Background(), 10)
	if err := pool.ProcessBatch(context.Background(), pending2); err != nil {
		t.Fatalf("2ª passagem: %v", err)
	}
	if repo.saveFindingsCall != 1 {
		t.Fatalf("2ª passagem: SaveFindingsTx re-executado (%d×) — atalho não pegou", repo.saveFindingsCall)
	}
	if repo.setHashCall != 1 {
		t.Fatalf("2ª passagem: hash re-gravado (%d×) — atalho não pegou", repo.setHashCall)
	}
	final, _ := repo.GetEditionByNumber(context.Background(), "9001")
	if final.Status != domain.EditionStatusCompleted {
		t.Fatalf("2ª passagem: status = %s, esperado COMPLETED", final.Status)
	}
}
