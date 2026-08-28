package domain

import (
	"context"

	"github.com/google/uuid"
)

// Repository define a interface de persistência para as demandas mensais e documentos.
type Repository interface {
	// Create inicia uma nova demanda mensal para um contrato.
	Create(ctx context.Context, demand MonthlyDemand) error
	
	// GetByID busca uma demanda com todos os seus documentos anexados.
	GetByID(ctx context.Context, id uuid.UUID) (MonthlyDemand, error)
	
	// Update atualiza os dados da demanda (etapa, status).
	Update(ctx context.Context, demand MonthlyDemand) error
	
	// ListByContract retorna todas as demandas de um contrato específico.
	ListByContract(ctx context.Context, contratoID uuid.UUID) ([]MonthlyDemand, error)
	
	// ListAllKanban retorna todas as demandas ativas em todas as etapas.
	ListAllKanban(ctx context.Context) (map[EtapaKanban][]MonthlyDemand, error)
	
	// AddDocument anexa um novo documento exigido (compliance).
	AddDocument(ctx context.Context, doc DemandDocument) error
	
	// RemoveDocument remove um documento anexado caso haja erro de upload.
	RemoveDocument(ctx context.Context, docID uuid.UUID) error
}
