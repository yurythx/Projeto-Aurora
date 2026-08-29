package application

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova/internal/modules/contratos/domain"
)

// fakeRepo embute domain.Repository (nil) e só implementa o que o casador
// toca — método não previsto = panic, o que é o comportamento desejado num
// teste unitário.
type fakeRepo struct {
	domain.Repository
	byStatus map[domain.Status][]domain.Contrato
	links    []domain.DiarioRef
	linkNew  map[string]bool // edition_number -> é nova?
	refs     map[uuid.UUID][]domain.DiarioRef
	alerts   map[string]bool // "contratoID|kind|refKey" já registrado
}

func (f *fakeRepo) ListByStatus(_ context.Context) (map[domain.Status][]domain.Contrato, error) {
	return f.byStatus, nil
}

func (f *fakeRepo) LinkDiarioRef(_ context.Context, ref domain.DiarioRef) (bool, error) {
	f.links = append(f.links, ref)
	isNew := f.linkNew == nil || f.linkNew[ref.EditionNumber]
	if isNew {
		if f.refs == nil {
			f.refs = map[uuid.UUID][]domain.DiarioRef{}
		}
		f.refs[ref.ContratoID] = append(f.refs[ref.ContratoID], ref)
	}
	return isNew, nil
}

func (f *fakeRepo) ListDiarioRefs(_ context.Context, id uuid.UUID) ([]domain.DiarioRef, error) {
	return f.refs[id], nil
}

func (f *fakeRepo) RecordAlertOnce(_ context.Context, id uuid.UUID, kind, refKey string) (bool, error) {
	if f.alerts == nil {
		f.alerts = map[string]bool{}
	}
	k := id.String() + "|" + kind + "|" + refKey
	if f.alerts[k] {
		return false, nil
	}
	f.alerts[k] = true
	return true, nil
}

type fakeSource struct{ out []DiarioFinding }

func (f fakeSource) FindForContract(_ context.Context, _, _ string) ([]DiarioFinding, error) {
	return f.out, nil
}

type spyEmitter struct{ events []string }

