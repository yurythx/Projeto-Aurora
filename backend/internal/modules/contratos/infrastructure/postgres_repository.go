package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/domain/pagination"
	"github.com/yurythx/projeto-nova/internal/modules/contratos/domain"
)

// PostgresRepository implementa domain.Repository usando PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository constrói o repositório.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Create insere um novo contrato.
func (r *PostgresRepository) Create(ctx context.Context, c domain.Contrato) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO contratos
			(id, numero, objeto, contratante, contratado, cnpj, valor,
			 data_assinatura, data_vigencia_inicio, data_vigencia_fim,
			 status, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		c.ID, c.Numero, c.Objeto, c.Contratante, c.Contratado, c.CNPJ, c.Valor,
		c.DataAssinatura, c.DataVigenciaInicio, c.DataVigenciaFim,
		string(c.Status), c.CreatedBy, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_contratos_numero") {
			return apperrors.Conflict(fmt.Sprintf("contrato com número '%s' já existe", c.Numero))
		}
		return fmt.Errorf("contratos: create: %w", err)
	}
	return nil
}

// GetByID busca um contrato por ID sem detalhes.
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Contrato, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, numero, objeto, contratante, contratado, cnpj, valor,
		       data_assinatura, data_vigencia_inicio, data_vigencia_fim,
		       status, created_by, created_at, updated_at
		FROM contratos WHERE id = $1`, id)
	return scanContrato(row)
}

// GetWithDetails busca um contrato por ID e carrega refs e aditivos.
func (r *PostgresRepository) GetWithDetails(ctx context.Context, id uuid.UUID) (domain.Contrato, error) {
	c, err := r.GetByID(ctx, id)
	if err != nil {
		return domain.Contrato{}, err
	}
	c.DiarioRefs, err = r.ListDiarioRefs(ctx, id)
	if err != nil {
		return domain.Contrato{}, err
	}
	c.Aditivos, err = r.ListAditivos(ctx, id)
	if err != nil {
		return domain.Contrato{}, err
	}
	return c, nil
}

// FindByNumero busca um contrato pelo número único.
func (r *PostgresRepository) FindByNumero(ctx context.Context, numero string) (domain.Contrato, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, numero, objeto, contratante, contratado, cnpj, valor,
		       data_assinatura, data_vigencia_inicio, data_vigencia_fim,
		       status, created_by, created_at, updated_at
		FROM contratos WHERE numero = $1`, numero)
	return scanContrato(row)
}

// List retorna uma página de contratos com filtros.
func (r *PostgresRepository) List(ctx context.Context, params domain.ListParams) ([]domain.Contrato, int64, error) {
	where := []string{"1=1"}
	args := []any{}
	argN := 1

	if params.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argN))
		args = append(args, string(params.Status))
		argN++
	}
	if params.Busca != "" {
		where = append(where, fmt.Sprintf(
			"(objeto ILIKE $%d OR contratado ILIKE $%d)",
			argN, argN,
		))
		args = append(args, "%"+params.Busca+"%")
		argN++
	}

	whereClause := strings.Join(where, " AND ")

	var total int64
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM contratos WHERE %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("contratos: list count: %w", err)
	}

	offset := (params.Page - 1) * params.PageSize
	args = append(args, params.PageSize, offset)

	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, numero, objeto, contratante, contratado, cnpj, valor,
		       data_assinatura, data_vigencia_inicio, data_vigencia_fim,
		       status, created_by, created_at, updated_at
		FROM contratos
		WHERE %s
		ORDER BY updated_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argN, argN+1),
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("contratos: list query: %w", err)
	}
	defer rows.Close()

	var result []domain.Contrato
	for rows.Next() {
		c, err := scanContratoRow(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, c)
	}
	return result, total, rows.Err()
}

// ListPage é um alias de List que usa pagination.Params.
func (r *PostgresRepository) ListPage(ctx context.Context, params domain.ListParams, page pagination.Params) ([]domain.Contrato, int64, error) {
	params.Page = page.Page
	params.PageSize = page.PageSize
	return r.List(ctx, params)
}

// UpdateStatus atualiza o status de um contrato.
func (r *PostgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus domain.Status) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE contratos SET status = $1, updated_at = $2 WHERE id = $3",
		string(newStatus), time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("contratos: update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NotFound("contrato não encontrado")
	}
	return nil
}

