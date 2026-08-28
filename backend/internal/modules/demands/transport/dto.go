package transport

import (
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

// CreateDemandRequest é o payload para iniciar o fluxo de uma demanda mensal.
type CreateDemandRequest struct {
	ContratoID  uuid.UUID `json:"contrato_id"`
	AnoMes      string    `json:"ano_mes"` // ex: "2026-08"
	Observacoes string    `json:"observacoes,omitempty"`
}

// UpdateEtapaRequest é o payload para mover um card no Kanban.
type UpdateEtapaRequest struct {
	TargetEtapa int `json:"target_etapa"`
}

// AddDocumentRequest é o payload para anexar um documento de compliance.
type AddDocumentRequest struct {
	DocType  string  `json:"doc_type"`
	FilePath string  `json:"file_path"`
	FileName string  `json:"file_name"`
	Validade *string `json:"validade_ate,omitempty"` // Formato YYYY-MM-DD
}

// DemandResponse é o espelho de saída de uma demanda.
type DemandResponse struct {
	ID             uuid.UUID          `json:"id"`
	ContratoID     uuid.UUID          `json:"contrato_id"`
	AnoMes         string             `json:"ano_mes"`
	Etapa          int                `json:"etapa"`
	StatusEtapa    string             `json:"status_etapa"`
	Observacoes    string             `json:"observacoes"`
	EtapaStartedAt time.Time          `json:"etapa_started_at"`
	Documents      []DocumentResponse `json:"documents,omitempty"`
	// Campos enriquecidos via JOIN com contratos
	ContratoNumero string  `json:"contrato_numero,omitempty"`
	ContratoObjeto string  `json:"contrato_objeto,omitempty"`
	Contratado     string  `json:"contratado,omitempty"`
	ContratoValor  *float64 `json:"contrato_valor,omitempty"`
}

// DocumentResponse reflete um arquivo anexo.
type DocumentResponse struct {
	ID         uuid.UUID  `json:"id"`
	DocType    string     `json:"doc_type"`
	FilePath   string     `json:"file_path"`
	FileName   string     `json:"file_name"`
	UploadedAt time.Time  `json:"uploaded_at"`
	Validade   *time.Time `json:"validade_ate,omitempty"`
}

// toDemandResponse converte do domínio puro para a saída HTTP.
func toDemandResponse(d domain.MonthlyDemand) DemandResponse {
	docs := make([]DocumentResponse, len(d.Documents))
	for i, doc := range d.Documents {
		docs[i] = DocumentResponse{
			ID:         doc.ID,
			DocType:    string(doc.DocType),
			FilePath:   doc.FilePath,
			FileName:   doc.FileName,
			UploadedAt: doc.UploadedAt,
			Validade:   doc.Validade,
		}
	}

	return DemandResponse{
		ID:             d.ID,
		ContratoID:     d.ContratoID,
		AnoMes:         d.AnoMes,
		Etapa:          int(d.Etapa),
		StatusEtapa:    string(d.StatusEtapa),
		Observacoes:    d.Observacoes,
		EtapaStartedAt: d.EtapaStartedAt,
		Documents:      docs,
		ContratoNumero: d.ContratoNumero,
		ContratoObjeto: d.ContratoObjeto,
		Contratado:     d.Contratado,
		ContratoValor:  d.ContratoValor,
	}
}
