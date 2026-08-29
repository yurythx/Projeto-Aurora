package transport

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova/internal/modules/demands/application"
	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

// A demanda parada além do SLA da etapa serializa sla.breached=true e
// carrega o due_at calculado a partir de etapa_started_at.
func TestToDemandResponseAt_SLA(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	d := domain.MonthlyDemand{
		ID:             uuid.New(),
		ContratoID:     uuid.New(),
		AnoMes:         "2026-08",
		Etapa:          domain.Etapa2TramitarPlan, // SLA 15 dias
		StatusEtapa:    domain.StatusPendente,
		EtapaStartedAt: now.AddDate(0, 0, -20),
	}

	resp := toDemandResponseAt(d, now)
	if resp.SLA == nil {
		t.Fatal("SLA não deveria ser nil quando etapa_started_at está preenchido")
	}
	if !resp.SLA.Breached {
		t.Errorf("20 dias na etapa 2 (SLA 15) deveria estar breached: %+v", resp.SLA)
	}
	if resp.SLA.DaysInStage != 20 || resp.SLA.SLADays != 15 {
		t.Errorf("dias/sla inesperados: %+v", resp.SLA)
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)
	for _, want := range []string{`"sla":`, `"breached":true`, `"due_at":`, `"days_in_stage":20`} {
		if !strings.Contains(js, want) {
			t.Errorf("JSON não contém %q: %s", want, js)
		}
	}
}

// Sem etapa_started_at (zero value) não há bloco sla no JSON.
func TestToDemandResponseAt_SLAOmittedWhenNoStart(t *testing.T) {
	d := domain.MonthlyDemand{ID: uuid.New(), Etapa: domain.Etapa1ElaborarOF}
	b, _ := json.Marshal(toDemandResponseAt(d, time.Now()))
	if strings.Contains(string(b), `"sla":`) {
		t.Errorf("sla não deveria aparecer sem etapa_started_at: %s", b)
	}
}

// O checklist da próxima etapa serializa em next_requirements com o
// bloqueio "falta:" para uma demanda na Etapa 1 sem documentos.
func TestNextRequirements_JSONShape(t *testing.T) {
	d := domain.MonthlyDemand{
		ID:             uuid.New(),
		Etapa:          domain.Etapa1ElaborarOF,
		EtapaStartedAt: time.Now(),
	}
	resp := toDemandResponseAt(d, time.Now())
	chk := application.CheckAdvance(d, d.Etapa+1, time.Now())
	resp.NextRequirements = &chk

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)
	if !strings.Contains(js, `"next_requirements":`) || !strings.Contains(js, `"can_advance":false`) {
		t.Errorf("next_requirements ausente/incorreto: %s", js)
	}
	if !strings.Contains(js, `"blocking":`) || !strings.Contains(strings.ToLower(js), "falta") {
		t.Errorf("esperava bloqueio 'falta:' no checklist: %s", js)
	}
}
