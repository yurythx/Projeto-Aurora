package application

import (
	"context"
	"testing"

	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/gazette"
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
)

// stubEditionRepo embute a interface (nil) e só implementa o que a fila de
// revisão toca — qualquer outro método chamado por engano dá panic no teste.
type stubEditionRepo struct {
	domain.EditionRepository
	findings map[uuid.UUID]*domain.Finding
	deleted  []uuid.UUID
	acked    []uuid.UUID
	lastIn   domain.FindingReviewInput
}

func (s *stubEditionRepo) GetFindingByID(_ context.Context, id uuid.UUID) (*domain.Finding, error) {
	f, ok := s.findings[id]
	if !ok {
		return nil, apperrors.NotFound("not found")
	}
	return f, nil
}

func (s *stubEditionRepo) UpdateFindingReview(_ context.Context, id uuid.UUID, in domain.FindingReviewInput, _ *uuid.UUID) (*domain.Finding, error) {
	f, ok := s.findings[id]
	if !ok {
		return nil, apperrors.NotFound("not found")
	}
	s.lastIn = in
	f.ActType = in.ActType
	f.Confidence = in.Confidence
	f.ServidorNome = in.ServidorNome
	return f, nil
}

func (s *stubEditionRepo) AcknowledgeFinding(_ context.Context, id uuid.UUID, _ *uuid.UUID, _ string) error {
	if _, ok := s.findings[id]; !ok {
		return apperrors.NotFound("not found")
	}
	s.acked = append(s.acked, id)
	return nil
}

func (s *stubEditionRepo) DeleteFinding(_ context.Context, id uuid.UUID) error {
	if _, ok := s.findings[id]; !ok {
		return apperrors.NotFound("not found")
	}
	s.deleted = append(s.deleted, id)
	return nil
}

type spyReindexer struct{ editions []int64 }

func (r *spyReindexer) ReindexEdition(_ context.Context, id int64) (int, error) {
	r.editions = append(r.editions, id)
	return 1, nil
}

func newReviewFixture() (*Service, *stubEditionRepo, *spyReindexer, uuid.UUID) {
	id := uuid.New()
	repo := &stubEditionRepo{findings: map[uuid.UUID]*domain.Finding{
		id: {ID: id, EditionID: 42, ActType: "OUTROS", Confidence: gazette.ConfidenceLow, ServidorNome: strPtr("fulano")},
	}}
	rx := &spyReindexer{}
	svc := &Service{editionRepo: repo, reindexer: rx, logger: testLogger()}
	return svc, repo, rx, id
}

func TestPromoteFinding_UpdatesReindexesAndValidates(t *testing.T) {
	svc, repo, rx, id := newReviewFixture()

	out, err := svc.PromoteFinding(context.Background(), id, domain.FindingReviewInput{
		ActType:      "nomeacao_comissionado", // minúsculo de propósito
		Confidence:   "medium",
		ServidorNome: strPtr("  JOÃO DA SILVA  "),
	}, nil)
	if err != nil {
		t.Fatalf("PromoteFinding: %v", err)
	}
	if out.Confidence != "medium" || out.ActType != gazette.ActNomeacaoComissionado {
		t.Errorf("finding não normalizado: %+v", out)
	}
	if repo.lastIn.ServidorNome == nil || *repo.lastIn.ServidorNome != "JOÃO DA SILVA" {
		t.Errorf("servidor_nome não foi trimado: %v", repo.lastIn.ServidorNome)
	}
	if len(rx.editions) != 1 || rx.editions[0] != 42 {
		t.Errorf("esperava reindex da edição 42, got %v", rx.editions)
	}
}

func TestPromoteFinding_RejectsBadConfidenceAndActType(t *testing.T) {
	svc, _, rx, id := newReviewFixture()

	if _, err := svc.PromoteFinding(context.Background(), id, domain.FindingReviewInput{
		ActType: gazette.ActNomeacaoEfetivo, Confidence: "low", ServidorNome: strPtr("x y"),
	}, nil); err == nil {
		t.Error("confidence=low deveria ser rejeitado ao promover")
	}
	if _, err := svc.PromoteFinding(context.Background(), id, domain.FindingReviewInput{
		ActType: "BANANA", Confidence: "medium", ServidorNome: strPtr("x y"),
	}, nil); err == nil {
		t.Error("act_type inválido deveria ser rejeitado")
	}
	if _, err := svc.PromoteFinding(context.Background(), id, domain.FindingReviewInput{
		ActType: gazette.ActContrato, Confidence: "medium",
	}, nil); err == nil {
		t.Error("sem servidor_nome nem empresa_nome deveria ser rejeitado")
	}
	if len(rx.editions) != 0 {
		t.Errorf("nenhum reindex deveria ocorrer em erro de validação, got %v", rx.editions)
	}
}

func TestDiscardFinding_DeletesAndReindexes(t *testing.T) {
	svc, repo, rx, id := newReviewFixture()
	if err := svc.DiscardFinding(context.Background(), id, nil); err != nil {
		t.Fatalf("DiscardFinding: %v", err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != id {
		t.Errorf("finding não foi deletado: %v", repo.deleted)
	}
	if len(rx.editions) != 1 || rx.editions[0] != 42 {
		t.Errorf("esperava reindex da edição 42 após descarte, got %v", rx.editions)
	}
}

func TestAcknowledgeFinding_StampsOnly(t *testing.T) {
	svc, repo, rx, id := newReviewFixture()
	if err := svc.AcknowledgeFinding(context.Background(), id, "é low mas confere", nil); err != nil {
		t.Fatalf("AcknowledgeFinding: %v", err)
	}
	if len(repo.acked) != 1 {
		t.Errorf("finding não foi marcado como revisado: %v", repo.acked)
	}
	if len(rx.editions) != 0 {
		t.Errorf("ack não deve reindexar (finding continua fora da busca), got %v", rx.editions)
	}
}
