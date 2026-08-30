package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/application"
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
	"github.com/yurythx/projeto-nova/internal/platform/auth"
)

// A fila de revisão (GET/PATCH/POST/DELETE .../rondonopolis/review-queue) só
// toca editionRepo + reindexer + logger — nada de pool/outbox. Então, ao
// contrário dos outros testes de handler deste módulo, estes rodam com um
// Service montado sobre fakes puros, sem TEST_DATABASE_URL.

type fakeReviewRepo struct {
	domain.EditionRepository // nil: qualquer método não sobrescrito dá panic

	queue    []domain.Finding
	byID     map[uuid.UUID]*domain.Finding
	notFound bool // força NotFound em Update/Get

	promoted []uuid.UUID
	acked    []uuid.UUID
	deleted  []uuid.UUID
	lastIn   domain.FindingReviewInput
}

func (f *fakeReviewRepo) ListFindingsForReview(_ context.Context, _ int) ([]domain.Finding, error) {
	return f.queue, nil
}

func (f *fakeReviewRepo) UpdateFindingReview(_ context.Context, id uuid.UUID, in domain.FindingReviewInput, _ *uuid.UUID) (*domain.Finding, error) {
	if f.notFound {
		return nil, apperrors.NotFound("finding não encontrado")
	}
	f.promoted = append(f.promoted, id)
	f.lastIn = in
	fnd := f.byID[id]
	if fnd == nil {
		fnd = &domain.Finding{ID: id, EditionID: 42}
	}
	fnd.ActType = in.ActType
	fnd.Confidence = in.Confidence
	return fnd, nil
}

func (f *fakeReviewRepo) GetFindingByID(_ context.Context, id uuid.UUID) (*domain.Finding, error) {
	if f.notFound {
		return nil, apperrors.NotFound("finding não encontrado")
	}
	if fnd := f.byID[id]; fnd != nil {
		return fnd, nil
	}
	return &domain.Finding{ID: id, EditionID: 42}, nil
}

func (f *fakeReviewRepo) AcknowledgeFinding(_ context.Context, id uuid.UUID, _ *uuid.UUID, _ string) error {
	if f.notFound {
		return apperrors.NotFound("finding não encontrado")
	}
	f.acked = append(f.acked, id)
	return nil
}

func (f *fakeReviewRepo) DeleteFinding(_ context.Context, id uuid.UUID) error {
	f.deleted = append(f.deleted, id)
	return nil
}

type spyReindexer struct{ editions []int64 }

func (s *spyReindexer) ReindexEdition(_ context.Context, id int64) (int, error) {
	s.editions = append(s.editions, id)
	return 1, nil
}

func newReviewHandlers(repo domain.EditionRepository, rx application.FindingReindexer) *Handlers {
	svc := application.NewService(nil, nil, nil, nil, nil, nil, nil, nil, testLogger())
	svc.SetEditionRepository(repo)
	if rx != nil {
		svc.WithReindexer(rx)
	}
	return NewHandlers(svc, testLogger())
}

// withURLParam devolve r com um route context do chi carregando {key}=val e
// uma identidade autenticada no contexto (o middleware de auth faria os
// dois), já que estes testes chamam o handler direto, sem montar as rotas.
func withURLParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = auth.WithIdentity(ctx, auth.Identity{Subject: uuid.NewString()})
	return r.WithContext(ctx)
}

func validPromoteBody() string {
	return `{"act_type":"EXONERACAO","confidence":"medium","servidor_nome":"Fulano de Tal"}`
}

func TestListReviewQueue_ReturnsItems(t *testing.T) {
	repo := &fakeReviewRepo{queue: []domain.Finding{
		{ID: uuid.New(), ActType: "OUTROS", Confidence: "low"},
		{ID: uuid.New(), ActType: "EXONERACAO", Confidence: "low"},
	}}
	h := newReviewHandlers(repo, nil)

	rec := httptest.NewRecorder()
	h.ListReviewQueue(rec, httptest.NewRequest(http.MethodGet, "/rondonopolis/review-queue?limit=50", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	got := decodeEnvelope[[]domain.Finding](t, rec.Body.Bytes())
	if len(got) != 2 {
		t.Fatalf("itens = %d, want 2", len(got))
	}
}

func TestPromoteReviewFinding_InvalidID_Returns400(t *testing.T) {
	h := newReviewHandlers(&fakeReviewRepo{}, nil)

	r := withURLParam(
		httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader(validPromoteBody())),
		"id", "nao-e-uuid",
	)
	rec := httptest.NewRecorder()
	h.PromoteReviewFinding(rec, r)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}

func TestPromoteReviewFinding_MissingNames_Returns400(t *testing.T) {
	repo := &fakeReviewRepo{}
	h := newReviewHandlers(repo, &spyReindexer{})

	body := `{"act_type":"EXONERACAO","confidence":"medium"}` // sem servidor_nome/empresa_nome
	r := withURLParam(
		httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader(body)),
		"id", uuid.New().String(),
	)
	rec := httptest.NewRecorder()
	h.PromoteReviewFinding(rec, r)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.promoted) != 0 {
		t.Errorf("não deveria ter chamado UpdateFindingReview com corpo inválido")
	}
}

