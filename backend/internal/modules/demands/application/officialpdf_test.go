package application

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

func demandForPDF(etapa domain.EtapaKanban) domain.MonthlyDemand {
	v := 12345.67
	return domain.MonthlyDemand{
		ID:             uuid.New(),
		AnoMes:         "2026-08",
		Etapa:          etapa,
		ContratoNumero: "012/2026",
		ContratoObjeto: "Fornecimento de material de limpeza e higienização — pregão 45/2026",
		Contratado:     "AÇÃO LIMPEZA & CONSERVAÇÃO LTDA",
		ContratoValor:  &v,
		Observacoes:    "Medição referente à 2ª parcela.",
		Documents: []domain.DemandDocument{
			{ID: uuid.New(), DocType: domain.DocCertidaoFGTS, FileName: "fgts.pdf", UploadedAt: time.Now(), Validade: ptrTime(time.Now().AddDate(0, 0, -1))},
			{ID: uuid.New(), DocType: domain.DocRelatorioPgto, FileName: "relatorio.pdf", UploadedAt: time.Now()},
		},
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestRenderDemandPDF_ProducesValidPDFPerKind(t *testing.T) {
	cases := []struct {
		kind  PDFKind
		etapa domain.EtapaKanban
	}{
		{PDFOficio, domain.Etapa1ElaborarOF},
		{PDFOrdemServico, domain.Etapa3EmitirOS},
		{PDFRelatorio, domain.Etapa5RelatorioPgto},
	}
	for _, c := range cases {
		var buf bytes.Buffer
		if err := RenderDemandPDF(c.kind, demandForPDF(c.etapa), &buf); err != nil {
			t.Fatalf("%s: RenderDemandPDF: %v", c.kind, err)
		}
		b := buf.Bytes()
		if !bytes.HasPrefix(b, []byte("%PDF-")) {
			t.Errorf("%s: saída não começa com %%PDF- (got %q)", c.kind, b[:min(8, len(b))])
		}
		if !bytes.Contains(b, []byte("%%EOF")) {
			t.Errorf("%s: PDF sem marcador %%%%EOF", c.kind)
		}
		if len(b) < 1000 {
			t.Errorf("%s: PDF suspeitosamente pequeno (%d bytes)", c.kind, len(b))
		}
	}
}

func TestRenderDemandPDF_BlocksBelowMinEtapa(t *testing.T) {
	var buf bytes.Buffer
	err := RenderDemandPDF(PDFRelatorio, demandForPDF(domain.Etapa2TramitarPlan), &buf)
	if err == nil {
		t.Fatal("relatório na etapa 2 deveria ser bloqueado")
	}
	var appErr *apperrors.Error
	if !errors.As(err, &appErr) || appErr.Code != apperrors.CodeBadRequest {
		t.Errorf("esperava BadRequest, got %T: %v", err, err)
	}
	if buf.Len() != 0 {
		t.Errorf("nada deveria ter sido escrito no writer em caso de bloqueio")
	}
}

func TestPDFFileName(t *testing.T) {
	d := demandForPDF(domain.Etapa5RelatorioPgto)
	got := PDFFileName(PDFRelatorio, d)
	if !strings.HasPrefix(got, "relatorio-fiscalizacao-012-2026-2026-08") || !strings.HasSuffix(got, ".pdf") {
		t.Errorf("PDFFileName = %q", got)
	}
}
