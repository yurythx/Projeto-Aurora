// Package audit grava a trilha de audit_logs (§49): login/logout,
// user.created/updated, integration.test, integration.config.changed,
// job.created/completed/failed. Quem chama é responsável por nunca
// colocar uma senha, access/refresh token, client secret ou segredo de
// integração em Entry.Metadata — este pacote não tenta redigir/mascarar
// conteúdo cujo formato ele não conhece; a auditoria imutável (migration
// 000008, que bloqueia UPDATE/DELETE/TRUNCATE em audit_logs) protege
// contra alteração posterior, não contra segredo indevidamente gravado.
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// Nomes de ação bem conhecidos (§49). Quem chama também pode usar suas
// próprias strings "<contexto>.<ação>" para eventos específicos de módulo
// não listados aqui.
const (
	ActionLogin                    = "login"
	ActionLogout                   = "logout"
	ActionUserCreated              = "user.created"
	ActionUserUpdated              = "user.updated"
	ActionIntegrationTest          = "integration.test"
	ActionIntegrationConfigChanged = "integration.config.changed"
	ActionJobCreated               = "job.created"
	ActionJobCompleted             = "job.completed"
	ActionJobFailed                = "job.failed"
	// ActionLGPDConsentGiven registra a aceitação formal dos Termos de Uso
	// e Política de Privacidade de Dados (Lei 13.709/2018 - LGPD).
	ActionLGPDConsentGiven = "lgpd_consent_given"
)

// Entry é um registro de auditoria.
type Entry struct {
	UserID        *uuid.UUID
	Action        string
	ResourceType  string
	ResourceID    string
	Metadata      map[string]any
	CorrelationID *uuid.UUID
	IPAddress     string
}

// execer é satisfeita tanto por *pgxpool.Pool quanto por pgx.Tx, para que
// Record possa ser usado tanto isoladamente quanto atomicamente junto de
// outras escritas numa transação já existente (ex.: gravar o audit_log de
// "user.created" na mesma transação que insere a linha do usuário).
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Writer insere linhas em audit_logs.
type Writer struct {
	db execer
}

// NewWriter constrói um Writer sobre db, que pode ser um *pgxpool.Pool
// para escritas isoladas ou um pgx.Tx para tornar a entrada de auditoria
// atômica com outras mudanças na mesma transação.
func NewWriter(db execer) *Writer {
	return &Writer{db: db}
}

// Record insere entry.
func (w *Writer) Record(ctx context.Context, entry Entry) error {
	metadata := entry.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("audit: marshal metadata: %w", err)
	}

	const q = `
		INSERT INTO audit_logs (id, user_id, action, resource_type, resource_id, metadata, correlation_id, ip_address, created_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, $7, NULLIF($8, '')::inet, now())
	`
	_, err = w.db.Exec(ctx, q,
		uuid.New(), entry.UserID, entry.Action, entry.ResourceType, entry.ResourceID,
		metadataJSON, entry.CorrelationID, sanitizeIP(entry.IPAddress),
	)
	if err != nil {
		return fmt.Errorf("audit: insert entry for action %s: %w", entry.Action, err)
	}
	return nil
}

// sanitizeIP normaliza o que vem em Entry.IPAddress para algo que a coluna
// `inet` aceite: aceita um IP puro, tira a porta de um "host:porta" e, se
// mesmo assim não for um IP válido, devolve "" (a query grava NULL). Um
// valor inválido nunca deve derrubar o INSERT de auditoria — que pode
// estar rodando dentro da transação de negócio e a levaria junto no
// rollback.
func sanitizeIP(raw string) string {
	if raw == "" {
		return ""
	}
	if ip := net.ParseIP(raw); ip != nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
	}
	return ""
}
