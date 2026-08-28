package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

// PostgresRepository implementa a persistência de demandas no PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository constrói um novo repositório apontando para o pool do pgx.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Create insere uma nova demanda mensal na tabela.
func (r *PostgresRepository) Create(ctx context.Context, demand domain.MonthlyDemand) error {
	query := `
		INSERT INTO monthly_demands (
			id, contrato_id, ano_mes, etapa, status_etapa, observacoes, etapa_started_at, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		demand.ID,
		demand.ContratoID,
		demand.AnoMes,
		demand.Etapa,
		demand.StatusEtapa,
		demand.Observacoes,
		demand.EtapaStartedAt,
		demand.CreatedBy,
		demand.CreatedAt,
		demand.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres_repository: create monthly_demand: %w", err)
	}
	return nil
}

// GetByID busca a demanda, descobrindo também o tipo de contrato e documentos anexos.
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.MonthlyDemand, error) {
	queryDemand := `
		SELECT 
			m.id, m.contrato_id, m.ano_mes, m.etapa, m.status_etapa, m.observacoes, m.etapa_started_at, m.created_by, m.created_at, m.updated_at,
			c.tipo_contrato, c.numero, c.objeto, c.contratado, c.valor
		FROM monthly_demands m
		JOIN contratos c ON c.id = m.contrato_id
		WHERE m.id = $1
	`
	
	var d domain.MonthlyDemand
	err := r.pool.QueryRow(ctx, queryDemand, id).Scan(
		&d.ID, &d.ContratoID, &d.AnoMes, &d.Etapa, &d.StatusEtapa, &d.Observacoes, &d.EtapaStartedAt, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt,
		&d.ContractType, &d.ContratoNumero, &d.ContratoObjeto, &d.Contratado, &d.ContratoValor,
	)
	if err != nil {
		return domain.MonthlyDemand{}, fmt.Errorf("postgres_repository: get demand by id: %w", err)
	}

	// Carregar documentos anexados
	queryDocs := `
		SELECT id, demanda_id, doc_type, file_path, file_name, uploaded_by, uploaded_at, validade_ate
		FROM demand_documents
		WHERE demanda_id = $1
	`
	rows, err := r.pool.Query(ctx, queryDocs, id)
	if err != nil {
		return domain.MonthlyDemand{}, fmt.Errorf("postgres_repository: get demand docs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var doc domain.DemandDocument
		if err := rows.Scan(&doc.ID, &doc.DemandaID, &doc.DocType, &doc.FilePath, &doc.FileName, &doc.UploadedBy, &doc.UploadedAt, &doc.Validade); err != nil {
			return domain.MonthlyDemand{}, fmt.Errorf("postgres_repository: scan demand doc: %w", err)
		}
		d.Documents = append(d.Documents, doc)
	}
	
	return d, nil
}

// Update salva mudanças na etapa e observações.
func (r *PostgresRepository) Update(ctx context.Context, demand domain.MonthlyDemand) error {
	query := `
		UPDATE monthly_demands
		SET etapa = $1, status_etapa = $2, observacoes = $3, etapa_started_at = $4, updated_at = $5
		WHERE id = $6
	`
	_, err := r.pool.Exec(ctx, query,
		demand.Etapa,
		demand.StatusEtapa,
		demand.Observacoes,
		demand.EtapaStartedAt,
		time.Now(),
		demand.ID,
	)
	if err != nil {
		return fmt.Errorf("postgres_repository: update demand: %w", err)
	}
	return nil
}

// UpdateTx salva mudanças na etapa e observações, garantindo atomicidade na transação fornecida.
func (r *PostgresRepository) UpdateTx(ctx context.Context, tx pgx.Tx, demand domain.MonthlyDemand) error {
	query := `
		UPDATE monthly_demands
		SET etapa = $1, status_etapa = $2, observacoes = $3, etapa_started_at = $4, updated_at = $5
		WHERE id = $6
	`
	_, err := tx.Exec(ctx, query,
		demand.Etapa,
		demand.StatusEtapa,
		demand.Observacoes,
		demand.EtapaStartedAt,
		time.Now(),
		demand.ID,
	)
	if err != nil {
		return fmt.Errorf("postgres_repository: update demand tx: %w", err)
	}
	return nil
}

// ListByContract busca todas as demandas (faturas) geradas para um contrato.
func (r *PostgresRepository) ListByContract(ctx context.Context, contratoID uuid.UUID) ([]domain.MonthlyDemand, error) {
	query := `
		SELECT 
			m.id, m.contrato_id, m.ano_mes, m.etapa, m.status_etapa, m.observacoes, m.etapa_started_at, m.created_at, m.updated_at,
			c.numero, c.objeto, c.contratado, c.valor
		FROM monthly_demands m
		JOIN contratos c ON c.id = m.contrato_id
		WHERE m.contrato_id = $1
		ORDER BY m.ano_mes DESC
	`
	rows, err := r.pool.Query(ctx, query, contratoID)
	if err != nil {
		return nil, fmt.Errorf("postgres_repository: list by contract: %w", err)
	}
	defer rows.Close()

	var list []domain.MonthlyDemand
	for rows.Next() {
		var d domain.MonthlyDemand
		if err := rows.Scan(&d.ID, &d.ContratoID, &d.AnoMes, &d.Etapa, &d.StatusEtapa, &d.Observacoes, &d.EtapaStartedAt, &d.CreatedAt, &d.UpdatedAt, &d.ContratoNumero, &d.ContratoObjeto, &d.Contratado, &d.ContratoValor); err != nil {
			return nil, fmt.Errorf("postgres_repository: scan demand in list: %w", err)
		}
		list = append(list, d)
	}
	return list, nil
}

// ListAllKanban retorna as demandas para alimentar a interface do quadro.
func (r *PostgresRepository) ListAllKanban(ctx context.Context) (map[domain.EtapaKanban][]domain.MonthlyDemand, error) {
	query := `
		SELECT 
			m.id, m.contrato_id, m.ano_mes, m.etapa, m.status_etapa, m.observacoes, m.etapa_started_at, m.created_at, m.updated_at,
			c.numero, c.objeto, c.contratado, c.valor
		FROM monthly_demands m
		JOIN contratos c ON c.id = m.contrato_id
		ORDER BY m.etapa ASC, m.updated_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres_repository: list all kanban: %w", err)
	}
	defer rows.Close()

	result := make(map[domain.EtapaKanban][]domain.MonthlyDemand)
	for i := 1; i <= 6; i++ {
		result[domain.EtapaKanban(i)] = []domain.MonthlyDemand{}
	}

	for rows.Next() {
		var d domain.MonthlyDemand
		if err := rows.Scan(&d.ID, &d.ContratoID, &d.AnoMes, &d.Etapa, &d.StatusEtapa, &d.Observacoes, &d.EtapaStartedAt, &d.CreatedAt, &d.UpdatedAt, &d.ContratoNumero, &d.ContratoObjeto, &d.Contratado, &d.ContratoValor); err != nil {
			return nil, fmt.Errorf("postgres_repository: scan kanban demand: %w", err)
		}
		result[d.Etapa] = append(result[d.Etapa], d)
	}
	return result, nil
}

// AddDocument vincula um novo documento à demanda.
func (r *PostgresRepository) AddDocument(ctx context.Context, doc domain.DemandDocument) error {
	query := `
		INSERT INTO demand_documents (
			id, demanda_id, doc_type, file_path, file_name, uploaded_by, uploaded_at, validade_ate
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		doc.ID, doc.DemandaID, doc.DocType, doc.FilePath, doc.FileName, doc.UploadedBy, doc.UploadedAt, doc.Validade,
	)
	if err != nil {
		return fmt.Errorf("postgres_repository: add document: %w", err)
	}
	return nil
}

// RemoveDocument remove o documento da base de dados.
func (r *PostgresRepository) RemoveDocument(ctx context.Context, docID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM demand_documents WHERE id = $1`, docID)
	if err != nil {
		return fmt.Errorf("postgres_repository: remove document: %w", err)
	}
	return nil
}
