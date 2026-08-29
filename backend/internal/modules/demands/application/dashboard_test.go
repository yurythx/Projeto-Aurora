package application

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

type dashRepo struct {
	domain.Repository
	counts map[domain.EtapaKanban]int
	stale  []domain.MonthlyDemand
	occ    []domain.Occurrence
	certs  []domain.ExpiringCertidao
}

func (d dashRepo) CountByEtapa(context.Context) (map[domain.EtapaKanban]int, error) {
	return d.counts, nil
}
func (d dashRepo) ListStale(context.Context, time.Time) ([]domain.MonthlyDemand, error) {
	return d.stale, nil
}
func (d dashRepo) ListOpenOccurrences(context.Context, int) ([]domain.Occurrence, error) {
	return d.occ, nil
}
func (d dashRepo) ListExpiringCertidoes(context.Context, time.Time, int) ([]domain.ExpiringCertidao, error) {
	return d.certs, nil
}

func TestDashboard_FunnelSlaAndCertidoes(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	repo := dashRepo{
		counts: map[domain.EtapaKanban]int{
			domain.Etapa1ElaborarOF:    4,
			domain.Etapa3EmitirOS:      2,
			domain.Etapa6Contabilidade: 7,
		},
		stale: []domain.MonthlyDemand{
			// Etapa 2 (SLA 15) parada há 30 dias -> breached
			{ID: uuid.New(), Etapa: domain.Etapa2TramitarPlan, EtapaStartedAt: now.AddDate(0, 0, -30), ContratoNumero: "010/2026"},
			// Etapa 4 (SLA 30) parada há 10 dias -> dentro do prazo
			{ID: uuid.New(), Etapa: domain.Etapa4ExecucaoRecepcao, EtapaStartedAt: now.AddDate(0, 0, -10), ContratoNumero: "011/2026"},
		},
		occ: []domain.Occurrence{{ID: uuid.New()}, {ID: uuid.New()}},
		certs: []domain.ExpiringCertidao{
			{DemandaID: uuid.New(), ContratoNumero: "010/2026", DocType: domain.DocCertidaoFGTS, ValidadeAte: now.AddDate(0, 0, -3)}, // vencida
			{DemandaID: uuid.New(), ContratoNumero: "012/2026", DocType: domain.DocCertidaoCNDT, ValidadeAte: now.AddDate(0, 0, 12)}, // a vencer
		},
	}
	svc := &Service{repo: repo, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	d, err := svc.Dashboard(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}

	if len(d.Funnel) != 6 {
		t.Fatalf("funil deve ter 6 etapas, got %d", len(d.Funnel))
	}
	if d.Funnel[0].Total != 4 || d.Funnel[0].Label == "" {
		t.Errorf("etapa 1: %+v", d.Funnel[0])
	}
	if d.Funnel[1].Total != 0 {
		t.Errorf("etapa 2 sem demandas deve vir 0, got %d", d.Funnel[1].Total)
	}
	if d.TotalEmAndamento != 6 { // 4 + 2
		t.Errorf("em andamento (etapas 1..5) = %d, esperava 6", d.TotalEmAndamento)
	}
	if d.TotalArquivadas != 7 {
		t.Errorf("arquivadas (etapa 6) = %d, esperava 7", d.TotalArquivadas)
	}

	if d.SLABreached != 1 || len(d.SLABreachedItems) != 1 {
		t.Errorf("SLA breached = %d / itens %d, esperava 1/1", d.SLABreached, len(d.SLABreachedItems))
	}
	if d.SLABreachedItems[0].ContratoNumero != "010/2026" || d.SLABreachedItems[0].SLADays != 15 {
		t.Errorf("item SLA inesperado: %+v", d.SLABreachedItems[0])
	}

	if d.OpenOccurrences != 2 {
		t.Errorf("pendências = %d, esperava 2", d.OpenOccurrences)
	}

	if d.CertidoesExpired != 1 || d.CertidoesExpiring30d != 1 {
		t.Errorf("certidões vencidas/a vencer = %d/%d, esperava 1/1", d.CertidoesExpired, d.CertidoesExpiring30d)
	}
	if d.CertidoesItems[0].DocType != "CERTIDAO_FGTS" || !d.CertidoesItems[0].Expired || d.CertidoesItems[0].Label == "" {
		t.Errorf("item de certidão inesperado: %+v", d.CertidoesItems[0])
	}
}
