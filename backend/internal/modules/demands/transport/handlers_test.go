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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/modules/demands/application"
	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

// --- Fakes -------------------------------------------------------------
//
// A camada de transporte não tinha teste próprio (achado da auditoria de
// 2026-08). Estes fakes acionam os handlers de ponta a ponta —
// HTTP -> application.Service real -> fakes — sem Postgres/MinIO, graças à
// injeção por interface (TxRunner/OutboxWriter/Repository) que a refação
// do Service abriu.

// fakeRepo embute domain.Repository e só implementa o que os handlers
// testados exercitam; qualquer método não sobrescrito que for chamado
// causa panic (nil interface) — proposital, denuncia teste tocando algo
// não previsto.
type fakeRepo struct {
	domain.Repository
	demands   map[uuid.UUID]domain.MonthlyDemand
	kanban    map[domain.EtapaKanban][]domain.MonthlyDemand
	createErr error
	updated   []domain.MonthlyDemand // registra Update/UpdateTx
	docs      []domain.DemandDocument
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{demands: map[uuid.UUID]domain.MonthlyDemand{}, kanban: map[domain.EtapaKanban][]domain.MonthlyDemand{}}
}

func (f *fakeRepo) Create(_ context.Context, d domain.MonthlyDemand) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.demands[d.ID] = d
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id uuid.UUID) (domain.MonthlyDemand, error) {
	d, ok := f.demands[id]
	if !ok {
		return domain.MonthlyDemand{}, apperrors.NotFound("demanda não encontrada")
	}
	return d, nil
}

func (f *fakeRepo) Update(_ context.Context, d domain.MonthlyDemand) error {
	f.demands[d.ID] = d
	f.updated = append(f.updated, d)
	return nil
}

// UpdateTx satisfaz o txUpdater opcional que o Service prefere quando o
// repo o implementa (mantém o caminho atômico exercitado nos testes).
func (f *fakeRepo) UpdateTx(_ context.Context, _ pgx.Tx, d domain.MonthlyDemand) error {
	return f.Update(context.Background(), d)
}

func (f *fakeRepo) ListAllKanban(_ context.Context) (map[domain.EtapaKanban][]domain.MonthlyDemand, error) {
	return f.kanban, nil
}

func (f *fakeRepo) AddDocument(_ context.Context, doc domain.DemandDocument) error {
	f.docs = append(f.docs, doc)
	return nil
}

// Stubs vazios para os métodos que o dashboard/ocorrências percorrem —
// suficiente para exercitar os handlers de leitura.
func (f *fakeRepo) CountByEtapa(context.Context) (map[domain.EtapaKanban]int, error) {
	return map[domain.EtapaKanban]int{domain.Etapa1ElaborarOF: 3}, nil
}
func (f *fakeRepo) ListStale(context.Context, time.Time) ([]domain.MonthlyDemand, error) {
	return nil, nil
}
func (f *fakeRepo) ListOpenOccurrences(context.Context, int) ([]domain.Occurrence, error) {
	return []domain.Occurrence{{ID: uuid.New(), Descricao: "pendência de teste"}}, nil
}
func (f *fakeRepo) ListExpiringCertidoes(context.Context, time.Time, int) ([]domain.ExpiringCertidao, error) {
	return nil, nil
}

// fakeTx roda fn direto com um pgx.Tx nil — os fakes de repo/outbox não
// desreferenciam o tx, então basta.
type fakeTx struct{ calls int }

func (f *fakeTx) WithTx(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	f.calls++
	return fn(ctx, nil)
}

// fakeOutbox só registra o que foi escrito, para o teste conferir que a
// mudança de etapa publicou o evento (e que um avanço bloqueado NÃO
// publicou nada).
type fakeOutbox struct {
	writes []outboxWrite
}

type outboxWrite struct {
	eventType     string
	aggregateType string
	aggregateID   string
}

func (f *fakeOutbox) Write(_ context.Context, _ pgx.Tx, eventType, aggregateType, aggregateID string, _ uuid.UUID, _ any) error {
	f.writes = append(f.writes, outboxWrite{eventType, aggregateType, aggregateID})
	return nil
}

