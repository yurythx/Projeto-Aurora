package audit

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/yurythx/projeto-aurora/internal/domain/errors"
	"github.com/yurythx/projeto-aurora/internal/platform/auth"
	"github.com/yurythx/projeto-aurora/pkg/httputil"
)

type Exporter struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewExporter(db *pgxpool.Pool, logger *slog.Logger) *Exporter {
	return &Exporter{db: db, logger: logger}
}

func (e *Exporter) RegisterRoutes(r interface {
	Get(path string, fn http.HandlerFunc)
}) {
	r.Get("/api/v1/audit/export", e.handleExportCSV)
}

func (e *Exporter) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	_, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, r, e.logger, apperrors.Unauthorized("Não autenticado"))
		return
	}

	const query = `
		SELECT 
			id, 
			COALESCE(user_id::text, 'SISTEMA') AS user_id, 
			action, 
			COALESCE(resource_type, '') AS resource_type, 
			COALESCE(resource_id, '') AS resource_id, 
			COALESCE(ip_address::text, '') AS ip_address, 
			created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT 1000
	`

	rows, err := e.db.Query(r.Context(), query)
	if err != nil {
		httputil.WriteError(w, r, e.logger, apperrors.Internal(fmt.Errorf("audit export query: %w", err)))
		return
	}
	defer rows.Close()

	filename := fmt.Sprintf("relatorio_auditoria_lai_%s.csv", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"ID", "Usuario_ID", "Acao", "Tipo_Recurso", "ID_Recurso", "IP", "Data_Hora"})

	for rows.Next() {
		var id, userID, action, resourceType, resourceID, ip string
		var createdAt time.Time
		if err := rows.Scan(&id, &userID, &action, &resourceType, &resourceID, &ip, &createdAt); err != nil {
			continue
		}
		_ = writer.Write([]string{
			id, userID, action, resourceType, resourceID, ip, createdAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
}
