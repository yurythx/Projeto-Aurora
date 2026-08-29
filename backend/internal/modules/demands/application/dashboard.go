package application

import (
	"context"
	"fmt"
	"time"

	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

// FunnelStage é uma coluna do funil (quantas demandas em cada etapa).
type FunnelStage struct {
	Etapa int    `json:"etapa"`
	Label string `json:"label"`
	Total int    `json:"total"`
}

// SLABreachRow resume uma demanda que estourou o SLA da etapa atual.
type SLABreachRow struct {
	DemandaID      string `json:"demanda_id"`
	ContratoNumero string `json:"contrato_numero"`
	Etapa          int    `json:"etapa"`
	DaysInStage    int    `json:"days_in_stage"`
	SLADays        int    `json:"sla_days"`
}

// CertidaoRow é uma certidão de compliance vencida ou a vencer.
type CertidaoRow struct {
	DemandaID      string `json:"demanda_id"`
	ContratoNumero string `json:"contrato_numero"`
	DocType        string `json:"doc_type"`
	Label          string `json:"label"`
	ValidadeAte    string `json:"validade_ate"` // YYYY-MM-DD
	Expired        bool   `json:"expired"`
}

// DashboardData é o modelo de leitura do painel de Demandas Mensais.
type DashboardData struct {
	Funnel               []FunnelStage  `json:"funnel"`
	TotalEmAndamento     int            `json:"total_em_andamento"` // etapas 1..5
	TotalArquivadas      int            `json:"total_arquivadas"`   // etapa 6
	SLABreached          int            `json:"sla_breached"`
	SLABreachedItems     []SLABreachRow `json:"sla_breached_items"`
	OpenOccurrences      int            `json:"open_occurrences"`
	CertidoesExpired     int            `json:"certidoes_expired"`
	CertidoesExpiring30d int            `json:"certidoes_expiring_30d"`
	CertidoesItems       []CertidaoRow  `json:"certidoes_items"`
}

// Dashboard monta o painel: funil por etapa, demandas fora do SLA,
// pendências abertas e certidões vencidas / a vencer em 30 dias.
func (s *Service) Dashboard(ctx context.Context, now time.Time) (DashboardData, error) {
	var out DashboardData

	counts, err := s.repo.CountByEtapa(ctx)
	if err != nil {
		return out, fmt.Errorf("demands dashboard: count by etapa: %w", err)
	}
	for e := domain.Etapa1ElaborarOF; e <= domain.Etapa6Contabilidade; e++ {
		n := counts[e]
		out.Funnel = append(out.Funnel, FunnelStage{Etapa: int(e), Label: e.Label(), Total: n})
		if e == domain.Etapa6Contabilidade {
			out.TotalArquivadas = n
		} else {
			out.TotalEmAndamento += n
		}
	}

	// SLA: mesma regra do SweepSLA — demandas paradas há mais que o menor SLA
	// (5 dias) e efetivamente fora do prazo da própria etapa.
	stale, err := s.repo.ListStale(ctx, now.AddDate(0, 0, -5))
	if err != nil {
		return out, fmt.Errorf("demands dashboard: list stale: %w", err)
	}
	for _, d := range stale {
		sla := domain.ComputeSLA(d.Etapa, d.EtapaStartedAt, now)
		if !sla.Breached {
			continue
		}
		out.SLABreached++
		if len(out.SLABreachedItems) < 15 {
			out.SLABreachedItems = append(out.SLABreachedItems, SLABreachRow{
				DemandaID:      d.ID.String(),
				ContratoNumero: d.ContratoNumero,
				Etapa:          int(d.Etapa),
				DaysInStage:    sla.DaysInStage,
				SLADays:        sla.SLADays,
			})
		}
	}

	occ, err := s.repo.ListOpenOccurrences(ctx, 500)
	if err != nil {
		return out, fmt.Errorf("demands dashboard: open occurrences: %w", err)
	}
	out.OpenOccurrences = len(occ)

	// Certidões: vencidas + a vencer nos próximos 30 dias.
	certs, err := s.repo.ListExpiringCertidoes(ctx, now.AddDate(0, 0, 30), 60)
	if err != nil {
		return out, fmt.Errorf("demands dashboard: expiring certidoes: %w", err)
	}
	for _, c := range certs {
		expired := c.ValidadeAte.Before(now)
		if expired {
			out.CertidoesExpired++
		} else {
			out.CertidoesExpiring30d++
		}
		out.CertidoesItems = append(out.CertidoesItems, CertidaoRow{
			DemandaID:      c.DemandaID.String(),
			ContratoNumero: c.ContratoNumero,
			DocType:        string(c.DocType),
			Label:          c.DocType.Label(),
			ValidadeAte:    c.ValidadeAte.Format("2006-01-02"),
			Expired:        expired,
		})
	}

	return out, nil
}