// --- Helpers ---------------------------------------------------------------

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestHandlers(repo domain.Repository, tx application.TxRunner, ob application.OutboxWriter) *Handlers {
	svc := application.NewServiceWithDeps(tx, repo, ob, nil, nil, "test-bucket", testLogger())
	return NewHandlers(svc, testLogger())
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

// --- CreateDemand --------------------------------------------------------

func TestCreateDemand_Success(t *testing.T) {
	repo := newFakeRepo()
	h := newTestHandlers(repo, &fakeTx{}, &fakeOutbox{})

	contratoID := uuid.New()
	body := `{"contrato_id":"` + contratoID.String() + `","ano_mes":"2026-08","observacoes":"teste"}`
	rec := httptest.NewRecorder()
	h.CreateDemand(rec, httptest.NewRequest(http.MethodPost, "/demands", bytes.NewBufferString(body)))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	got := decodeData[DemandResponse](t, rec.Body.Bytes())
	if got.ContratoID != contratoID {
		t.Errorf("ContratoID = %s, want %s", got.ContratoID, contratoID)
	}
	if got.Etapa != int(domain.Etapa1ElaborarOF) {
		t.Errorf("Etapa = %d, want 1 (toda demanda nasce na etapa 1)", got.Etapa)
	}
	if len(repo.demands) != 1 {
		t.Errorf("repo tem %d demandas, want 1", len(repo.demands))
	}
}

func TestCreateDemand_MissingContratoID(t *testing.T) {
	h := newTestHandlers(newFakeRepo(), &fakeTx{}, &fakeOutbox{})
	rec := httptest.NewRecorder()
	h.CreateDemand(rec, httptest.NewRequest(http.MethodPost, "/demands",
		bytes.NewBufferString(`{"ano_mes":"2026-08"}`)))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestCreateDemand_MissingAnoMes(t *testing.T) {
	h := newTestHandlers(newFakeRepo(), &fakeTx{}, &fakeOutbox{})
	rec := httptest.NewRecorder()
	h.CreateDemand(rec, httptest.NewRequest(http.MethodPost, "/demands",
		bytes.NewBufferString(`{"contrato_id":"`+uuid.New().String()+`"}`)))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestCreateDemand_MalformedJSON(t *testing.T) {
	h := newTestHandlers(newFakeRepo(), &fakeTx{}, &fakeOutbox{})
	rec := httptest.NewRecorder()
	h.CreateDemand(rec, httptest.NewRequest(http.MethodPost, "/demands",
		bytes.NewBufferString(`{"contrato_id": not-json`)))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// --- GetByID -----------------------------------------------------------

func TestGetByID_Found(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.demands[id] = domain.MonthlyDemand{ID: id, AnoMes: "2026-08", Etapa: domain.Etapa3EmitirOS}
	h := newTestHandlers(repo, &fakeTx{}, &fakeOutbox{})

	rec := httptest.NewRecorder()
	r := withURLParam(httptest.NewRequest(http.MethodGet, "/demands/"+id.String(), nil), "id", id.String())
	h.GetByID(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	got := decodeData[DemandResponse](t, rec.Body.Bytes())
	if got.ID != id || got.Etapa != int(domain.Etapa3EmitirOS) {
		t.Errorf("got id=%s etapa=%d, want id=%s etapa=3", got.ID, got.Etapa, id)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	h := newTestHandlers(newFakeRepo(), &fakeTx{}, &fakeOutbox{})
	missing := uuid.New()
	rec := httptest.NewRecorder()
	r := withURLParam(httptest.NewRequest(http.MethodGet, "/demands/"+missing.String(), nil), "id", missing.String())
	h.GetByID(rec, r)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestGetByID_MalformedID(t *testing.T) {
	h := newTestHandlers(newFakeRepo(), &fakeTx{}, &fakeOutbox{})
	rec := httptest.NewRecorder()
	r := withURLParam(httptest.NewRequest(http.MethodGet, "/demands/xxx", nil), "id", "not-a-uuid")
	h.GetByID(rec, r)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (id malformado é erro do cliente)", rec.Code)
	}
}

// --- KanbanView ------------------------------------------------------------

func TestKanbanView_GroupsIntoSixColumns(t *testing.T) {
	repo := newFakeRepo()
	repo.kanban = map[domain.EtapaKanban][]domain.MonthlyDemand{
		domain.Etapa1ElaborarOF: {
			{ID: uuid.New(), Etapa: domain.Etapa1ElaborarOF, AnoMes: "2026-08"},
			{ID: uuid.New(), Etapa: domain.Etapa1ElaborarOF, AnoMes: "2026-08"},
		},
		domain.Etapa3EmitirOS: {
			{ID: uuid.New(), Etapa: domain.Etapa3EmitirOS, AnoMes: "2026-08"},
		},
	}
	h := newTestHandlers(repo, &fakeTx{}, &fakeOutbox{})

	rec := httptest.NewRecorder()
	h.KanbanView(rec, httptest.NewRequest(http.MethodGet, "/demands/kanban", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct {
			Columns []struct {
				Status string `json:"status"`
				Total  int    `json:"total"`
			} `json:"columns"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(env.Data.Columns) != 6 {
		t.Fatalf("colunas = %d, want 6 (sempre as 6 etapas)", len(env.Data.Columns))
	}
	if env.Data.Columns[0].Total != 2 || env.Data.Columns[2].Total != 1 {
		t.Errorf("totais errados: etapa1=%d (want 2), etapa3=%d (want 1)",
			env.Data.Columns[0].Total, env.Data.Columns[2].Total)
	}
	if env.Data.Columns[1].Total != 0 {
		t.Errorf("etapa 2 sem demanda deveria ter total 0, veio %d", env.Data.Columns[1].Total)
	}
}

// --- MoveKanbanCard ------------------------------------------------------

func TestMoveKanbanCard_BlockedWhenDocsMissing(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	// Etapa 1 sem nenhum documento anexado -> avançar para 2 exige
	// OF_PRE_EMPENHO + OFICIO_PLANEJAMENTO (RequiredDocsForAdvance).
	repo.demands[id] = domain.MonthlyDemand{ID: id, Etapa: domain.Etapa1ElaborarOF}
	ob := &fakeOutbox{}
	tx := &fakeTx{}
	h := newTestHandlers(repo, tx, ob)

	rec := httptest.NewRecorder()
	r := withURLParam(
		httptest.NewRequest(http.MethodPatch, "/demands/"+id.String()+"/etapa",
			bytes.NewBufferString(`{"target_etapa":2}`)),
		"id", id.String())
	h.MoveKanbanCard(rec, r)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (avanço bloqueado por compliance), body=%s", rec.Code, rec.Body.String())
	}
	if len(ob.writes) != 0 {
		t.Errorf("outbox escreveu %d evento(s) num avanço bloqueado — deveria ser 0", len(ob.writes))
	}
	if tx.calls != 0 {
		t.Errorf("transação abriu %d vez(es) num avanço bloqueado — deveria ser 0", tx.calls)
	}
	if repo.demands[id].Etapa != domain.Etapa1ElaborarOF {
		t.Errorf("etapa mudou para %d apesar do bloqueio", repo.demands[id].Etapa)
	}
}

func TestMoveKanbanCard_RegressionIsAllowedAndPublishesEvent(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.demands[id] = domain.MonthlyDemand{ID: id, Etapa: domain.Etapa3EmitirOS, ContratoID: uuid.New()}
	ob := &fakeOutbox{}
	tx := &fakeTx{}
	h := newTestHandlers(repo, tx, ob)

	rec := httptest.NewRecorder()
	r := withURLParam(
		httptest.NewRequest(http.MethodPatch, "/demands/"+id.String()+"/etapa",
			bytes.NewBufferString(`{"target_etapa":2}`)), // regressão 3 -> 2
		"id", id.String())
	h.MoveKanbanCard(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (regressão sempre pode), body=%s", rec.Code, rec.Body.String())
	}
	if repo.demands[id].Etapa != domain.Etapa2TramitarPlan {
		t.Errorf("etapa = %d, want 2", repo.demands[id].Etapa)
	}
	if len(ob.writes) != 1 {
		t.Fatalf("outbox escreveu %d evento(s), want 1", len(ob.writes))
	}
	if ob.writes[0].eventType != "demand.etapa_changed" || ob.writes[0].aggregateType != "demand" {
		t.Errorf("evento errado: %+v", ob.writes[0])
	}
	if tx.calls != 1 {
		t.Errorf("transação abriu %d vez(es), want 1 (persistência + outbox atômicos)", tx.calls)
	}
}

func TestMoveKanbanCard_InvalidTargetEtapa(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.demands[id] = domain.MonthlyDemand{ID: id, Etapa: domain.Etapa1ElaborarOF}
	h := newTestHandlers(repo, &fakeTx{}, &fakeOutbox{})

	rec := httptest.NewRecorder()
	r := withURLParam(
		httptest.NewRequest(http.MethodPatch, "/demands/"+id.String()+"/etapa",
			bytes.NewBufferString(`{"target_etapa":99}`)),
		"id", id.String())
	h.MoveKanbanCard(rec, r)

	if rec.Code < 400 || rec.Code >= 500 {
		t.Errorf("status = %d, want um 4xx para etapa inválida", rec.Code)
	}
}

func TestMoveKanbanCard_MalformedID(t *testing.T) {
	h := newTestHandlers(newFakeRepo(), &fakeTx{}, &fakeOutbox{})
	rec := httptest.NewRecorder()
	r := withURLParam(
		httptest.NewRequest(http.MethodPatch, "/demands/x/etapa", bytes.NewBufferString(`{"target_etapa":2}`)),
		"id", "not-a-uuid")
	h.MoveKanbanCard(rec, r)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// --- NextRequirements ----------------------------------------------------

func TestNextRequirements_ListsMissingDocsForStageOne(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	repo.demands[id] = domain.MonthlyDemand{ID: id, Etapa: domain.Etapa1ElaborarOF}
	h := newTestHandlers(repo, &fakeTx{}, &fakeOutbox{})

	rec := httptest.NewRecorder()
	r := withURLParam(httptest.NewRequest(http.MethodGet, "/demands/"+id.String()+"/requirements", nil), "id", id.String())
	h.NextRequirements(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	chk := decodeData[application.StageCheck](t, rec.Body.Bytes())
	if chk.CanAdvance {
		t.Error("CanAdvance = true, want false (etapa 1 sem documentos)")
	}
	if chk.FromEtapa != 1 || chk.ToEtapa != 2 {
		t.Errorf("From/To = %d/%d, want 1/2", chk.FromEtapa, chk.ToEtapa)
	}
	if len(chk.Docs) != 2 {
		t.Errorf("Docs = %d, want 2 (OF_PRE_EMPENHO + OFICIO_PLANEJAMENTO)", len(chk.Docs))
	}
}

func TestNextRequirements_MalformedID(t *testing.T) {
	h := newTestHandlers(newFakeRepo(), &fakeTx{}, &fakeOutbox{})
	rec := httptest.NewRecorder()
	r := withURLParam(httptest.NewRequest(http.MethodGet, "/demands/x/requirements", nil), "id", "nope")
	h.NextRequirements(rec, r)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// --- Dashboard / ListOccurrences / GetHistory --------------------------

func TestDashboard_ReturnsFunnel(t *testing.T) {
	h := newTestHandlers(newFakeRepo(), &fakeTx{}, &fakeOutbox{})
	rec := httptest.NewRecorder()
	h.Dashboard(rec, httptest.NewRequest(http.MethodGet, "/demands/dashboard", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data application.DashboardData `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(env.Data.Funnel) == 0 {
		t.Error("Funnel vazio — esperava as etapas do funil")
	}
}

func TestListOccurrences_ReturnsOpenItems(t *testing.T) {
	h := newTestHandlers(newFakeRepo(), &fakeTx{}, &fakeOutbox{})
	rec := httptest.NewRecorder()
	h.ListOccurrences(rec, httptest.NewRequest(http.MethodGet, "/demands/occurrences", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got := decodeData[[]domain.Occurrence](t, rec.Body.Bytes())
	if len(got) != 1 {
		t.Errorf("got %d ocorrência(s), want 1", len(got))
	}
}

func TestGetHistory_EmptyWithoutAuditReader(t *testing.T) {
	// Sem audit.Reader injetado, o serviço devolve lista vazia (não erro).
	h := newTestHandlers(newFakeRepo(), &fakeTx{}, &fakeOutbox{})
	id := uuid.New()
	rec := httptest.NewRecorder()
	r := withURLParam(httptest.NewRequest(http.MethodGet, "/demands/"+id.String()+"/history", nil), "id", id.String())
	h.GetHistory(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
}
