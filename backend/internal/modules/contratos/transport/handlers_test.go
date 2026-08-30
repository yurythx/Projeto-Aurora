package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/modules/contratos/application"
	"github.com/yurythx/projeto-nova/internal/modules/contratos/domain"
)

// A camada de transporte de contratos não tinha teste próprio (achado da
// auditoria de 2026-08). O Service aqui só depende de domain.Repository
// (sem *pgxpool.Pool nos caminhos HTTP), então basta um fake de repo em
// memória para acionar os handlers de ponta a ponta.

type fakeRepo struct {
	domain.Repository
	byID      map[uuid.UUID]domain.Contrato
	createErr error
	list      []domain.Contrato
	listTotal int64
	byStatus  map[domain.Status][]domain.Contrato
	stats     domain.DashboardStats
	refs      []domain.DiarioRef
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byID: map[uuid.UUID]domain.Contrato{}, byStatus: map[domain.Status][]domain.Contrato{}}
}

func (f *fakeRepo) Create(_ context.Context, c domain.Contrato) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.byID[c.ID] = c
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Contrato, error) {
	c, ok := f.byID[id]
	if !ok {
		return domain.Contrato{}, apperrors.NotFound("contrato não encontrado")
	}
	return c, nil
}

func (f *fakeRepo) GetWithDetails(ctx context.Context, id uuid.UUID) (domain.Contrato, error) {
	return f.GetByID(ctx, id)
}

func (f *fakeRepo) List(_ context.Context, _ domain.ListParams) ([]domain.Contrato, int64, error) {
	return f.list, f.listTotal, nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, id uuid.UUID, s domain.Status) error {
	c := f.byID[id]
	c.Status = s
	f.byID[id] = c
	return nil
}

func (f *fakeRepo) GetDashboardStats(_ context.Context) (domain.DashboardStats, error) {
	return f.stats, nil
}

func (f *fakeRepo) AddDiarioRef(_ context.Context, ref domain.DiarioRef) error {
	f.refs = append(f.refs, ref)
	return nil
}

func (f *fakeRepo) KanbanCols(_ context.Context, _ int) (map[domain.Status][]domain.Contrato, error) {
	return f.byStatus, nil
}

// --- helpers ---------------------------------------------------------------

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func newTestHandlers(repo domain.Repository) *Handlers {
	return NewHandlers(application.NewService(repo, testLogger()), testLogger())
}

func withURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func decodeData[T any](t *testing.T, body []byte) T {
	t.Helper()
	var env struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v, body=%s", err, body)
	}
	return env.Data
}

// --- CreateContrato ------------------------------------------------------

func TestCreateContrato_Success(t *testing.T) {
	repo := newFakeRepo()
	h := newTestHandlers(repo)

	body := `{"numero":"010/2026","objeto":"Manutenção predial","contratante":"SMA","contratado":"ACME LTDA"}`
	rec := httptest.NewRecorder()
	h.CreateContrato(rec, httptest.NewRequest(http.MethodPost, "/contratos", bytes.NewBufferString(body)))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	got := decodeData[ContratoResponse](t, rec.Body.Bytes())
	if got.Numero != "010/2026" {
		t.Errorf("Numero = %q, want 010/2026", got.Numero)
	}
	if got.Status != string(domain.StatusRascunho) {
		t.Errorf("Status = %q, want rascunho (todo contrato nasce em rascunho)", got.Status)
	}
	if len(repo.byID) != 1 {
		t.Errorf("repo tem %d contratos, want 1", len(repo.byID))
	}
}

