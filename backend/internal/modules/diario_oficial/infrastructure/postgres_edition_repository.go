package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/gazette"
	"github.com/yurythx/projeto-nova/internal/modules/diario_oficial/domain"
)

// findingSelectBase é o SELECT compartilhado por toda leitura de findings: os
// campos de diario_oficial_findings mais os metadados da edição via JOIN
// (edition_number, pdf_url e a edition_date REAL — a chave de ordenação
// cronológica, ao contrário de created_at que é só o instante da ingestão).
const findingSelectBase = `
	SELECT f.id, f.edition_id, f.external_id, f.act_type,
	       f.servidor_nome, f.cpf, f.matricula, f.empresa_nome, f.cnpj, f.valor,
	       f.secretaria, f.job_role, f.das_level, f.portaria_number, f.pdf_page_number, f.confidence,
	       f.raw_content, f.created_at,
	       COALESCE(e.edition_number, ''), COALESCE(e.pdf_url, ''), e.edition_date
	FROM diario_oficial_findings f
	LEFT JOIN diario_oficial_editions e ON e.id = f.edition_id
`

// scanFinding lê uma linha no layout de findingSelectBase.
func scanFinding(row pgx.Row, f *domain.Finding) error {
	var edDate *time.Time
	if err := row.Scan(
		&f.ID, &f.EditionID, &f.ExternalID, &f.ActType,
		&f.ServidorNome, &f.CPF, &f.Matricula, &f.EmpresaNome, &f.CNPJ, &f.Valor,
		&f.Secretaria, &f.JobRole, &f.DASLevel, &f.PortariaNumber, &f.PDFPageNumber, &f.Confidence,
		&f.RawContent, &f.CreatedAt,
		&f.EditionNumber, &f.PdfURL, &edDate,
	); err != nil {
		return err
	}
	if edDate != nil {
		f.EditionDate = *edDate
	}
	return nil
}

type PostgresEditionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresEditionRepository(pool *pgxpool.Pool) *PostgresEditionRepository {
	return &PostgresEditionRepository{pool: pool}
}

var _ domain.EditionRepository = (*PostgresEditionRepository)(nil)

func (r *PostgresEditionRepository) SaveEdition(ctx context.Context, ed *domain.Edition) error {
	// Data desconhecida entra como NULL (não 0001-01-01 nem NOW()); a
	// ordenação cronológica usa NULLS LAST e o watcher tenta preencher depois.
	var editionDate *time.Time
	if !ed.EditionDate.IsZero() {
		d := ed.EditionDate
		editionDate = &d
	}
	query := `
		INSERT INTO diario_oficial_editions (edition_number, edition_date, pdf_url, status, records_count, error_message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (edition_number) DO UPDATE
		SET pdf_url = EXCLUDED.pdf_url,
		    edition_date = COALESCE(diario_oficial_editions.edition_date, EXCLUDED.edition_date),
		    updated_at = NOW()
		RETURNING id, created_at, updated_at;
	`
	err := r.pool.QueryRow(ctx, query, ed.EditionNumber, editionDate, ed.PdfURL, ed.Status, ed.RecordsCount, ed.ErrorMessage).Scan(&ed.ID, &ed.CreatedAt, &ed.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres_edition_repository save edition: %w", err)
	}
	return nil
}

const editionSelectCols = `id, edition_number, edition_date, pdf_url, status, records_count, retry_count, error_message, created_at, updated_at`

func scanEdition(row pgx.Row, ed *domain.Edition) error {
	// edition_date virou nullable (000031): NULL não entra num time.Time direto.
	var edDate *time.Time
	if err := row.Scan(
		&ed.ID, &ed.EditionNumber, &edDate, &ed.PdfURL, &ed.Status,
		&ed.RecordsCount, &ed.RetryCount, &ed.ErrorMessage, &ed.CreatedAt, &ed.UpdatedAt,
	); err != nil {
		return err
	}
	if edDate != nil {
		ed.EditionDate = *edDate
	} else {
		ed.EditionDate = time.Time{}
	}
	return nil
}

func (r *PostgresEditionRepository) queryEditions(ctx context.Context, tail string, args ...interface{}) ([]domain.Edition, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+editionSelectCols+" FROM diario_oficial_editions "+tail, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres_edition_repository query editions: %w", err)
	}
	defer rows.Close()

	editions := make([]domain.Edition, 0)
	for rows.Next() {
		var ed domain.Edition
		if err := scanEdition(rows, &ed); err != nil {
			return nil, err
		}
		editions = append(editions, ed)
	}
	return editions, rows.Err()
}