// Update atualiza os campos editáveis de um contrato.
func (r *PostgresRepository) Update(ctx context.Context, c domain.Contrato) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE contratos
		SET objeto=$1, contratante=$2, contratado=$3, cnpj=$4, valor=$5,
		    data_assinatura=$6, data_vigencia_inicio=$7, data_vigencia_fim=$8,
		    updated_at=$9
		WHERE id=$10`,
		c.Objeto, c.Contratante, c.Contratado, c.CNPJ, c.Valor,
		c.DataAssinatura, c.DataVigenciaInicio, c.DataVigenciaFim,
		time.Now(), c.ID,
	)
	if err != nil {
		return fmt.Errorf("contratos: update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NotFound("contrato não encontrado")
	}
	return nil
}

// AddAditivo insere um aditivo dentro de uma transação.
func (r *PostgresRepository) AddAditivo(ctx context.Context, tx pgx.Tx, a domain.Aditivo) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO contrato_aditivos
			(id, contrato_id, numero, objeto, valor_adicional, data_assinatura, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		a.ID, a.ContratoID, a.Numero, a.Objeto, a.ValorAdicional, a.DataAssinatura, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("contratos: add aditivo: %w", err)
	}
	return nil
}

// AddDiarioRef vincula uma publicação ao contrato (ON CONFLICT DO NOTHING).
func (r *PostgresRepository) AddDiarioRef(ctx context.Context, ref domain.DiarioRef) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO contrato_diario_refs
			(contrato_id, edition_number, tipo_evento, publicado_em, contexto, doc_url)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (contrato_id, edition_number) DO NOTHING`,
		ref.ContratoID, ref.EditionNumber, ref.TipoEvento,
		ref.PublicadoEm, ref.Contexto, ref.DocURL,
	)
	if err != nil {
		return fmt.Errorf("contratos: add diario ref: %w", err)
	}
	return nil
}

