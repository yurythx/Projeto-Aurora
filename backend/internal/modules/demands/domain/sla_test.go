package domain

import (
	"testing"
	"time"
)

func TestComputeSLA(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

	// Etapa 2 (SLA 15 dias), parada há 10 dias -> dentro do prazo.
	in := ComputeSLA(Etapa2TramitarPlan, now.AddDate(0, 0, -10), now)
	if in.DaysInStage != 10 || in.SLADays != 15 || in.Breached {
		t.Errorf("dentro do prazo: %+v", in)
	}
	if in.DueAt == nil || !in.DueAt.Equal(now.AddDate(0, 0, 5)) {
		t.Errorf("DueAt esperado %v, got %v", now.AddDate(0, 0, 5), in.DueAt)
	}

	// Parada há 20 dias -> estourou.
	out := ComputeSLA(Etapa2TramitarPlan, now.AddDate(0, 0, -20), now)
	if !out.Breached {
		t.Errorf("20 dias na etapa 2 (SLA 15) deveria estar Breached: %+v", out)
	}

	// startedAt no futuro -> 0 dias, não estoura.
	fut := ComputeSLA(Etapa1ElaborarOF, now.AddDate(0, 0, 3), now)
	if fut.DaysInStage != 0 || fut.Breached {
		t.Errorf("startedAt no futuro: %+v", fut)
	}
}

func TestSLADays_AllStagesDefined(t *testing.T) {
	for e := Etapa1ElaborarOF; e <= Etapa6Contabilidade; e++ {
		if SLADays(e) <= 0 {
			t.Errorf("etapa %d sem SLA definido", e)
		}
	}
}
