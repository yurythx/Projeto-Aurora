package lgpd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-aurora/internal/domain/errors"
	"github.com/yurythx/projeto-aurora/internal/platform/audit"
	"github.com/yurythx/projeto-aurora/internal/platform/auth"
	"github.com/yurythx/projeto-aurora/internal/platform/httpserver"
	"github.com/yurythx/projeto-aurora/pkg/httputil"
)

// Direitos do titular — LGPD art. 18 (F3.2 do roadmap de conformidade).
//
//   - GET  /api/v1/lgpd/meus-dados          → acesso + portabilidade
//     (art. 18, II e V): baixa, em JSON estruturado, TUDO que a
//     plataforma guarda sobre o próprio titular. Síncrono — o volume é o
//     de um único usuário.
//   - POST /api/v1/lgpd/solicitar-exclusao  → eliminação (art. 18, VI):
//     cria uma solicitação; o worker anonimiza (nunca DELETE — a linha é
//     mantida por integridade referencial e a trilha de auditoria é
//     preservada como registro legal da própria operação). 202 Accepted.
//   - GET  /api/v1/lgpd/minhas-solicitacoes → acompanhamento do prazo
//     legal de resposta (art. 19).

// resolveUserID mapeia a identidade autenticada para o id da linha em
// "users": num token local o Subject já É esse id; num token do Keycloak
// o Subject é o "sub" externo (coluna keycloak_subject).
func (s *Service) resolveUserID(ctx context.Context, identity auth.Identity) (uuid.UUID, error) {
	if id, err := uuid.Parse(identity.Subject); err == nil {
		var exists bool
		if qerr := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, id).Scan(&exists); qerr == nil && exists {
			return id, nil
		}
	}
	var id uuid.UUID
	err := s.db.QueryRow(ctx, `SELECT id FROM users WHERE keycloak_subject = $1`, identity.Subject).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, apperrors.NotFound("nenhuma conta local associada a esta identidade ainda")
		}
		return uuid.Nil, fmt.Errorf("lgpd: resolve user id: %w", err)
	}
	return id, nil
}

// RegisterDSRRoutes monta os endpoints de direitos do titular. r já vem
// escopado em /api/v1 e atrás de auth.RequireAuthentication.
func (s *Service) RegisterDSRRoutes(r interface {
	Get(path string, fn http.HandlerFunc)
	Post(path string, fn http.HandlerFunc)
}) {
	r.Get("/lgpd/meus-dados", s.handleExportMyData)
	r.Post("/lgpd/solicitar-exclusao", s.handleRequestErasure)
	r.Get("/lgpd/minhas-solicitacoes", s.handleListMyRequests)
}

type myDataPackage struct {
	GeneratedAt string           `json:"generated_at"`
	TermVersion string           `json:"current_term_version"`
	Account     map[string]any   `json:"account"`
	Consents    []map[string]any `json:"consents"`
	AuditTrail  []map[string]any `json:"audit_trail"`
	DSRRequests []map[string]any `json:"data_subject_requests"`
	Note        string           `json:"note"`
}

func (s *Service) handleExportMyData(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, r, s.logger, apperrors.Unauthorized("Não autenticado"))
		return
	}
	uid, err := s.resolveUserID(r.Context(), identity)
	if err != nil {
		httputil.WriteError(w, r, s.logger, err)
		return
	}
	ctx := r.Context()

	pkg := myDataPackage{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		TermVersion: CurrentTermVersion,
		Note: "Pacote de dados pessoais do titular (LGPD art. 18, II e V). " +
			"A trilha de auditoria e os registros de consentimento são mantidos " +
			"como registro legal mesmo após um pedido de eliminação.",
	}

	// Conta
	var (
		username, email, displayName string
		active                       bool
		createdAt                    time.Time
		lastSeen                     *time.Time
		roles                        []string
	)
	err = s.db.QueryRow(ctx, `
		SELECT username, email, display_name, active, created_at, last_seen_at, roles
		FROM users WHERE id = $1`, uid).
		Scan(&username, &email, &displayName, &active, &createdAt, &lastSeen, &roles)
	if err != nil {
		httputil.WriteError(w, r, s.logger, apperrors.Internal(fmt.Errorf("lgpd: export account: %w", err)))
		return
	}
	pkg.Account = map[string]any{
		"id": uid.String(), "username": username, "email": email,
		"display_name": displayName, "active": active, "roles": roles,
		"created_at": createdAt.UTC().Format(time.RFC3339),
	}
	if lastSeen != nil {
		pkg.Account["last_seen_at"] = lastSeen.UTC().Format(time.RFC3339)
	}

	pkg.Consents = s.collectRows(ctx, `
		SELECT term_version, COALESCE(ip_address::text,''), COALESCE(user_agent,''), accepted_at
		FROM user_consents WHERE user_id = $1 ORDER BY accepted_at DESC`, uid,
		"term_version", "ip_address", "user_agent", "accepted_at")

	pkg.AuditTrail = s.collectRows(ctx, `
		SELECT action, COALESCE(resource_type,''), COALESCE(resource_id,''),
		       COALESCE(metadata::text,'{}'), COALESCE(ip_address::text,''), created_at
		FROM audit_logs WHERE user_id = $1 ORDER BY created_at DESC LIMIT 5000`, uid,
		"action", "resource_type", "resource_id", "metadata", "ip_address", "created_at")

	pkg.DSRRequests = s.collectRows(ctx, `
		SELECT kind, status, COALESCE(detail,''), created_at, completed_at
		FROM data_subject_requests WHERE user_id = $1 ORDER BY created_at DESC`, uid,
		"kind", "status", "detail", "created_at", "completed_at")

	filename := fmt.Sprintf("meus-dados-aurora-%s.json", time.Now().UTC().Format("20060102_150405"))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Cache-Control", "no-store")
	httputil.WriteJSON(w, http.StatusOK, pkg, nil)

	entry := audit.FromRequest(r)
	entry.UserID = &uid
	entry.Action = "lgpd.data_export.self"
	entry.ResourceType = "user"
	entry.ResourceID = uid.String()
	if aerr := audit.NewWriter(s.db).Record(r.Context(), entry); aerr != nil {
		s.logger.Warn("lgpd: falha ao auditar exportação de dados do titular", "error", aerr)
	}
}

