package audit

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-aurora/internal/domain/errors"
	"github.com/yurythx/projeto-aurora/internal/platform/auth"
	"github.com/yurythx/projeto-aurora/pkg/httputil"
)

// maxExportWindow limita o intervalo [from,to] de um único relatório. Um
// pedido sem janela usa os últimos 30 dias; um pedido com janela maior
// que isto é recusado — quem precisa do histórico completo pagina com
// ?cursor (X-Next-Cursor).
const maxExportWindow = 366 * 24 * time.Hour

// maxExportRows é o teto de linhas por página. O relatório completo é
// obtido seguindo o cursor keyset devolvido em X-Next-Cursor.
const maxExportRows = 20000

const defaultExportWindow = 30 * 24 * time.Hour

type Exporter struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewExporter(db *pgxpool.Pool, logger *slog.Logger) *Exporter {
	return &Exporter{db: db, logger: logger}
}

// RegisterRoutes monta a exportação da trilha de auditoria (LAI), restrita
// a quem tem auth.PermAuditRead. r já vem escopado em /api/v1.
//
// Gaps G-09 e G-12 da auditoria de conformidade:
//   - G-12: antes era CSV fixo, LIMIT 1000 hard-coded, sem filtro de
//     período. Agora aceita ?from=&to= (RFC3339 ou AAAA-MM-DD), ?action=,
//     paginação keyset por ?cursor, e negociação de formato
//     (csv | json | xml) por ?format= ou pelo header Accept.
//   - G-09: cada exportação bem-sucedida gera ela mesma uma linha
//     "audit.exported" na trilha — consultar dado de transparência
//     também deixa rastro.
func (e *Exporter) RegisterRoutes(r chi.Router) {
	r.With(auth.RequirePermission(e.logger, auth.PermAuditRead)).Get("/audit/export", e.handleExport)
}

type exportRow struct {
	ID            string `json:"id" xml:"id"`
	UserID        string `json:"user_id" xml:"user_id"`
	Action        string `json:"action" xml:"action"`
	ResourceType  string `json:"resource_type" xml:"resource_type"`
	ResourceID    string `json:"resource_id" xml:"resource_id"`
	CorrelationID string `json:"correlation_id" xml:"correlation_id"`
	IPAddress     string `json:"ip_address" xml:"ip_address"`
	Metadata      string `json:"metadata" xml:"metadata"`
	CreatedAt     string `json:"created_at" xml:"created_at"`
}

type exportMeta struct {
	Dataset      string `json:"dataset" xml:"dataset"`
	WindowFrom   string `json:"window_from" xml:"window_from"`
	WindowTo     string `json:"window_to" xml:"window_to"`
	ActionFilter string `json:"action_filter,omitempty" xml:"action_filter,omitempty"`
	Rows         int    `json:"rows" xml:"rows"`
	NextCursor   string `json:"next_cursor,omitempty" xml:"next_cursor,omitempty"`
	GeneratedAt  string `json:"generated_at" xml:"generated_at"`
}

type exportEnvelope struct {
	XMLName xml.Name    `json:"-" xml:"relatorio_auditoria"`
	Meta    exportMeta  `json:"dataset" xml:"dataset"`
	Rows    []exportRow `json:"rows" xml:"registros>registro"`
}

