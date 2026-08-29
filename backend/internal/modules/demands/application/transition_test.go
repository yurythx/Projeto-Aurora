package application

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

func doc(dt domain.DocumentType, validade *time.Time) domain.DemandDocument {
	return domain.DemandDocument{ID: uuid.New(), DocType: dt, FileName: string(dt) + ".pdf", Validade: validade}
}

func demandAt(etapa domain.EtapaKanban, ct domain.ContractType, docs ...domain.DemandDocument) domain.MonthlyDemand {
	return domain.MonthlyDemand{ID: uuid.New(), Etapa: etapa, ContractType: ct, Documents: docs}
}

func TestValidateTransition_NoSkippingStages(t *testing.T) {
	d := demandAt(domain.Etapa1ElaborarOF, domain.CompraConsumo)
	if err := ValidateTransition(d, domain.Etapa3EmitirOS, time.Now()); err == nil {
		t.Error("pular da etapa 1 para 3 deveria ser bloqueado")
	}
}

func TestValidateTransition_RegressionAlwaysAllowed(t *testing.T) {
	d := demandAt(domain.Etapa4ExecucaoRecepcao, domain.CompraConsumo)
	if err := ValidateTransition(d, domain.Etapa2TramitarPlan, time.Now()); err != nil {
		t.Errorf("regredir de etapa deveria ser sempre permitido: %v", err)
	}
}

func TestValidateTransition_Stage1_BlocksWithoutDocs(t *testing.T) {
	d := demandAt(domain.Etapa1ElaborarOF, domain.CompraConsumo, doc(domain.DocOFPreEmpenho, nil))
	if err := ValidateTransition(d, domain.Etapa2TramitarPlan, time.Now()); err == nil {
		t.Error("etapa 1→2 sem o Ofício ao Planejamento deveria bloquear")
	}

	d = demandAt(domain.Etapa1ElaborarOF, domain.CompraConsumo,
		doc(domain.DocOFPreEmpenho, nil), doc(domain.DocOficioPlanej, nil))
	if err := ValidateTransition(d, domain.Etapa2TramitarPlan, time.Now()); err != nil {
		t.Errorf("etapa 1→2 com OF + Ofício deveria passar: %v", err)
	}
}

func TestValidateTransition_Stage2_WaitOnly(t *testing.T) {
	d := demandAt(domain.Etapa2TramitarPlan, domain.CompraConsumo)
	if err := ValidateTransition(d, domain.Etapa3EmitirOS, time.Now()); err != nil {
		t.Errorf("etapa 2→3 é espera externa, não exige documento: %v", err)
	}
	if !CheckAdvance(d, domain.Etapa3EmitirOS, time.Now()).WaitOnly {
		t.Error("etapa 2 deveria ser marcada WaitOnly")
	}
}

func allComplianceDocs(validade *time.Time) []domain.DemandDocument {
	return []domain.DemandDocument{
		doc(domain.DocExtratoEmpenho, nil), doc(domain.DocRelatorioPgto, nil),
		doc(domain.DocCertidaoSimples, validade), doc(domain.DocCertidaoCNDT, validade),
		doc(domain.DocCertidaoFGTS, validade), doc(domain.DocCertidaoMunicipal, validade),
		doc(domain.DocCertidaoEstadual, validade), doc(domain.DocCertidaoFederal, validade),
	}
}

// TestValidateTransition_Stage5_ExpiredCertidaoBlocks é a regressão do bug:
// certidão anexada mas VENCIDA passava pelo guardrail.
func TestValidateTransition_Stage5_ExpiredCertidaoBlocks(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	valid := now.AddDate(0, 1, 0)
	expired := now.AddDate(0, 0, -5)

	ok := demandAt(domain.Etapa5RelatorioPgto, domain.CompraConsumo, allComplianceDocs(&valid)...)
	if err := ValidateTransition(ok, domain.Etapa6Contabilidade, now); err != nil {
		t.Errorf("etapa 5→6 com todas as certidões válidas deveria passar: %v", err)
	}

	bad := demandAt(domain.Etapa5RelatorioPgto, domain.CompraConsumo, allComplianceDocs(&expired)...)
	err := ValidateTransition(bad, domain.Etapa6Contabilidade, now)
	if err == nil {
		t.Fatal("etapa 5→6 com certidão VENCIDA deveria bloquear (IN SCL 01/2019)")
	}

	// E o checklist deve marcar Expired.
	check := CheckAdvance(bad, domain.Etapa6Contabilidade, now)
	if check.CanAdvance {
		t.Error("CheckAdvance.CanAdvance deveria ser false")
	}
	anyExpired := false
	for _, ds := range check.Docs {
		if ds.IsCertidao && ds.Expired {
			anyExpired = true
		}
	}
	if !anyExpired {
		t.Error("CheckAdvance deveria reportar ao menos uma certidão Expired")
	}
}

func TestValidateTransition_Stage5_ServiceContractRequiresDAMandMedicao(t *testing.T) {
	now := time.Now()
	valid := now.AddDate(0, 1, 0)

	base := allComplianceDocs(&valid)
	svc := demandAt(domain.Etapa5RelatorioPgto, domain.ServicosTerceirizados, base...)
	if err := ValidateTransition(svc, domain.Etapa6Contabilidade, now); err == nil {
		t.Error("contrato de serviço na etapa 5→6 sem DAM/Medição deveria bloquear")
	}

	withExtra := append(base, doc(domain.DocGuiaDAMISSQN, nil), doc(domain.DocPlanilhaMedicao, nil))
	svc2 := demandAt(domain.Etapa5RelatorioPgto, domain.ServicosTerceirizados, withExtra...)
	if err := ValidateTransition(svc2, domain.Etapa6Contabilidade, now); err != nil {
		t.Errorf("contrato de serviço com DAM + Medição deveria passar: %v", err)
	}

	// Contrato de compra NÃO exige os dois extras.
	buy := demandAt(domain.Etapa5RelatorioPgto, domain.CompraConsumo, base...)
	if err := ValidateTransition(buy, domain.Etapa6Contabilidade, now); err != nil {
		t.Errorf("contrato de compra não deveria exigir DAM/Medição: %v", err)
	}
}
