package application

import (
	"fmt"
	"strings"
	"time"

	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

// DocStatus descreve a situação de UM documento exigido para avançar de etapa.
type DocStatus struct {
	DocType    string     `json:"doc_type"`
	Label      string     `json:"label"`
	Present    bool       `json:"present"`
	IsCertidao bool       `json:"is_certidao"`
	Expired    bool       `json:"expired"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
}

// StageCheck é o resultado de avaliar a transição de uma demanda para a etapa
// seguinte: o que falta, o que está vencido, e se pode avançar. É o mesmo
// cálculo que o guardrail usa — exposto via GET /demands/{id}/requirements
// para o Kanban desenhar o checklist ANTES de o fiscal tentar mover.
type StageCheck struct {
	FromEtapa  int         `json:"from_etapa"`
	ToEtapa    int         `json:"to_etapa"`
	CanAdvance bool        `json:"can_advance"`
	Blocking   []string    `json:"blocking,omitempty"`
	Docs       []DocStatus `json:"docs"`
	// WaitOnly = etapa de espera externa (2 e 6): não exige documento, só tempo.
	WaitOnly bool `json:"wait_only"`
}

// CheckAdvance avalia a passagem de demand.Etapa para target (que deve ser
// demand.Etapa+1 para fazer sentido; regressão sempre pode).
func CheckAdvance(demand domain.MonthlyDemand, target domain.EtapaKanban, now time.Time) StageCheck {
	check := StageCheck{
		FromEtapa: int(demand.Etapa),
		ToEtapa:   int(target),
		Docs:      []DocStatus{},
	}

	// Regressão ou permanência: sempre permitido, nada a exigir.
	if target <= demand.Etapa {
		check.CanAdvance = true
		return check
	}
	if target > demand.Etapa+1 {
		check.Blocking = append(check.Blocking,
			fmt.Sprintf("não é permitido pular etapas (de %d para %d)", demand.Etapa, target))
		return check
	}

	present := make(map[domain.DocumentType]domain.DemandDocument, len(demand.Documents))
	for _, d := range demand.Documents {
		present[d.DocType] = d
	}

	required := domain.RequiredDocsForAdvance(demand.Etapa, demand.ContractType)
	if len(required) == 0 {
		check.WaitOnly = true
		check.CanAdvance = true
		return check
	}

	for _, dt := range required {
		doc, has := present[dt]
		st := DocStatus{
			DocType:    string(dt),
			Label:      dt.Label(),
			Present:    has,
			IsCertidao: dt.IsCertidao(),
		}
		if has && doc.Validade != nil {
			st.ValidUntil = doc.Validade
			st.Expired = doc.Validade.Before(now)
		}
		check.Docs = append(check.Docs, st)

		switch {
		case !has:
			check.Blocking = append(check.Blocking, "falta: "+st.Label)
		case st.Expired:
			check.Blocking = append(check.Blocking,
				fmt.Sprintf("%s vencida em %s", st.Label, doc.Validade.Format("02/01/2006")))
		}
	}

	check.CanAdvance = len(check.Blocking) == 0
	return check
}

// ValidateTransition é a Máquina de Estados do Kanban: verifica se a demanda
// pode ir da etapa atual para nextStage com base nos documentos anexados e na
// validade das certidões (IN SCL 01/2019 — "card locked if any required
// certificate is expired/missing").
func ValidateTransition(demand domain.MonthlyDemand, nextStage domain.EtapaKanban, now time.Time) error {
	check := CheckAdvance(demand, nextStage, now)
	if check.CanAdvance {
		return nil
	}
	return fmt.Errorf("transição bloqueada (etapa %d → %d): %s",
		check.FromEtapa, check.ToEtapa, strings.Join(check.Blocking, "; "))
}