// collectRows roda uma query de N colunas de texto/tempo e devolve
// []map[coluna]valor — usado só para montar o pacote de exportação, onde
// o formato de saída é JSON genérico.
func (s *Service) collectRows(ctx context.Context, query string, arg any, cols ...string) []map[string]any {
	out := []map[string]any{}
	rows, err := s.db.Query(ctx, query, arg)
	if err != nil {
		s.logger.Warn("lgpd: export sub-query falhou", "error", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			continue
		}
		m := make(map[string]any, len(cols))
		for i, c := range cols {
			if i >= len(vals) {
				break
			}
			if t, ok := vals[i].(time.Time); ok {
				m[c] = t.UTC().Format(time.RFC3339)
			} else {
				m[c] = vals[i]
			}
		}
		out = append(out, m)
	}
	return out
}

func (s *Service) handleRequestErasure(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, r, s.logger, apperrors.Unauthorized("Não autenticado"))
		return
	}
	uid, err := s.resolveUserID(r.Context(), identity)
	if err != nil {
		httputil.WriteError(w, r, s.logger, err)
		return
	}

	// Já existe uma solicitação em aberto? Devolve ela — o endpoint é
	// idempotente do ponto de vista do titular.
	var existingID, existingStatus string
	err = s.db.QueryRow(r.Context(), `
		SELECT id::text, status FROM data_subject_requests
		WHERE user_id = $1 AND kind = 'erasure' AND status IN ('pending','processing')
		ORDER BY created_at DESC LIMIT 1`, uid).Scan(&existingID, &existingStatus)
	if err == nil {
		httputil.WriteJSON(w, http.StatusOK, map[string]any{
			"request_id": existingID, "status": existingStatus,
			"message": "Já há uma solicitação de exclusão em andamento para esta conta.",
		}, nil)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		httputil.WriteError(w, r, s.logger, apperrors.Internal(fmt.Errorf("lgpd: check pending erasure: %w", err)))
		return
	}

	// O trigger da migration 000005 grava a linha em audit_logs.
	var newID string
	err = s.db.QueryRow(r.Context(), `
		INSERT INTO data_subject_requests (user_id, kind, status, requested_ip)
		VALUES ($1, 'erasure', 'pending', NULLIF($2,'')::inet)
		RETURNING id::text`, uid, httpserver.ClientIP(r, s.trustedProxies)).Scan(&newID)
	if err != nil {
		httputil.WriteError(w, r, s.logger, apperrors.Internal(fmt.Errorf("lgpd: create erasure request: %w", err)))
		return
	}

	httputil.WriteJSON(w, http.StatusAccepted, map[string]any{
		"request_id": newID,
		"status":     "pending",
		"message": "Solicitação registrada. A anonimização é processada automaticamente; " +
			"acompanhe em /api/v1/lgpd/minhas-solicitacoes.",
	}, nil)
}

func (s *Service) handleListMyRequests(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, r, s.logger, apperrors.Unauthorized("Não autenticado"))
		return
	}
	uid, err := s.resolveUserID(r.Context(), identity)
	if err != nil {
		httputil.WriteError(w, r, s.logger, err)
		return
	}
	list := s.collectRows(r.Context(), `
		SELECT id::text, kind, status, COALESCE(detail,''), created_at, completed_at
		FROM data_subject_requests WHERE user_id = $1 ORDER BY created_at DESC`, uid,
		"id", "kind", "status", "detail", "created_at", "completed_at")
	httputil.WriteOK(w, list)
}
