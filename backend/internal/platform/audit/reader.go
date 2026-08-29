package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// LogRow é uma linha de audit_logs devolvida para leitura (ex.: o painel
// de histórico de uma demanda). O pacote continua não interpretando o
// conteúdo de Metadata — só o repassa como veio.
type LogRow struct {
	ID            uuid.UUID      `json:"id"`
	UserID        *uuid.UUID     `json:"user_id,omitempty"`
	Action        string         `json:"action"`
	ResourceType  string         `json:"resource_type"`
	ResourceID    string         `json:"resource_id"`
	Metadata      map[string]any `json:"metadata"`
	CorrelationID *uuid.UUID     `json:"correlation_id,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

// queryer é satisfeita por *pgxpool.Pool e por pgx.Tx.
type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Reader lê a trilha imutável de audit_logs. Só leitura — escrita é do
// Writer, e a migration 000008 bloqueia UPDATE/DELETE/TRUNCATE na tabela.
type Reader struct {
	db queryer
}

// NewReader constrói um Reader sobre db.
func NewReader(db queryer) *Reader {
	return &Reader{db: db}
}

// ListByResource devolve as entradas de um recurso específico
// (resource_type + resource_id), da mais recente para a mais antiga.
// limit <= 0 usa 100; teto de 500.
func (r *Reader) ListByResource(ctx context.Context, resourceType, resourceID string, limit int) ([]LogRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	const q = `
		SELECT id, user_id, action, COALESCE(resource_type, ''), COALESCE(resource_id, ''),
		       metadata, correlation_id, created_at
		FROM audit_logs
		WHERE resource_type = $1 AND resource_id = $2
		ORDER BY created_at DESC
		LIMIT $3`
	rows, err := r.db.Query(ctx, q, resourceType, resourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("audit: list by resource: %w", err)
	}
	defer rows.Close()

	out := make([]LogRow, 0)
	for rows.Next() {
		var row LogRow
		var meta []byte
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.Action, &row.ResourceType, &row.ResourceID,
			&meta, &row.CorrelationID, &row.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("audit: scan log row: %w", err)
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &row.Metadata)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