func TestPromoteReviewFinding_Success_Returns200AndReindexes(t *testing.T) {
	id := uuid.New()
	repo := &fakeReviewRepo{byID: map[uuid.UUID]*domain.Finding{id: {ID: id, EditionID: 7}}}
	rx := &spyReindexer{}
	h := newReviewHandlers(repo, rx)

	r := withURLParam(
		httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader(validPromoteBody())),
		"id", id.String(),
	)
	rec := httptest.NewRecorder()
	h.PromoteReviewFinding(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.promoted) != 1 || repo.promoted[0] != id {
		t.Fatalf("UpdateFindingReview não recebeu o id esperado: %v", repo.promoted)
	}
	if repo.lastIn.ActType != "EXONERACAO" || repo.lastIn.Confidence != "medium" {
		t.Errorf("input normalizado errado: %+v", repo.lastIn)
	}
	if len(rx.editions) != 1 || rx.editions[0] != 7 {
		t.Errorf("edição não reindexada (esperado [7]): %v", rx.editions)
	}
}

func TestPromoteReviewFinding_NotFound_Returns404(t *testing.T) {
	repo := &fakeReviewRepo{notFound: true}
	h := newReviewHandlers(repo, &spyReindexer{})

	r := withURLParam(
		httptest.NewRequest(http.MethodPatch, "/x", strings.NewReader(validPromoteBody())),
		"id", uuid.New().String(),
	)
	rec := httptest.NewRecorder()
	h.PromoteReviewFinding(rec, r)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", rec.Code, rec.Body.String())
	}
}

func TestAckReviewFinding_Success_Returns200(t *testing.T) {
	id := uuid.New()
	repo := &fakeReviewRepo{}
	h := newReviewHandlers(repo, nil)

	r := withURLParam(
		httptest.NewRequest(http.MethodPost, "/x/ack", strings.NewReader(`{"review_note":"conferido"}`)),
		"id", id.String(),
	)
	rec := httptest.NewRecorder()
	h.AckReviewFinding(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.acked) != 1 || repo.acked[0] != id {
		t.Fatalf("AcknowledgeFinding não recebeu o id: %v", repo.acked)
	}
}

func TestAckReviewFinding_MalformedBody_Returns400(t *testing.T) {
	repo := &fakeReviewRepo{}
	h := newReviewHandlers(repo, nil)

	r := withURLParam(
		httptest.NewRequest(http.MethodPost, "/x/ack", strings.NewReader(`{not json`)),
		"id", uuid.New().String(),
	)
	rec := httptest.NewRecorder()
	h.AckReviewFinding(rec, r)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.acked) != 0 {
		t.Errorf("não deveria ter marcado nada com corpo malformado")
	}
}

func TestDiscardReviewFinding_Success_Returns200AndReindexes(t *testing.T) {
	id := uuid.New()
	repo := &fakeReviewRepo{byID: map[uuid.UUID]*domain.Finding{id: {ID: id, EditionID: 9}}}
	rx := &spyReindexer{}
	h := newReviewHandlers(repo, rx)

	r := withURLParam(httptest.NewRequest(http.MethodDelete, "/x", nil), "id", id.String())
	rec := httptest.NewRecorder()
	h.DiscardReviewFinding(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != id {
		t.Fatalf("DeleteFinding não recebeu o id: %v", repo.deleted)
	}
	if len(rx.editions) != 1 || rx.editions[0] != 9 {
		t.Errorf("edição não reindexada (esperado [9]): %v", rx.editions)
	}
}

func TestDiscardReviewFinding_NotFound_Returns404(t *testing.T) {
	repo := &fakeReviewRepo{notFound: true}
	h := newReviewHandlers(repo, &spyReindexer{})

	r := withURLParam(httptest.NewRequest(http.MethodDelete, "/x", nil), "id", uuid.New().String())
	rec := httptest.NewRecorder()
	h.DiscardReviewFinding(rec, r)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.deleted) != 0 {
		t.Errorf("não deveria ter deletado quando o finding não existe")
	}
}

func TestDiscardReviewFinding_InvalidID_Returns400(t *testing.T) {
	h := newReviewHandlers(&fakeReviewRepo{}, nil)

	r := withURLParam(httptest.NewRequest(http.MethodDelete, "/x", nil), "id", "xyz")
	rec := httptest.NewRecorder()
	h.DiscardReviewFinding(rec, r)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}
