package lgpd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-aurora/internal/domain/errors"
	"github.com/yurythx/projeto-aurora/internal/platform/auth"
	"github.com/yurythx/projeto-aurora/internal/platform/httpserver"
	"github.com/yurythx/projeto-aurora/pkg/httputil"
)

const CurrentTermVersion = "v1.0.0-2026"

type Service struct {
	db             *pgxpool.Pool
	logger         *slog.Logger
	trustedProxies []*net.IPNet
}

func NewService(db *pgxpool.Pool, logger *slog.Logger, trustedProxies []*net.IPNet) *Service {
	return &Service{db: db, logger: logger, trustedProxies: trustedProxies}
}

// HasAcceptedCurrentTerm reporta se o usuário já aceitou a versão mais recente dos termos.
func (s *Service) HasAcceptedCurrentTerm(ctx context.Context, userID uuid.UUID, termVersion string) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM user_consents WHERE user_id = $1 AND term_version = $2)`
	var exists bool
	err := s.db.QueryRow(ctx, q, userID, termVersion).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("lgpd: check consent: %w", err)
	}
	return exists, nil
}

// RecordConsent insere o registro de aceite de termos na tabela user_consents.
func (s *Service) RecordConsent(ctx context.Context, userID uuid.UUID, termVersion, ipAddress, userAgent string) error {
	const q = `
		INSERT INTO user_consents (user_id, term_version, ip_address, user_agent, accepted_at)
		VALUES ($1, $2, NULLIF($3, '')::inet, NULLIF($4, ''), now())
		ON CONFLICT (user_id, term_version) DO UPDATE SET accepted_at = now()
	`
	_, err := s.db.Exec(ctx, q, userID, termVersion, ipAddress, userAgent)
	if err != nil {
		return fmt.Errorf("lgpd: record consent: %w", err)
	}
	return nil
}

// RegisterRoutes registra os endpoints REST de consentimento LGPD. r já
// vem escopado em /api/v1 (ver internal/app/router.go) — os caminhos aqui
// são relativos a isso, nunca prefixados com /api/v1 de novo (achado de
// auditoria: os dois endpoints estavam registrados em
// /api/v1/api/v1/lgpd/*, inalcançáveis nos caminhos que o frontend/o
// openapi.yaml de fato documentam, silenciosamente quebrando o consentimento LGPD).
func (s *Service) RegisterRoutes(r interface {
	Get(path string, fn http.HandlerFunc)
	Post(path string, fn http.HandlerFunc)
}) {
	r.Get("/lgpd/status", s.handleStatus)
	r.Post("/lgpd/accept", s.handleAccept)
}

func (s *Service) handleStatus(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, r, s.logger, apperrors.Unauthorized("Não autenticado"))
		return
	}

	userID, err := uuid.Parse(identity.Subject)
	if err != nil {
		httputil.WriteOK(w, map[string]any{
			"accepted":     true,
			"term_version": CurrentTermVersion,
		})
		return
	}

	accepted, err := s.HasAcceptedCurrentTerm(r.Context(), userID, CurrentTermVersion)
	if err != nil {
		httputil.WriteError(w, r, s.logger, apperrors.Internal(err))
		return
	}

	httputil.WriteOK(w, map[string]any{
		"accepted":     accepted,
		"term_version": CurrentTermVersion,
	})
}

func (s *Service) handleAccept(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, r, s.logger, apperrors.Unauthorized("Não autenticado"))
		return
	}

	var req struct {
		TermVersion string `json:"term_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TermVersion == "" {
		req.TermVersion = CurrentTermVersion
	}

	userID, err := uuid.Parse(identity.Subject)
	if err != nil {
		httputil.WriteOK(w, map[string]any{
			"status":       "ok",
			"term_version": req.TermVersion,
		})
		return
	}

	// Gap G-04: antes lia o X-Forwarded-For cru — spoofável, e o IP entra
	// na prova de consentimento (art. 8º §1º). httpserver.ClientIP só
	// honra o XFF quando a conexão vem de um proxy reverso confiável
	// (TRUSTED_PROXIES); caso contrário devolve o RemoteAddr direto.
	ip := httpserver.ClientIP(r, s.trustedProxies)

	err = s.RecordConsent(r.Context(), userID, req.TermVersion, ip, r.UserAgent())
	if err != nil {
		httputil.WriteError(w, r, s.logger, apperrors.Internal(err))
		return
	}

	httputil.WriteOK(w, map[string]any{
		"status":       "ok",
		"term_version": req.TermVersion,
		"accepted_at":  time.Now().UTC(),
	})
}
