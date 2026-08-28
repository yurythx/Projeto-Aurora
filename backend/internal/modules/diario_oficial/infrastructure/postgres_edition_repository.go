package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
)

type PostgresEditionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresEditionRepository(pool *pgxpool.Pool) *PostgresEditionRepository {
	return &PostgresEditionRepository{pool: pool}
}

var _ domain.EditionRepository = (*PostgresEditionRepository)(nil)

func (r *PostgresEditionRepository) SaveEdition(ctx context.Context, ed *domain.Edition) error {
	query := `
		INSERT INTO diario_oficial_editions (edition_number, edition_date, pdf_url, status, records_count, error_message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (edition_number) DO UPDATE
		SET pdf_url = EXCLUDED.pdf_url,
		    updated_at = NOW()
		RETURNING id, created_at, updated_at;
	`
	err := r.pool.QueryRow(ctx, query, ed.EditionNumber, ed.EditionDate, ed.PdfURL, ed.Status, ed.RecordsCount, ed.ErrorMessage).Scan(&ed.ID, &ed.CreatedAt, &ed.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres_edition_repository save edition: %w", err)
	}
	return nil
}

func (r *PostgresEditionRepository) GetEditionByNumber(ctx context.Context, editionNumber string) (*domain.Edition, error) {
	query := `
		SELECT id, edition_number, edition_date, pdf_url, status, records_count, error_message, created_at, updated_at
		FROM diario_oficial_editions
		WHERE edition_number = $1;
	`
	var ed domain.Edition
	err := r.pool.QueryRow(ctx, query, editionNumber).Scan(
		&ed.ID, &ed.EditionNumber, &ed.EditionDate, &ed.PdfURL, &ed.Status, &ed.RecordsCount, &ed.ErrorMessage, &ed.CreatedAt, &ed.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres_edition_repository get edition: %w", err)
	}
	return &ed, nil
}

func (r *PostgresEditionRepository) GetPendingEditions(ctx context.Context, limit int) ([]domain.Edition, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, edition_number, edition_date, pdf_url, status, records_count, error_message, created_at, updated_at
		FROM diario_oficial_editions
		WHERE status = 'PENDING'
		ORDER BY edition_date DESC
		LIMIT $1;
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres_edition_repository get pending editions: %w", err)
	}
	defer rows.Close()

	editions := make([]domain.Edition, 0)
	for rows.Next() {
		var ed domain.Edition
		if err := rows.Scan(&ed.ID, &ed.EditionNumber, &ed.EditionDate, &ed.PdfURL, &ed.Status, &ed.RecordsCount, &ed.ErrorMessage, &ed.CreatedAt, &ed.UpdatedAt); err != nil {
			return nil, err
		}
		editions = append(editions, ed)
	}
	return editions, nil
}

func (r *PostgresEditionRepository) ListAllEditions(ctx context.Context, limit int) ([]domain.Edition, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, edition_number, edition_date, pdf_url, status, records_count, error_message, created_at, updated_at
		FROM diario_oficial_editions
		ORDER BY edition_date DESC
		LIMIT $1;
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres_edition_repository list all editions: %w", err)
	}
	defer rows.Close()

	editions := make([]domain.Edition, 0)
	for rows.Next() {
		var ed domain.Edition
		if err := rows.Scan(&ed.ID, &ed.EditionNumber, &ed.EditionDate, &ed.PdfURL, &ed.Status, &ed.RecordsCount, &ed.ErrorMessage, &ed.CreatedAt, &ed.UpdatedAt); err != nil {
			return nil, err
		}
		editions = append(editions, ed)
	}
	return editions, nil
}

func (r *PostgresEditionRepository) UpdateEditionStatus(ctx context.Context, editionID int64, status domain.EditionStatus, recordsCount int, errMsg *string) error {
	query := `
		UPDATE diario_oficial_editions
		SET status = $1,
		    records_count = $2,
		    error_message = $3,
		    updated_at = NOW()
		WHERE id = $4;
	`
	_, err := r.pool.Exec(ctx, query, status, recordsCount, errMsg, editionID)
	if err != nil {
		return fmt.Errorf("postgres_edition_repository update edition status: %w", err)
	}
	return nil
}

