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
	RetryCount    int           `json:"retry_count"`
	ErrorMessage  *string       `json:"error_message,omitempty"`
	// PDFSHA256 é o SHA-256 (hex minúsculo, 64 chars) dos bytes brutos do PDF,
	// gravado só após a ingestão completa. Vazio = ainda não ingerida sob a
	// idempotência por hash binário (ver migration 000040 e SyncWorkerPool).
	PDFSHA256 string    `json:"pdf_sha256,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Finding representa um ato administrativo (pessoal ou contrato) extraído do texto do PDF.
//
// Os campos estruturados (Secretaria, JobRole, DASLevel, PortariaNumber,
// PDFPageNumber) são preenchidos pelo parser na ingestão e persistidos junto
// com o finding — antes disso o Service os re-derivava do RawContent com regex
// a cada consulta, o que divergia do que era indexado no Typesense.
//
// EditionNumber, PdfURL e EditionDate NÃO são colunas de diario_oficial_findings:
// vêm do JOIN com diario_oficial_editions em SearchFindings. EditionDate é a
// data real de publicação no Diário Oficial e é a chave de ordenação
// cronológica correta (CreatedAt é apenas o instante da ingestão do ETL).
type Finding struct {
	ID             uuid.UUID `json:"id"`
	EditionID      int64     `json:"edition_id"`
	ExternalID     string    `json:"external_id"`
	ActType        string    `json:"act_type"` // vocabulário canônico — ver internal/gazette/act_type.go
	ServidorNome   *string   `json:"servidor_nome,omitempty"`
	CPF            *string   `json:"cpf,omitempty"`
	Matricula      *string   `json:"matricula,omitempty"`
	EmpresaNome    *string   `json:"empresa_nome,omitempty"`
	CNPJ           *string   `json:"cnpj,omitempty"`
	Valor          *float64  `json:"valor,omitempty"`
	Secretaria     *string   `json:"secretaria,omitempty"`
	JobRole        *string   `json:"job_role,omitempty"`
	DASLevel       *string   `json:"das_level,omitempty"`
	PortariaNumber *string   `json:"portaria_number,omitempty"`
	PDFPageNumber  int       `json:"pdf_page_number"`
	Confidence     string    `json:"confidence"` // high | medium | low — ver internal/gazette
	RawContent     string    `json:"raw_content"`
	CreatedAt      time.Time `json:"created_at"`

	// Preenchidos pelo JOIN com diario_oficial_editions:
	EditionNumber string    `json:"edition_number,omitempty"`
	PdfURL        string    `json:"pdf_url,omitempty"`
	EditionDate   time.Time `json:"edition_date,omitempty"`
}

// EditionRepository define a interface de persistência de edições e achados/findings no PostgreSQL.
type EditionRepository interface {
	SaveEdition(ctx context.Context, ed *Edition) error
	GetEditionByNumber(ctx context.Context, editionNumber string) (*Edition, error)
	GetPendingEditions(ctx context.Context, limit int) ([]Edition, error)
	ListAllEditions(ctx context.Context, limit int) ([]Edition, error)

	// RequeueFailedEditions reencaminha edições FAILED para PENDING enquanto
	// retry_count < maxRetries (incrementando-o), e devolve quantas foram
	// reencaminhadas. Chamado pelo watcher a cada ciclo.
	RequeueFailedEditions(ctx context.Context, maxRetries int) (int, error)
	UpdateEditionStatus(ctx context.Context, editionID int64, status EditionStatus, recordsCount int, errMsg *string) error

	// SetEditionPDFHash grava o SHA-256 dos bytes brutos do PDF já ingerido.
	// Chamado só depois de SaveFindingsTx ter sucesso — o hash é o carimbo de
	// "estes bytes exatos já foram integralmente processados" que habilita o
	// atalho de idempotência estrita numa reprocessagem.
	SetEditionPDFHash(ctx context.Context, editionID int64, sha256Hex string) error

	SaveFindingsTx(ctx context.Context, editionID int64, findings []Finding) error
	SearchFindings(ctx context.Context, query string, actType string, limit, offset int) ([]Finding, int, error)

	// ListFindingsForIndex pagina TODOS os findings (com os metadados da
	// edição via JOIN) por keyset no id, para o reindexador do Typesense.
	// afterID zero (uuid.Nil) começa do início. Ordem estável por id asc.
	ListFindingsForIndex(ctx context.Context, afterID uuid.UUID, limit int) ([]Finding, error)

	// ListFindingsByEdition retorna os findings de uma edição já com os
	// metadados dela preenchidos — usado para reindexar uma única edição
	// logo após a ingestão.
	ListFindingsByEdition(ctx context.Context, editionID int64) ([]Finding, error)

	// ListFindingsForReview retorna os findings de baixa confiança
	// (confidence='low') AINDA NÃO revisados — o resíduo do parser que ficou
	// fora da busca do usuário e precisa de triagem manual.
	ListFindingsForReview(ctx context.Context, limit int) ([]Finding, error)

	// GetFindingByID busca um finding pelo id (com os metadados da edição via
	// JOIN). apperrors.NotFound quando não existe.
	GetFindingByID(ctx context.Context, id uuid.UUID) (*Finding, error)

	// UpdateFindingReview sobrescreve os campos extraídos de um finding com a
	// versão corrigida pelo revisor, ajusta a confiança e carimba a revisão
	// (reviewed_at/by + nota). Devolve o finding já atualizado, com os
	// metadados da edição. É o caminho "promover" da fila de revisão.
	UpdateFindingReview(ctx context.Context, id uuid.UUID, in FindingReviewInput, reviewedBy *uuid.UUID) (*Finding, error)

	// AcknowledgeFinding só carimba a revisão (reviewed_at/by + nota) sem
	// mexer na confiança — o caminho "manter": o finding é low mas legítimo,
	// sai da fila sem entrar na busca.
	AcknowledgeFinding(ctx context.Context, id uuid.UUID, reviewedBy *uuid.UUID, note string) error

	// DeleteFinding remove o finding em definitivo — o caminho "descartar"
	// (ruído do parser). Volta se a edição for reprocessada.
	DeleteFinding(ctx context.Context, id uuid.UUID) error
}

// FindingReviewInput é a versão corrigida à mão dos campos extraídos de um
// finding. Ponteiro nil zera a coluna (NULL); a chamada substitui TODAS estas
// colunas, então o cliente envia o estado completo já corrigido.
type FindingReviewInput struct {
	ActType        string   `json:"act_type"`
	Confidence     string   `json:"confidence"` // high | medium
	ServidorNome   *string  `json:"servidor_nome"`
	CPF            *string  `json:"cpf"`
	Matricula      *string  `json:"matricula"`
	EmpresaNome    *string  `json:"empresa_nome"`
	CNPJ           *string  `json:"cnpj"`
	Valor          *float64 `json:"valor"`
	Secretaria     *string  `json:"secretaria"`
	JobRole        *string  `json:"job_role"`
	DASLevel       *string  `json:"das_level"`
	PortariaNumber *string  `json:"portaria_number"`
	ReviewNote     string   `json:"review_note"`
}