// pgxExec é satisfeito por *pgxpool.Pool e por pgx.Tx — deixa o INSERT do
// casador ser escrito uma vez e reutilizado na variante transacional.
type pgxExec interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func linkDiarioRef(ctx context.Context, db pgxExec, ref domain.DiarioRef) (bool, error) {
	tag, err := db.Exec(ctx, `
		INSERT INTO contrato_diario_refs
			(contrato_id, edition_number, tipo_evento, publicado_em, contexto, doc_url)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (contrato_id, edition_number) DO NOTHING`,
		ref.ContratoID, ref.EditionNumber, ref.TipoEvento,
		ref.PublicadoEm, ref.Contexto, ref.DocURL,
	)
	if err != nil {
		return false, fmt.Errorf("contratos: link diario ref: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func recordAlertOnce(ctx context.Context, db pgxExec, contratoID uuid.UUID, kind, refKey string) (bool, error) {
	tag, err := db.Exec(ctx, `
		INSERT INTO contrato_diario_alertas (contrato_id, kind, ref_key)
		VALUES ($1,$2,$3)
		ON CONFLICT (contrato_id, kind, ref_key) DO NOTHING`,
		contratoID, kind, refKey,
	)
	if err != nil {
		return false, fmt.Errorf("contratos: record alert once: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func listDiarioRefs(ctx context.Context, db pgxExec, contratoID uuid.UUID) ([]domain.DiarioRef, error) {
	rows, err := db.Query(ctx, `
		SELECT contrato_id, edition_number, tipo_evento, publicado_em, contexto, doc_url
		FROM contrato_diario_refs
		WHERE contrato_id = $1
		ORDER BY publicado_em DESC NULLS LAST`, contratoID)
	if err != nil {
		return nil, fmt.Errorf("contratos: list diario refs: %w", err)
	}
	defer rows.Close()
	var out []domain.DiarioRef
	for rows.Next() {
		var ref domain.DiarioRef
		if err := rows.Scan(&ref.ContratoID, &ref.EditionNumber, &ref.TipoEvento, &ref.PublicadoEm, &ref.Contexto, &ref.DocURL); err != nil {
			return nil, fmt.Errorf("contratos: scan diario ref: %w", err)
		}
		out = append(out, ref)
	}
	return out, rows.Err()
}

// LinkDiarioRefTx / RecordAlertOnceTx / ListDiarioRefsTx: variantes que
// rodam na tx de negócio do casador (padrão Transactional Outbox).
func (r *PostgresRepository) LinkDiarioRefTx(ctx context.Context, tx pgx.Tx, ref domain.DiarioRef) (bool, error) {
	return linkDiarioRef(ctx, tx, ref)
}

func (r *PostgresRepository) RecordAlertOnceTx(ctx context.Context, tx pgx.Tx, contratoID uuid.UUID, kind, refKey string) (bool, error) {
	return recordAlertOnce(ctx, tx, contratoID, kind, refKey)
}

func (r *PostgresRepository) ListDiarioRefsTx(ctx context.Context, tx pgx.Tx, contratoID uuid.UUID) ([]domain.DiarioRef, error) {
	return listDiarioRefs(ctx, tx, contratoID)
}

// ListByStatus retorna todos os contratos agrupados por status.
func (r *PostgresRepository) ListByStatus(ctx context.Context) (map[domain.Status][]domain.Contrato, error) {
	return r.KanbanCols(ctx, 100)
}

// KanbanCols retorna até limitPerCol contratos por coluna de status.
func (r *PostgresRepository) KanbanCols(ctx context.Context, limitPerCol int) (map[domain.Status][]domain.Contrato, error) {
	result := make(map[domain.Status][]domain.Contrato)
	for _, s := range domain.AllStatuses() {
		rows, err := r.pool.Query(ctx, `
			SELECT id, numero, objeto, contratante, contratado, cnpj, valor,
			       data_assinatura, data_vigencia_inicio, data_vigencia_fim,
			       status, created_by, created_at, updated_at
			FROM contratos
			WHERE status = $1
			ORDER BY updated_at DESC
			LIMIT $2`, string(s), limitPerCol)
		if err != nil {
			return nil, fmt.Errorf("contratos: kanban col %s: %w", s, err)
		}
		var col []domain.Contrato
		for rows.Next() {
			c, err := scanContratoRow(rows)
			if err != nil {
				rows.Close()
				return nil, err
			}
			col = append(col, c)
		}
		rows.Close()
		if rows.Err() != nil {
			return nil, rows.Err()
		}
		result[s] = col
	}
	return result, nil
}

// ListDiarioRefs retorna as referências do Diário Oficial de um contrato.
func (r *PostgresRepository) ListDiarioRefs(ctx context.Context, contratoID uuid.UUID) ([]domain.DiarioRef, error) {
	return listDiarioRefs(ctx, r.pool, contratoID)
}

// ListAditivos retorna os aditivos de um contrato.
func (r *PostgresRepository) ListAditivos(ctx context.Context, contratoID uuid.UUID) ([]domain.Aditivo, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, contrato_id, numero, objeto, valor_adicional, data_assinatura, created_at
		FROM contrato_aditivos
		WHERE contrato_id = $1
		ORDER BY created_at ASC`, contratoID)
	if err != nil {
		return nil, fmt.Errorf("contratos: list aditivos: %w", err)
	}
	defer rows.Close()

	var result []domain.Aditivo
	for rows.Next() {
		var a domain.Aditivo
		if err := rows.Scan(
			&a.ID, &a.ContratoID, &a.Numero, &a.Objeto,
			&a.ValorAdicional, &a.DataAssinatura, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("contratos: scan aditivo: %w", err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

// --- helpers de scan ---

type scannable interface {
	Scan(dest ...any) error
}

func scanContrato(row scannable) (domain.Contrato, error) {
	var c domain.Contrato
	var statusStr string
	if err := row.Scan(
		&c.ID, &c.Numero, &c.Objeto, &c.Contratante, &c.Contratado,
		&c.CNPJ, &c.Valor,
		&c.DataAssinatura, &c.DataVigenciaInicio, &c.DataVigenciaFim,
		&statusStr, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		if err.Error() == "no rows in result set" {
			return domain.Contrato{}, apperrors.NotFound("contrato não encontrado")
		}
		return domain.Contrato{}, fmt.Errorf("contratos: scan: %w", err)
	}
	c.Status = domain.Status(statusStr)
	return c, nil
}

func scanContratoRow(rows pgx.Rows) (domain.Contrato, error) {
	var c domain.Contrato
	var statusStr string
	if err := rows.Scan(
		&c.ID, &c.Numero, &c.Objeto, &c.Contratante, &c.Contratado,
		&c.CNPJ, &c.Valor,
		&c.DataAssinatura, &c.DataVigenciaInicio, &c.DataVigenciaFim,
		&statusStr, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return domain.Contrato{}, fmt.Errorf("contratos: scan row: %w", err)
	}
	c.Status = domain.Status(statusStr)
	return c, nil
}

// GetDashboardStats retorna estatísticas agregadas (vigentes e vencimentos próximos).
func (r *PostgresRepository) GetDashboardStats(ctx context.Context) (domain.DashboardStats, error) {
	var stats domain.DashboardStats

	// 1. Total e Valor de Vigentes
	queryVigentes := `
		SELECT COUNT(*), COALESCE(SUM(valor), 0)
		FROM contratos
		WHERE status = 'vigente'
	`
	if err := r.pool.QueryRow(ctx, queryVigentes).Scan(&stats.TotalVigentes, &stats.ValorTotalVigentes); err != nil {
		return domain.DashboardStats{}, fmt.Errorf("contratos: stats vigentes: %w", err)
	}

	// 2. Próximos ao vencimento (< 30 dias)
	queryVencimento := `
		SELECT COUNT(*)
		FROM contratos
		WHERE status = 'vigente'
		  AND data_vigencia_fim IS NOT NULL
		  AND data_vigencia_fim <= NOW() + INTERVAL '30 days'
	`
	if err := r.pool.QueryRow(ctx, queryVencimento).Scan(&stats.ProximosVencimento); err != nil {
		return domain.DashboardStats{}, fmt.Errorf("contratos: stats vencimento: %w", err)
	}

	return stats, nil
}