// SaveFindingsTx salva os achados/findings dentro de uma transação SQL atômica e idempotente.
func (r *PostgresEditionRepository) SaveFindingsTx(ctx context.Context, editionID int64, findings []domain.Finding) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("postgres_edition_repository begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	findingInsertQuery := `
		INSERT INTO diario_oficial_findings (id, edition_id, external_id, act_type, servidor_nome, cpf, matricula, empresa_nome, cnpj, valor, raw_content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (external_id) DO NOTHING;
	`

	insertedCount := 0
	for _, f := range findings {
		cmdTag, fErr := tx.Exec(ctx, findingInsertQuery,
			f.ID, editionID, f.ExternalID, f.ActType, f.ServidorNome, f.CPF, f.Matricula, f.EmpresaNome, f.CNPJ, f.Valor, f.RawContent, time.Now(),
		)
		if fErr != nil {
			return fmt.Errorf("postgres_edition_repository insert finding: %w", fErr)
		}
		if cmdTag.RowsAffected() > 0 {
			insertedCount++
		}
	}

	// Atualiza status da edição para COMPLETED na mesma transação atômica
	updateEditionQuery := `
		UPDATE diario_oficial_editions
		SET status = 'COMPLETED',
		    records_count = records_count + $1,
		    updated_at = NOW()
		WHERE id = $2;
	`
	if _, err := tx.Exec(ctx, updateEditionQuery, insertedCount, editionID); err != nil {
		return fmt.Errorf("postgres_edition_repository update edition status in tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres_edition_repository commit tx: %w", err)
	}

	return nil
}

// SearchFindings executa a consulta otimizada por Trigramas e Full-Text Search no PostgreSQL.
func (r *PostgresEditionRepository) SearchFindings(ctx context.Context, queryStr string, actType string, limit, offset int) ([]domain.Finding, int, error) {
	if limit <= 0 {
		limit = 20
	}

	var sb strings.Builder
	args := make([]interface{}, 0)
	argIdx := 1

	sb.WriteString(`
		SELECT id, edition_id, external_id, act_type, servidor_nome, cpf, matricula, empresa_nome, cnpj, valor, raw_content, created_at
		FROM diario_oficial_findings
		WHERE 1=1
	`)

	if actType != "" {
		sb.WriteString(fmt.Sprintf(" AND act_type = $%d", argIdx))
		args = append(args, actType)
		argIdx++
	}

	if strings.TrimSpace(queryStr) != "" {
		sClean := strings.TrimSpace(queryStr)
		likePattern := "%" + sClean + "%"
		digitsOnly := regexp.MustCompile(`\D`).ReplaceAllString(sClean, "")

		sb.WriteString(fmt.Sprintf(` AND (
			servidor_nome ILIKE $%d OR
			empresa_nome ILIKE $%d OR
			cpf ILIKE $%d OR
			matricula ILIKE $%d OR
			cnpj ILIKE $%d OR
			raw_content ILIKE $%d`, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx))
		args = append(args, likePattern)
		argIdx++

		if digitsOnly != "" && len(digitsOnly) >= 3 {
			sb.WriteString(fmt.Sprintf(` OR REPLACE(REPLACE(REPLACE(cpf, '.', ''), '-', ''), '/', '') LIKE $%d`, argIdx))
			sb.WriteString(fmt.Sprintf(` OR REPLACE(REPLACE(REPLACE(cnpj, '.', ''), '-', ''), '/', '') LIKE $%d`, argIdx))
			args = append(args, "%"+digitsOnly+"%")
			argIdx++
		}

		sb.WriteString(")")
	}

	sb.WriteString(" ORDER BY created_at DESC")
	sb.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1))
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres_edition_repository search findings: %w", err)
	}
	defer rows.Close()

	findings := make([]domain.Finding, 0)
	for rows.Next() {
		var f domain.Finding
		if err := rows.Scan(
			&f.ID, &f.EditionID, &f.ExternalID, &f.ActType, &f.ServidorNome, &f.CPF, &f.Matricula, &f.EmpresaNome, &f.CNPJ, &f.Valor, &f.RawContent, &f.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		findings = append(findings, f)
	}

	return findings, len(findings), nil
}