func (r *PostgresEditionRepository) GetEditionByNumber(ctx context.Context, editionNumber string) (*domain.Edition, error) {
	var ed domain.Edition
	err := scanEdition(
		r.pool.QueryRow(ctx, "SELECT "+editionSelectCols+" FROM diario_oficial_editions WHERE edition_number = $1", editionNumber),
		&ed,
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
	return r.queryEditions(ctx, "WHERE status = 'PENDING' ORDER BY edition_date DESC NULLS LAST LIMIT $1", limit)
}

func (r *PostgresEditionRepository) ListAllEditions(ctx context.Context, limit int) ([]domain.Edition, error) {
	if limit <= 0 {
		limit = 50
	}
	return r.queryEditions(ctx, "ORDER BY edition_date DESC NULLS LAST LIMIT $1", limit)
}

// RequeueFailedEditions reencaminha edições FAILED de volta para PENDING
// enquanto retry_count < maxRetries, incrementando o contador. Uma edição
// permanentemente quebrada para de ser tentada ao atingir o limite.
func (r *PostgresEditionRepository) RequeueFailedEditions(ctx context.Context, maxRetries int) (int, error) {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE diario_oficial_editions
		   SET status = 'PENDING', retry_count = retry_count + 1, updated_at = NOW()
		 WHERE status = 'FAILED' AND retry_count < $1
	`, maxRetries)
	if err != nil {
		return 0, fmt.Errorf("postgres_edition_repository requeue failed editions: %w", err)
	}
	return int(tag.RowsAffected()), nil
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

	// Findings são 100% derivados do PDF: reprocessar uma edição substitui o
	// conjunto inteiro. Sem este DELETE, uma mudança no parser gera um
	// external_id novo (o hash inclui trecho do texto) e as linhas antigas
	// ficam acumuladas — era por isso que reprocessar exigia TRUNCATE manual.
	if _, err := tx.Exec(ctx, `DELETE FROM diario_oficial_findings WHERE edition_id = $1`, editionID); err != nil {
		return fmt.Errorf("postgres_edition_repository clear edition findings: %w", err)
	}

	findingInsertQuery := `
		INSERT INTO diario_oficial_findings
			(id, edition_id, external_id, act_type, servidor_nome, cpf, matricula, empresa_nome, cnpj, valor,
			 secretaria, job_role, das_level, portaria_number, pdf_page_number, confidence, raw_content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (external_id) DO NOTHING;
	`

	now := time.Now()
	insertedCount := 0
	for _, f := range findings {
		page := f.PDFPageNumber
		if page < 1 {
			page = 1
		}
		confidence := f.Confidence
		if confidence == "" {
			confidence = gazette.ConfidenceMedium
		}
		cmdTag, fErr := tx.Exec(ctx, findingInsertQuery,
			f.ID, editionID, f.ExternalID, f.ActType,
			sanitizeUTF8Ptr(f.ServidorNome), sanitizeUTF8Ptr(f.CPF), sanitizeUTF8Ptr(f.Matricula),
			sanitizeUTF8Ptr(f.EmpresaNome), sanitizeUTF8Ptr(f.CNPJ), f.Valor,
			sanitizeUTF8Ptr(f.Secretaria), sanitizeUTF8Ptr(f.JobRole), sanitizeUTF8Ptr(f.DASLevel), sanitizeUTF8Ptr(f.PortariaNumber),
			page, confidence, sanitizeUTF8(f.RawContent), now,
		)
		if fErr != nil {
			return fmt.Errorf("postgres_edition_repository insert finding: %w", fErr)
		}
		if cmdTag.RowsAffected() > 0 {
			insertedCount++
		}
	}

	// records_count é ABSOLUTO (o DELETE acima zerou o conjunto), não incremental.
	updateEditionQuery := `
		UPDATE diario_oficial_editions
		SET status = 'COMPLETED',
		    records_count = $1,
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
//
// Busca com relevância: full-text search em português sobre raw_content
// (usa o índice GIN de to_tsvector da migration 000028, que antes ficava
// ocioso), combinada com ILIKE parcial em nome/empresa e busca por dígitos em
// CPF/CNPJ. Ordena por ts_rank e depois pela data real da edição
// (NULLS LAST). total_count é um COUNT(*) real da janela.
func (r *PostgresEditionRepository) SearchFindings(ctx context.Context, queryStr string, actType string, limit, offset int) ([]domain.Finding, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	var sb strings.Builder
	args := make([]interface{}, 0)
	argIdx := 1

	rankExpr := "0::float4"
	sClean := strings.TrimSpace(queryStr)
	if sClean != "" {
		// $1 é sempre o termo de busca quando há um.
		rankExpr = "ts_rank(to_tsvector('portuguese', f.raw_content), websearch_to_tsquery('portuguese', $1))"
	}

	sel := strings.Replace(findingSelectBase, "\tFROM diario_oficial_findings f",
		"\t     , COUNT(*) OVER() AS total_count\n\t     , "+rankExpr+" AS rank\n\tFROM diario_oficial_findings f", 1)
	sb.WriteString(sel)
	sb.WriteString(" WHERE 1=1\n")

	if sClean != "" {
		args = append(args, sClean) // $1
		argIdx++
		likePattern := "%" + sClean + "%"
		digitsOnly := regexp.MustCompile(`\D`).ReplaceAllString(sClean, "")

		sb.WriteString(fmt.Sprintf(` AND (
			to_tsvector('portuguese', f.raw_content) @@ websearch_to_tsquery('portuguese', $1) OR
			f.servidor_nome ILIKE $%d OR
			f.empresa_nome ILIKE $%d`, argIdx, argIdx))
		args = append(args, likePattern)
		argIdx++

		if digitsOnly != "" && len(digitsOnly) >= 3 {
			sb.WriteString(fmt.Sprintf(` OR REPLACE(REPLACE(REPLACE(f.cpf, '.', ''), '-', ''), '/', '') LIKE $%d`, argIdx))
			sb.WriteString(fmt.Sprintf(` OR REPLACE(REPLACE(REPLACE(f.cnpj, '.', ''), '-', ''), '/', '') LIKE $%d`, argIdx))
			sb.WriteString(fmt.Sprintf(` OR f.matricula LIKE $%d`, argIdx))
			args = append(args, "%"+digitsOnly+"%")
			argIdx++
		}
		sb.WriteString(")")
	}

	if actType != "" {
		sb.WriteString(fmt.Sprintf(" AND f.act_type = $%d", argIdx))
		args = append(args, gazette.NormalizeActType(actType))
		argIdx++
	}

	sb.WriteString(" ORDER BY rank DESC, e.edition_date DESC NULLS LAST, f.created_at DESC")
	sb.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1))
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres_edition_repository search findings: %w", err)
	}
	defer rows.Close()

	findings := make([]domain.Finding, 0)
	total := 0
	for rows.Next() {
		var f domain.Finding
		var edDate *time.Time
		var rank float32
		if err := rows.Scan(
			&f.ID, &f.EditionID, &f.ExternalID, &f.ActType,
			&f.ServidorNome, &f.CPF, &f.Matricula, &f.EmpresaNome, &f.CNPJ, &f.Valor,
			&f.Secretaria, &f.JobRole, &f.DASLevel, &f.PortariaNumber, &f.PDFPageNumber, &f.Confidence,
			&f.RawContent, &f.CreatedAt,
			&f.EditionNumber, &f.PdfURL, &edDate,
			&total, &rank,
		); err != nil {
			return nil, 0, err
		}
		if edDate != nil {
			f.EditionDate = *edDate
		}
		findings = append(findings, f)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return findings, total, nil
}

// ListFindingsForIndex pagina todos os findings por keyset (id asc) para o
// reindexador do Typesense.
func (r *PostgresEditionRepository) ListFindingsForIndex(ctx context.Context, afterID uuid.UUID, limit int) ([]domain.Finding, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	q := findingSelectBase + " WHERE f.id > $1 ORDER BY f.id ASC LIMIT $2"
	rows, err := r.pool.Query(ctx, q, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres_edition_repository list findings for index: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Finding, 0, limit)
	for rows.Next() {
		var f domain.Finding
		if err := scanFinding(rows, &f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ListFindingsForReview retorna os findings de baixa confiança, mais recentes
// primeiro — a fila de revisão manual.
func (r *PostgresEditionRepository) ListFindingsForReview(ctx context.Context, limit int) ([]domain.Finding, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := findingSelectBase + " WHERE f.confidence = 'low' AND f.reviewed_at IS NULL ORDER BY f.created_at DESC LIMIT $1"
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres_edition_repository list findings for review: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Finding, 0)
	for rows.Next() {
		var f domain.Finding
		if err := scanFinding(rows, &f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// GetFindingByID busca um finding pelo id, com os metadados da edição.
func (r *PostgresEditionRepository) GetFindingByID(ctx context.Context, id uuid.UUID) (*domain.Finding, error) {
	q := findingSelectBase + " WHERE f.id = $1"
	var f domain.Finding
	if err := scanFinding(r.pool.QueryRow(ctx, q, id), &f); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("finding não encontrado")
		}
		return nil, fmt.Errorf("postgres_edition_repository get finding by id: %w", err)
	}
	return &f, nil
}

// UpdateFindingReview aplica a correção manual do revisor: substitui os campos
// extraídos, ajusta a confiança e carimba a revisão. Devolve a visão já com o
// JOIN da edição.
func (r *PostgresEditionRepository) UpdateFindingReview(ctx context.Context, id uuid.UUID, in domain.FindingReviewInput, reviewedBy *uuid.UUID) (*domain.Finding, error) {
	const q = `
		UPDATE diario_oficial_findings SET
			act_type        = $2,
			confidence      = $3,
			servidor_nome   = $4,
			cpf             = $5,
			matricula       = $6,
			empresa_nome    = $7,
			cnpj            = $8,
			valor           = $9,
			secretaria      = $10,
			job_role        = $11,
			das_level       = $12,
			portaria_number = $13,
			review_note     = $14,
			reviewed_by     = $15,
			reviewed_at     = NOW()
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id,
		in.ActType, in.Confidence,
		sanitizeUTF8Ptr(in.ServidorNome), sanitizeUTF8Ptr(in.CPF), sanitizeUTF8Ptr(in.Matricula),
		sanitizeUTF8Ptr(in.EmpresaNome), sanitizeUTF8Ptr(in.CNPJ), in.Valor,
		sanitizeUTF8Ptr(in.Secretaria), sanitizeUTF8Ptr(in.JobRole), sanitizeUTF8Ptr(in.DASLevel),
		sanitizeUTF8Ptr(in.PortariaNumber), sanitizeUTF8(in.ReviewNote), reviewedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres_edition_repository update finding review: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperrors.NotFound("finding não encontrado")
	}
	return r.GetFindingByID(ctx, id)
}

// AcknowledgeFinding só carimba a revisão, sem mexer na confiança.
func (r *PostgresEditionRepository) AcknowledgeFinding(ctx context.Context, id uuid.UUID, reviewedBy *uuid.UUID, note string) error {
	const q = `UPDATE diario_oficial_findings
		SET reviewed_at = NOW(), reviewed_by = $2, review_note = $3
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id, reviewedBy, sanitizeUTF8(note))
	if err != nil {
		return fmt.Errorf("postgres_edition_repository acknowledge finding: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NotFound("finding não encontrado")
	}
	return nil
}

// DeleteFinding remove o finding em definitivo (descartar ruído do parser).
func (r *PostgresEditionRepository) DeleteFinding(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM diario_oficial_findings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres_edition_repository delete finding: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NotFound("finding não encontrado")
	}
	return nil
}

// ListFindingsByEdition retorna os findings de uma edição com os metadados dela
// já preenchidos — usado para reindexar uma edição logo após a ingestão.
func (r *PostgresEditionRepository) ListFindingsByEdition(ctx context.Context, editionID int64) ([]domain.Finding, error) {
	q := findingSelectBase + " WHERE f.edition_id = $1 ORDER BY f.pdf_page_number ASC, f.created_at ASC"
	rows, err := r.pool.Query(ctx, q, editionID)
	if err != nil {
		return nil, fmt.Errorf("postgres_edition_repository list findings by edition: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Finding, 0)
	for rows.Next() {
		var f domain.Finding
		if err := scanFinding(rows, &f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// MatchFindingsForContract devolve os findings (confiança >= média) que
// citam o número do contrato — em portaria_number ou no corpo do texto — ou
// que casam pelo CNPJ (só dígitos). numero e cnpjDigits vazios são ignorados;
// se ambos vierem vazios devolve nada. Usado pelo casador automático
// Contrato <-> Diário (worker contratos.diario_matcher).
func (r *PostgresEditionRepository) MatchFindingsForContract(ctx context.Context, numero, cnpjDigits string, limit int) ([]domain.Finding, error) {
	if numero == "" && cnpjDigits == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := findingSelectBase + `
		WHERE f.confidence <> 'low'
		  AND (
		        ($1 <> '' AND (
		            COALESCE(f.portaria_number, '') ILIKE '%' || $1 || '%'
		            OR f.raw_content ILIKE '%' || $1 || '%'
		        ))
		     OR ($2 <> '' AND regexp_replace(COALESCE(f.cnpj, ''), '\D', '', 'g') = $2)
		  )
		ORDER BY e.edition_date DESC NULLS LAST, f.created_at DESC
		LIMIT $3`
	rows, err := r.pool.Query(ctx, q, numero, cnpjDigits, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres_edition_repository match findings for contract: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Finding, 0)
	for rows.Next() {
		var f domain.Finding
		if err := scanFinding(rows, &f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// sanitizeUTF8 garante UTF-8 válido antes do INSERT (o PostgreSQL rejeita
// bytes inválidos com SQLSTATE 22021 — foi a causa de 2 edições ficarem
// FAILED). Defesa em profundidade: o texto já passou por
// gazette.RepairEncoding na extração; aqui é a última barreira antes do banco.
func sanitizeUTF8(s string) string {
	return gazette.RepairEncoding(s)
}

func sanitizeUTF8Ptr(s *string) *string {
	if s == nil {
		return nil
	}
	clean := sanitizeUTF8(*s)
	return &clean
}
