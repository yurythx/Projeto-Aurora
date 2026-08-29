package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova/internal/domain/pagination"
)

// DashboardStats contém agregadores rápidos para a tela inicial.
type DashboardStats struct {
	TotalVigentes      int     `json:"total_vigentes"`
	ValorTotalVigentes float64 `json:"valor_total_vigentes"`
	ProximosVencimento int     `json:"proximos_vencimento"`
}

// Repository abstrai a persistência do módulo de contratos.
// Implementado em infrastructure/postgres_repository.go.
type Repository interface {
	// Create grava um novo contrato. O ID é gerado pelo chamador antes
	// desta chamada (uuid.New()), mesmo padrão dos demais módulos.
	Create(ctx context.Context, c Contrato) error

	// GetByID busca um contrato por ID. Retorna apperrors.NotFound se
	// não existir. DiarioRefs e Aditivos não são preenchidos por esta
	// chamada — use GetWithDetails para incluí-los.
	GetByID(ctx context.Context, id uuid.UUID) (Contrato, error)

	// GetWithDetails busca um contrato por ID incluindo DiarioRefs e
	// Aditivos em consultas separadas.
	GetWithDetails(ctx context.Context, id uuid.UUID) (Contrato, error)

	// List retorna uma página de contratos conforme os filtros em params.
	List(ctx context.Context, params ListParams) ([]Contrato, int64, error)

	// UpdateStatus atualiza o status de um contrato. Retorna
	// apperrors.NotFound se não existir.
	UpdateStatus(ctx context.Context, id uuid.UUID, newStatus Status) error

	// Update atualiza os campos editáveis de um contrato (objeto,
	// contratante, contratado, cnpj, valor, datas de vigência).
	// Não atualiza status — use UpdateStatus para isso.
	Update(ctx context.Context, c Contrato) error

	// AddAditivo adiciona um aditivo ao contrato dentro de tx. Usado
	// no fluxo de worker (outbox) para atomicidade com o evento.
	AddAditivo(ctx context.Context, tx pgx.Tx, a Aditivo) error

	// AddDiarioRef vincula uma publicação do Diário Oficial ao contrato.
	// ON CONFLICT (contrato_id, edition_number) DO NOTHING — seguro de
	// chamar múltiplas vezes para a mesma referência.
	AddDiarioRef(ctx context.Context, ref DiarioRef) error

	// LinkDiarioRef é o AddDiarioRef do casador automático: mesmo INSERT
	// idempotente, mas devolve se a linha é NOVA (true) — o worker só
	// notifica quando de fato vinculou algo.
	LinkDiarioRef(ctx context.Context, ref DiarioRef) (inserted bool, err error)

	// RecordAlertOnce registra que um alerta (kind + ref_key) já foi
	// emitido para o contrato. Devolve true só na primeira vez — o worker
	// usa isso para não repetir a mesma notificação a cada ciclo.
	RecordAlertOnce(ctx context.Context, contratoID uuid.UUID, kind, refKey string) (firstTime bool, err error)

	// ListByStatus agrupa os contratos por status — usado pela visão
	// Kanban para retornar as colunas em uma única consulta.
	ListByStatus(ctx context.Context) (map[Status][]Contrato, error)

	// KanbanCols retorna os contratos agrupados por status com paginação
	// por coluna (limit por coluna). Usado pela rota /contratos/kanban.
	KanbanCols(ctx context.Context, limitPerCol int) (map[Status][]Contrato, error)

	// ListDiarioRefs lista as referências de Diário Oficial de um contrato.
	ListDiarioRefs(ctx context.Context, contratoID uuid.UUID) ([]DiarioRef, error)

	// ListAditivos lista os aditivos de um contrato.
	ListAditivos(ctx context.Context, contratoID uuid.UUID) ([]Aditivo, error)

	// FindByNumero busca um contrato pelo número (único no sistema).
	// Retorna apperrors.NotFound se não existir.
	FindByNumero(ctx context.Context, numero string) (Contrato, error)

	// Pagination helper: wraps List with pagination.Params.
	ListPage(ctx context.Context, params ListParams, page pagination.Params) ([]Contrato, int64, error)

	// GetDashboardStats retorna estatísticas macro (ex.: valor total de vigentes).
	GetDashboardStats(ctx context.Context) (DashboardStats, error)
}
