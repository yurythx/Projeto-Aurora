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
			c.numero, c.objeto, c.contratado, c.valor, COALESCE(c.tipo_contrato::text, '')
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

	byID := map[uuid.UUID]*domain.MonthlyDemand{}
	ids := []uuid.UUID{}
	for rows.Next() {
		var d domain.MonthlyDemand
		var ct string
		if err := rows.Scan(&d.ID, &d.ContratoID, &d.AnoMes, &d.Etapa, &d.StatusEtapa, &d.Observacoes, &d.EtapaStartedAt, &d.CreatedAt, &d.UpdatedAt, &d.ContratoNumero, &d.ContratoObjeto, &d.Contratado, &d.ContratoValor, &ct); err != nil {
			return nil, fmt.Errorf("postgres_repository: scan kanban demand: %w", err)
		}
		d.ContractType = domain.ContractType(ct)
		result[d.Etapa] = append(result[d.Etapa], d)
		ids = append(ids, d.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for etapa := range result {
		for i := range result[etapa] {
			byID[result[etapa][i].ID] = &result[etapa][i]
		}
	}

	// Anexa os documentos de todas as demandas do quadro numa única query —
	// o Kanban precisa disso para desenhar o checklist/cadeado de cada card.
	if len(ids) > 0 {
		drows, err := r.pool.Query(ctx, `
			SELECT demanda_id, id, doc_type, file_path, file_name, uploaded_at, validade_ate
			FROM demand_documents WHERE demanda_id = ANY($1)`, ids)
		if err != nil {
			return nil, fmt.Errorf("postgres_repository: kanban docs: %w", err)
		}
		defer drows.Close()
		for drows.Next() {
			var demID uuid.UUID
			var doc domain.DemandDocument
			if err := drows.Scan(&demID, &doc.ID, &doc.DocType, &doc.FilePath, &doc.FileName, &doc.UploadedAt, &doc.Validade); err != nil {
				return nil, fmt.Errorf("postgres_repository: scan kanban doc: %w", err)
			}
			doc.DemandaID = demID
			if d := byID[demID]; d != nil {
				d.Documents = append(d.Documents, doc)
			}
		}
		if err := drows.Err(); err != nil {
			return nil, err
		}
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

// ExistsForContractMonth reporta se já há demanda do contrato no mês.
func (r *PostgresRepository) ExistsForContractMonth(ctx context.Context, contratoID uuid.UUID, anoMes string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM monthly_demands WHERE contrato_id = $1 AND ano_mes = $2)`,
		contratoID, anoMes,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("postgres_repository: exists for contract month: %w", err)
	}
	return exists, nil
}

// ListStale traz as demandas cuja etapa começou antes de olderThan.
func (r *PostgresRepository) ListStale(ctx context.Context, olderThan time.Time) ([]domain.MonthlyDemand, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, m.contrato_id, m.ano_mes, m.etapa, m.status_etapa, m.etapa_started_at,
		       COALESCE(c.numero, '')
		FROM monthly_demands m
		LEFT JOIN contratos c ON c.id = m.contrato_id
		WHERE m.etapa < 6 AND m.etapa_started_at < $1
		ORDER BY m.etapa_started_at ASC`, olderThan)
	if err != nil {
		return nil, fmt.Errorf("postgres_repository: list stale: %w", err)
	}
	defer rows.Close()

	out := make([]domain.MonthlyDemand, 0)
	for rows.Next() {
		var d domain.MonthlyDemand
		if err := rows.Scan(&d.ID, &d.ContratoID, &d.AnoMes, &d.Etapa, &d.StatusEtapa, &d.EtapaStartedAt, &d.ContratoNumero); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// CountByEtapa devolve o número de demandas em cada etapa.
func (r *PostgresRepository) CountByEtapa(ctx context.Context) (map[domain.EtapaKanban]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT etapa, count(*) FROM monthly_demands GROUP BY etapa`)
	if err != nil {
		return nil, fmt.Errorf("postgres_repository: count by etapa: %w", err)
	}
	defer rows.Close()

	out := make(map[domain.EtapaKanban]int)
	for rows.Next() {
		var etapa int
		var n int
		if err := rows.Scan(&etapa, &n); err != nil {
			return nil, err
		}
		out[domain.EtapaKanban(etapa)] = n
	}
	return out, rows.Err()
}

// ListExpiringCertidoes: a certidão mais recente de cada tipo, por demanda em
// andamento, com validade anterior a `before`.
func (r *PostgresRepository) ListExpiringCertidoes(ctx context.Context, before time.Time, limit int) ([]domain.ExpiringCertidao, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT x.demanda_id, x.numero, x.doc_type, x.validade_ate, x.etapa
		FROM (
			SELECT DISTINCT ON (d.demanda_id, d.doc_type)
			       d.demanda_id, COALESCE(c.numero, '') AS numero, d.doc_type::text AS doc_type,
			       d.validade_ate, m.etapa
			FROM demand_documents d
			JOIN monthly_demands m ON m.id = d.demanda_id
			LEFT JOIN contratos c ON c.id = m.contrato_id
			WHERE d.doc_type::text LIKE 'CERTIDAO\_%'
			  AND d.validade_ate IS NOT NULL
			  AND m.etapa < 6
			ORDER BY d.demanda_id, d.doc_type, d.uploaded_at DESC
		) x
		WHERE x.validade_ate <= $1
		ORDER BY x.validade_ate ASC
		LIMIT $2`, before, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres_repository: list expiring certidoes: %w", err)
	}
	defer rows.Close()

	out := make([]domain.ExpiringCertidao, 0)
	for rows.Next() {
		var row domain.ExpiringCertidao
		var docType string
		if err := rows.Scan(&row.DemandaID, &row.ContratoNumero, &docType, &row.ValidadeAte, &row.Etapa); err != nil {
			return nil, err
		}
		row.DocType = domain.DocumentType(docType)
		out = append(out, row)
	}
	return out, rows.Err()
}

// EnsureSLAOccurrence grava a pendência só se não houver uma aberta do mesmo
// tipo para a demanda. Retorna true se inseriu. Race-safe via o índice
// parcial único idx_contract_occurrences_open (migration 000038).
func (r *PostgresRepository) EnsureSLAOccurrence(ctx context.Context, o domain.Occurrence) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO contract_occurrences (id, contrato_id, demanda_id, tipo, descricao, sla_vence_em, resolvida, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, false, now())
		ON CONFLICT (demanda_id, tipo) WHERE resolvida = false DO NOTHING`,
		o.ID, o.ContratoID, o.DemandaID, o.Tipo, o.Descricao, o.SLAVenceEm,
	)
	if err != nil {
		return false, fmt.Errorf("postgres_repository: ensure sla occurrence: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// ListOpenOccurrences lista as pendências não resolvidas, mais recentes primeiro.
func (r *PostgresRepository) ListOpenOccurrences(ctx context.Context, limit int) ([]domain.Occurrence, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, contrato_id, demanda_id, tipo, descricao, sla_vence_em, resolvida, created_by, created_at
		FROM contract_occurrences
		WHERE resolvida = false
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres_repository: list open occurrences: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Occurrence, 0)
	for rows.Next() {
		var o domain.Occurrence
		var sla *time.Time
		if err := rows.Scan(&o.ID, &o.ContratoID, &o.DemandaID, &o.Tipo, &o.Descricao, &sla, &o.Resolvida, &o.CreatedBy, &o.CreatedAt); err != nil {
			return nil, err
		}
		o.SLAVenceEm = sla
		out = append(out, o)
	}
	return out, rows.Err()
}