func (e *Exporter) handleExport(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.IdentityFromContext(r.Context()); !ok {
		httputil.WriteError(w, r, e.logger, apperrors.Unauthorized("Não autenticado"))
		return
	}

	from, to, err := parseWindow(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		httputil.WriteError(w, r, e.logger, apperrors.BadRequest(err.Error()))
		return
	}
	actionFilter := strings.TrimSpace(r.URL.Query().Get("action"))
	format := chooseFormat(r)

	cursorTime, cursorID, err := parseCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		httputil.WriteError(w, r, e.logger, apperrors.BadRequest("cursor inválido"))
		return
	}

	// Keyset: ORDER BY created_at DESC, id DESC. O cursor aponta para a
	// última linha da página anterior; a próxima começa estritamente
	// depois dela nessa ordem. LIMIT maxExportRows+1 para saber se há
	// mais uma página sem uma segunda query.
	var sb strings.Builder
	sb.WriteString(`
		SELECT id, COALESCE(user_id::text,'SISTEMA'), action,
		       COALESCE(resource_type,''), COALESCE(resource_id,''),
		       COALESCE(correlation_id::text,''), COALESCE(ip_address::text,''),
		       COALESCE(metadata::text,'{}'), created_at
		FROM audit_logs
		WHERE created_at >= $1 AND created_at < $2`)
	args := []any{from, to}
	if actionFilter != "" {
		args = append(args, actionFilter)
		sb.WriteString(fmt.Sprintf(" AND action = $%d", len(args)))
	}
	if !cursorTime.IsZero() {
		args = append(args, cursorTime, cursorID)
		sb.WriteString(fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", len(args)-1, len(args)))
	}
	sb.WriteString(fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT %d", maxExportRows+1))

	rows, err := e.db.Query(r.Context(), sb.String(), args...)
	if err != nil {
		httputil.WriteError(w, r, e.logger, apperrors.Internal(fmt.Errorf("audit export query: %w", err)))
		return
	}
	defer rows.Close()

	collected := make([]exportRow, 0, 512)
	for rows.Next() {
		var rw exportRow
		var createdAt time.Time
		if err := rows.Scan(&rw.ID, &rw.UserID, &rw.Action, &rw.ResourceType, &rw.ResourceID,
			&rw.CorrelationID, &rw.IPAddress, &rw.Metadata, &createdAt); err != nil {
			continue
		}
		rw.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		collected = append(collected, rw)
	}
	if rows.Err() != nil {
		httputil.WriteError(w, r, e.logger, apperrors.Internal(fmt.Errorf("audit export scan: %w", rows.Err())))
		return
	}

	nextCursor := ""
	if len(collected) > maxExportRows {
		collected = collected[:maxExportRows]
		last := collected[len(collected)-1]
		if lt, perr := time.Parse(time.RFC3339Nano, last.CreatedAt); perr == nil {
			nextCursor = encodeCursor(lt, last.ID)
		}
	}

	meta := exportMeta{
		Dataset:      "trilha_de_auditoria_lai",
		WindowFrom:   from.UTC().Format(time.RFC3339),
		WindowTo:     to.UTC().Format(time.RFC3339),
		ActionFilter: actionFilter,
		Rows:         len(collected),
		NextCursor:   nextCursor,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	// Metadados do dataset em cabeçalhos (padrão Dados Abertos — úteis
	// para consumo automatizado sem parsear o corpo).
	h := w.Header()
	h.Set("X-Dataset-Name", meta.Dataset)
	h.Set("X-Dataset-Window-From", meta.WindowFrom)
	h.Set("X-Dataset-Window-To", meta.WindowTo)
	h.Set("X-Dataset-Rows", strconv.Itoa(meta.Rows))
	h.Set("X-Generated-At", meta.GeneratedAt)
	if nextCursor != "" {
		h.Set("X-Next-Cursor", nextCursor)
	}

	stamp := time.Now().UTC().Format("20060102_150405")
	switch format {
	case "json":
		h.Set("Content-Type", "application/json; charset=utf-8")
		h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="relatorio_auditoria_lai_%s.json"`, stamp))
		_ = json.NewEncoder(w).Encode(exportEnvelope{Meta: meta, Rows: collected})
	case "xml":
		h.Set("Content-Type", "application/xml; charset=utf-8")
		h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="relatorio_auditoria_lai_%s.xml"`, stamp))
		_, _ = w.Write([]byte(xml.Header))
		enc := xml.NewEncoder(w)
		enc.Indent("", "  ")
		_ = enc.Encode(exportEnvelope{Meta: meta, Rows: collected})
	default: // csv
		h.Set("Content-Type", "text/csv; charset=utf-8")
		h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="relatorio_auditoria_lai_%s.csv"`, stamp))
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"ID", "Usuario_ID", "Acao", "Tipo_Recurso", "ID_Recurso", "Correlation_ID", "IP", "Metadata", "Data_Hora_UTC"})
		for _, rw := range collected {
			_ = cw.Write([]string{rw.ID, rw.UserID, rw.Action, rw.ResourceType, rw.ResourceID, rw.CorrelationID, rw.IPAddress, rw.Metadata, rw.CreatedAt})
		}
		cw.Flush()
	}

	// G-09: a própria exportação vira um fato auditável. Depois de já ter
	// escrito o corpo — a falha ao gravar a trilha não deve alterar a
	// resposta que o cliente recebeu.
	entry := FromRequest(r)
	entry.Action = "audit.exported"
	entry.ResourceType = "audit_logs"
	entry.ResourceID = meta.Dataset
	if id, ok := auth.IdentityFromContext(r.Context()); ok {
		if uid, perr := uuid.Parse(id.Subject); perr == nil {
			entry.UserID = &uid
		}
	}
	entry.Metadata = map[string]any{
		"format": format, "window_from": meta.WindowFrom, "window_to": meta.WindowTo,
		"action_filter": actionFilter, "rows": meta.Rows, "paginated": nextCursor != "",
	}
	if err := NewWriter(e.db).Record(r.Context(), entry); err != nil {
		e.logger.Warn("audit: falha ao registrar a própria exportação", slog.Any("error", err))
	}
}

// parseWindow interpreta from/to. Aceita RFC3339 ou "AAAA-MM-DD" (início
// do dia, UTC). Vazios: últimos defaultExportWindow. Recusa janela maior
// que maxExportWindow ou invertida.
func parseWindow(fromRaw, toRaw string) (from, to time.Time, err error) {
	now := time.Now().UTC()
	to = now
	from = now.Add(-defaultExportWindow)

	if toRaw != "" {
		if to, err = parseDateOrRFC3339(toRaw); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("parâmetro 'to' inválido (use AAAA-MM-DD ou RFC3339)")
		}
	}
	if fromRaw != "" {
		if from, err = parseDateOrRFC3339(fromRaw); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("parâmetro 'from' inválido (use AAAA-MM-DD ou RFC3339)")
		}
	} else if toRaw != "" {
		from = to.Add(-defaultExportWindow)
	}

	if !from.Before(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("'from' deve ser anterior a 'to'")
	}
	if to.Sub(from) > maxExportWindow {
		return time.Time{}, time.Time{}, fmt.Errorf("janela máxima por relatório é de 366 dias — para o histórico completo, pagine com ?cursor")
	}
	return from, to, nil
}

func parseDateOrRFC3339(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// chooseFormat: ?format= tem prioridade sobre o header Accept.
func chooseFormat(r *http.Request) string {
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format"))) {
	case "json":
		return "json"
	case "xml":
		return "xml"
	case "csv":
		return "csv"
	}
	accept := r.Header.Get("Accept")
	switch {
	case strings.Contains(accept, "application/json"):
		return "json"
	case strings.Contains(accept, "application/xml"), strings.Contains(accept, "text/xml"):
		return "xml"
	default:
		return "csv"
	}
}

func encodeCursor(t time.Time, id string) string {
	return t.UTC().Format(time.RFC3339Nano) + "," + id
}

func parseCursor(raw string) (time.Time, string, error) {
	if raw == "" {
		return time.Time{}, "", nil
	}
	parts := strings.SplitN(raw, ",", 2)
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("formato")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return time.Time{}, "", err
	}
	return t.UTC(), parts[1], nil
}
