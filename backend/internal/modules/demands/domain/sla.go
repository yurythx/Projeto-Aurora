package domain

import "time"

// slaDaysByStage é o prazo (dias corridos) que uma demanda deveria levar em
// cada etapa antes de virar pendência. As etapas 2 (Planejamento) e 6
// (Contabilidade) são esperas externas — é onde o SLA mais importa.
var slaDaysByStage = map[EtapaKanban]int{
	Etapa1ElaborarOF:       7,
	Etapa2TramitarPlan:     15,
	Etapa3EmitirOS:         5,
	Etapa4ExecucaoRecepcao: 30,
	Etapa5RelatorioPgto:    10,
	Etapa6Contabilidade:    15,
}

// SLADays devolve o prazo da etapa (0 = sem SLA definido).
func SLADays(etapa EtapaKanban) int { return slaDaysByStage[etapa] }

// SLAInfo resume a situação de prazo de uma demanda na etapa atual.
type SLAInfo struct {
	DaysInStage int        `json:"days_in_stage"`
	SLADays     int        `json:"sla_days"`
	DueAt       *time.Time `json:"due_at,omitempty"`
	Breached    bool       `json:"breached"`
}

// ComputeSLA calcula o SLAInfo de uma demanda parada em `etapa` desde
// `startedAt`, avaliado em `now`.
func ComputeSLA(etapa EtapaKanban, startedAt, now time.Time) SLAInfo {
	days := int(now.Sub(startedAt).Hours() / 24)
	if days < 0 {
		days = 0
	}
	sla := slaDaysByStage[etapa]
	info := SLAInfo{DaysInStage: days, SLADays: sla}
	if sla > 0 {
		due := startedAt.AddDate(0, 0, sla)
		info.DueAt = &due
		info.Breached = now.After(due)
	}
	return info
}

// OccurrenceSLABreach é o `tipo` de contract_occurrences gerado quando o SLA
// de uma etapa estoura.
const OccurrenceSLABreach = "SLA_ETAPA_ESTOURADO"
