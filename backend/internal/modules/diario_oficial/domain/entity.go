package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type EditionStatus string

const (
	EditionStatusPending    EditionStatus = "PENDING"
	EditionStatusProcessing EditionStatus = "PROCESSING"
	EditionStatusCompleted  EditionStatus = "COMPLETED"
	EditionStatusFailed     EditionStatus = "FAILED"
)

// Edition representa a entidade de uma edição individual do Diário Oficial de Rondonópolis.
type Edition struct {
	ID            int64         `json:"id"`
	EditionNumber string        `json:"edition_number"`
	EditionDate   time.Time     `json:"edition_date"`
	PdfURL        string        `json:"pdf_url"`
	Status        EditionStatus `json:"status"`
	RecordsCount  int           `json:"records_count"`
	ErrorMessage  *string       `json:"error_message,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// Finding representa um ato administrativo (pessoal ou contrato) extraído do texto do PDF.
type Finding struct {
	ID           uuid.UUID `json:"id"`
	EditionID    int64     `json:"edition_id"`
	ExternalID   string    `json:"external_id"`
	ActType      string    `json:"act_type"` // NOMEACAO, EXONERACAO, MUDANCA_SETOR, CONTRATO, LICITACAO
	ServidorNome *string   `json:"servidor_nome,omitempty"`
	CPF          *string   `json:"cpf,omitempty"`
	Matricula    *string   `json:"matricula,omitempty"`
	EmpresaNome  *string   `json:"empresa_nome,omitempty"`
	CNPJ         *string   `json:"cnpj,omitempty"`
	Valor        *float64  `json:"valor,omitempty"`
	RawContent   string    `json:"raw_content"`
	CreatedAt    time.Time `json:"created_at"`
}

// EditionRepository define a interface de persistência de edições e achados/findings no PostgreSQL.
type EditionRepository interface {
	SaveEdition(ctx context.Context, ed *Edition) error
	GetEditionByNumber(ctx context.Context, editionNumber string) (*Edition, error)
	GetPendingEditions(ctx context.Context, limit int) ([]Edition, error)
	ListAllEditions(ctx context.Context, limit int) ([]Edition, error)
	UpdateEditionStatus(ctx context.Context, editionID int64, status EditionStatus, recordsCount int, errMsg *string) error
	SaveFindingsTx(ctx context.Context, editionID int64, findings []Finding) error
	SearchFindings(ctx context.Context, query string, actType string, limit, offset int) ([]Finding, int, error)
}
