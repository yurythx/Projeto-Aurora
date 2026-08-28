package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
)

type mockEditionRepo struct {
	editions map[string]*domain.Edition
	findings []domain.Finding
}

func newMockRepo() *mockEditionRepo {
	return &mockEditionRepo{
		editions: make(map[string]*domain.Edition),
		findings: make([]domain.Finding, 0),
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

func (m *mockEditionRepo) SaveFindingsTx(ctx context.Context, editionID int64, findings []domain.Finding) error {
	m.findings = append(m.findings, findings...)
	return m.UpdateEditionStatus(ctx, editionID, domain.EditionStatusCompleted, len(findings), nil)
}

func (m *mockEditionRepo) SearchFindings(ctx context.Context, query string, actType string, limit, offset int) ([]domain.Finding, int, error) {
	return m.findings, len(m.findings), nil
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