func TestCreateContrato_MissingNumero(t *testing.T) {
	h := newTestHandlers(newFakeRepo())
	rec := httptest.NewRecorder()
	h.CreateContrato(rec, httptest.NewRequest(http.MethodPost, "/contratos",
		bytes.NewBufferString(`{"objeto":"x","contratante":"y","contratado":"z"}`)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestCreateContrato_MissingObjeto(t *testing.T) {
	h := newTestHandlers(newFakeRepo())
	rec := httptest.NewRecorder()
	h.CreateContrato(rec, httptest.NewRequest(http.MethodPost, "/contratos",
		bytes.NewBufferString(`{"numero":"1","contratante":"y","contratado":"z"}`)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestCreateContrato_MalformedJSON(t *testing.T) {
	h := newTestHandlers(newFakeRepo())
	rec := httptest.NewRecorder()
	h.CreateContrato(rec, httptest.NewRequest(http.MethodPost, "/contratos",
		bytes.NewBufferString(`{"numero": :}`)))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// --- GetContrato --------------------------------------------------------

func TestGetContrato_Found(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.byID[id] = domain.Contrato{ID: id, Numero: "010/2026", Status: domain.StatusVigente}
	h := newTestHandlers(repo)

	rec := httptest.NewRecorder()
	h.GetContrato(rec, withURLParam(httptest.NewRequest(http.MethodGet, "/contratos/"+id.String(), nil), "id", id.String()))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	got := decodeData[ContratoResponse](t, rec.Body.Bytes())
	if got.ID != id {
		t.Errorf("ID = %s, want %s", got.ID, id)
	}
}

func TestGetContrato_NotFound(t *testing.T) {
	h := newTestHandlers(newFakeRepo())
	missing := uuid.New()
	rec := httptest.NewRecorder()
	h.GetContrato(rec, withURLParam(httptest.NewRequest(http.MethodGet, "/contratos/"+missing.String(), nil), "id", missing.String()))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestGetContrato_MalformedID(t *testing.T) {
	h := newTestHandlers(newFakeRepo())
	rec := httptest.NewRecorder()
	h.GetContrato(rec, withURLParam(httptest.NewRequest(http.MethodGet, "/contratos/x", nil), "id", "not-a-uuid"))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// --- ListContratos -----------------------------------------------------

func TestListContratos_ReturnsPaginatedEnvelope(t *testing.T) {
	repo := newFakeRepo()
	repo.list = []domain.Contrato{
		{ID: uuid.New(), Numero: "1/2026", Status: domain.StatusVigente},
		{ID: uuid.New(), Numero: "2/2026", Status: domain.StatusRascunho},
	}
	repo.listTotal = 42
	h := newTestHandlers(repo)

	rec := httptest.NewRecorder()
	h.ListContratos(rec, httptest.NewRequest(http.MethodGet, "/contratos?page=1&page_size=20", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	got := decodeData[ListContratosResponse](t, rec.Body.Bytes())
	if got.Total != 42 {
		t.Errorf("Total = %d, want 42", got.Total)
	}
	if got.Pages != 3 { // ceil(42/20)
		t.Errorf("Pages = %d, want 3", got.Pages)
	}
	if len(got.Data) != 2 {
		t.Errorf("Data = %d itens, want 2", len(got.Data))
	}
}

// --- UpdateStatus -----------------------------------------------------

func TestUpdateStatus_AllowedTransition(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.byID[id] = domain.Contrato{ID: id, Status: domain.StatusRascunho}
	h := newTestHandlers(repo)

	rec := httptest.NewRecorder()
	r := withURLParam(
		httptest.NewRequest(http.MethodPatch, "/contratos/"+id.String()+"/status",
			bytes.NewBufferString(`{"status":"em_analise"}`)),
		"id", id.String())
	h.UpdateStatus(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if repo.byID[id].Status != domain.StatusEmAnalise {
		t.Errorf("status persistido = %q, want em_analise", repo.byID[id].Status)
	}
}

func TestUpdateStatus_ForbiddenTransition(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	// rascunho -> vigente não é uma transição permitida (só rascunho ->
	// em_analise | cancelado).
	repo.byID[id] = domain.Contrato{ID: id, Status: domain.StatusRascunho}
	h := newTestHandlers(repo)

	rec := httptest.NewRecorder()
	r := withURLParam(
		httptest.NewRequest(http.MethodPatch, "/contratos/"+id.String()+"/status",
			bytes.NewBufferString(`{"status":"vigente"}`)),
		"id", id.String())
	h.UpdateStatus(rec, r)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422 (transição proibida)", rec.Code)
	}
	if repo.byID[id].Status != domain.StatusRascunho {
		t.Errorf("status mudou apesar da transição proibida: %q", repo.byID[id].Status)
	}
}

func TestUpdateStatus_ContratoNotFound(t *testing.T) {
	h := newTestHandlers(newFakeRepo())
	missing := uuid.New()
	rec := httptest.NewRecorder()
	r := withURLParam(
		httptest.NewRequest(http.MethodPatch, "/contratos/"+missing.String()+"/status",
			bytes.NewBufferString(`{"status":"em_analise"}`)),
		"id", missing.String())
	h.UpdateStatus(rec, r)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// --- KanbanView / AddDiarioRef / Dashboard ---------------------------

func TestKanbanView_ReturnsColumns(t *testing.T) {
	repo := newFakeRepo()
	repo.byStatus = map[domain.Status][]domain.Contrato{
		domain.StatusVigente: {{ID: uuid.New(), Numero: "1/2026", Status: domain.StatusVigente}},
	}
	h := newTestHandlers(repo)

	rec := httptest.NewRecorder()
	h.KanbanView(rec, httptest.NewRequest(http.MethodGet, "/contratos/kanban", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
}

func TestAddDiarioRef_Success(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.byID[id] = domain.Contrato{ID: id, Numero: "010/2026"}
	h := newTestHandlers(repo)

	rec := httptest.NewRecorder()
	r := withURLParam(
		httptest.NewRequest(http.MethodPost, "/contratos/"+id.String()+"/diario-refs",
			bytes.NewBufferString(`{"edition_number":"6262","tipo_evento":"EXTRATO","contexto":"..."}`)),
		"id", id.String())
	h.AddDiarioRef(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if len(repo.refs) != 1 || repo.refs[0].ContratoID != id {
		t.Errorf("ref não foi gravada corretamente: %+v", repo.refs)
	}
}

func TestAddDiarioRef_ContratoNotFound(t *testing.T) {
	h := newTestHandlers(newFakeRepo())
	missing := uuid.New()
	rec := httptest.NewRecorder()
	r := withURLParam(
		httptest.NewRequest(http.MethodPost, "/contratos/"+missing.String()+"/diario-refs",
			bytes.NewBufferString(`{"edition_number":"6262"}`)),
		"id", missing.String())
	h.AddDiarioRef(rec, r)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (contrato inexistente)", rec.Code)
	}
}

func TestGetDashboardStats_ReturnsStats(t *testing.T) {
	repo := newFakeRepo()
	repo.stats = domain.DashboardStats{TotalVigentes: 7, ValorTotalVigentes: 123456.78, ProximosVencimento: 2}
	h := newTestHandlers(repo)

	rec := httptest.NewRecorder()
	h.GetDashboardStats(rec, httptest.NewRequest(http.MethodGet, "/contratos/dashboard", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got := decodeData[domain.DashboardStats](t, rec.Body.Bytes())
	if got.TotalVigentes != 7 || got.ProximosVencimento != 2 {
		t.Errorf("stats = %+v, want TotalVigentes=7 ProximosVencimento=2", got)
	}
}