func (s *spyEmitter) EmitContratoEvent(_ context.Context, eventType, _ string, _ any) error {
	s.events = append(s.events, eventType)
	return nil
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func newMatcherSvc(repo *fakeRepo, src DiarioMatchSource, em ContratoEventEmitter) *Service {
	return &Service{repo: repo, logger: discardLogger(), diario: src, events: em}
}

func vigente(numero, cnpj string) domain.Contrato {
	return domain.Contrato{ID: uuid.New(), Numero: numero, CNPJ: cnpj, Status: domain.StatusVigente}
}

func TestRunDiarioMatch_LinksRefsAndEmitsLinkedEvent(t *testing.T) {
	c := vigente("012/2026", "12.345.678/0001-99")
	repo := &fakeRepo{
		byStatus: map[domain.Status][]domain.Contrato{domain.StatusVigente: {c}},
	}
	src := fakeSource{out: []DiarioFinding{
		{EditionNumber: "6301", ActType: "CONTRATO", RawContent: "EXTRATO DO CONTRATO 012/2026 ..."},
		{EditionNumber: "6301", ActType: "CONTRATO", RawContent: "duplicata na mesma edição"},
		{EditionNumber: "6305", ActType: "DESIGNACAO_FUNCAO", RawContent: "designa fiscal do contrato 012/2026"},
	}}
	em := &spyEmitter{}
	linked, alerts, err := newMatcherSvc(repo, src, em).RunDiarioMatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if linked != 2 {
		t.Errorf("esperava 2 refs (uma por edição, dedupe da 6301), got %d — links=%+v", linked, repo.links)
	}
	if alerts != 0 {
		t.Errorf("nenhum alerta esperado aqui, got %d", alerts)
	}
	if len(em.events) != 1 || em.events[0] != EventDiarioRefLinked {
		t.Errorf("esperava 1 evento %s, got %v", EventDiarioRefLinked, em.events)
	}
	if repo.links[0].TipoEvento != "CONTRATO" {
		t.Errorf("tipo_evento do 1º link = %q", repo.links[0].TipoEvento)
	}
}

func TestRunDiarioMatch_FiscalExoneradoAlert_Once(t *testing.T) {
	c := vigente("050/2026", "")
	repo := &fakeRepo{
		byStatus: map[domain.Status][]domain.Contrato{domain.StatusVigente: {c}},
		linkNew:  map[string]bool{},                                              // nada é ref nova
		refs:     map[uuid.UUID][]domain.DiarioRef{c.ID: {{EditionNumber: "x"}}}, // já tem ref -> sem alerta "sem vínculo"
	}
	src := fakeSource{out: []DiarioFinding{
		{EditionNumber: "6310", ActType: "EXONERACAO", ServidorNome: "MARIA SOUZA", RawContent: "exonera MARIA SOUZA, fiscal do contrato 050/2026"},
	}}
	em := &spyEmitter{}
	svc := newMatcherSvc(repo, src, em)

	_, alerts1, _ := svc.RunDiarioMatch(context.Background())
	if alerts1 != 1 {
		t.Fatalf("esperava 1 alerta de movimentação de pessoal, got %d", alerts1)
	}
	if len(em.events) != 1 || em.events[0] != EventFiscalAlert {
		t.Fatalf("esperava evento %s, got %v", EventFiscalAlert, em.events)
	}

	// segundo ciclo: mesma publicação -> RecordAlertOnce barra -> nada novo
	_, alerts2, _ := svc.RunDiarioMatch(context.Background())
	if alerts2 != 0 {
		t.Errorf("o mesmo alerta não pode reabrir no ciclo seguinte, got %d", alerts2)
	}
	if len(em.events) != 1 {
		t.Errorf("nenhum evento novo esperado no 2º ciclo, total=%v", em.events)
	}
}

func TestRunDiarioMatch_ContratoVigenteSemVinculo(t *testing.T) {
	c := vigente("077/2026", "")
	repo := &fakeRepo{
		byStatus: map[domain.Status][]domain.Contrato{domain.StatusVigente: {c}},
		refs:     map[uuid.UUID][]domain.DiarioRef{}, // sem refs
	}
	em := &spyEmitter{}
	_, alerts, _ := newMatcherSvc(repo, fakeSource{}, em).RunDiarioMatch(context.Background())
	if alerts != 1 || len(em.events) != 1 || em.events[0] != EventFiscalAlert {
		t.Errorf("esperava 1 alerta SEM_VINCULO_DIARIO, got alerts=%d events=%v", alerts, em.events)
	}
}

func TestRunDiarioMatch_ShortNumeroAndNoCNPJ_Skips(t *testing.T) {
	c := domain.Contrato{ID: uuid.New(), Numero: "12", CNPJ: "", Status: domain.StatusAprovado}
	repo := &fakeRepo{byStatus: map[domain.Status][]domain.Contrato{domain.StatusAprovado: {c}}}
	src := fakeSource{out: []DiarioFinding{{EditionNumber: "9999", ActType: "CONTRATO"}}}
	em := &spyEmitter{}
	linked, alerts, _ := newMatcherSvc(repo, src, em).RunDiarioMatch(context.Background())
	if linked != 0 || alerts != 0 || len(repo.links) != 0 {
		t.Errorf("número curto sem CNPJ deveria ser ignorado; linked=%d links=%+v", linked, repo.links)
	}
}

func TestRunDiarioMatch_NoSourceIsNoop(t *testing.T) {
	svc := &Service{repo: &fakeRepo{}, logger: discardLogger()}
	linked, alerts, err := svc.RunDiarioMatch(context.Background())
	if err != nil || linked != 0 || alerts != 0 {
		t.Errorf("sem DiarioMatchSource o casador é no-op: %d %d %v", linked, alerts, err)
	}
}

func TestSnippet_RuneSafeAroundAccents(t *testing.T) {
	// termo cercado de caracteres acentuados dos dois lados: um corte por
	// byte partiria um rune e geraria UTF-8 inválido.
	raw := "operação de manutenção referente à execução do contrato 012/2026 na secretaria de habitação e regularização fundiária"
	out := snippet(raw, "012/2026")
	if !utf8ValidString(out) {
		t.Fatalf("snippet produziu UTF-8 inválido: %q", out)
	}
	if !strings.Contains(out, "012/2026") {
		t.Errorf("snippet deveria conter o termo: %q", out)
	}

	// termo ausente + texto curto -> devolve o texto inteiro, válido.
	short := snippet("contratação simples", "9999")
	if !utf8ValidString(short) || short != "contratação simples" {
		t.Errorf("snippet curto inesperado: %q", short)
	}
}

func utf8ValidString(s string) bool {
	for _, r := range s {
		if r == '�' {
			return false
		}
	}
	return true
}
