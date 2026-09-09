package transport

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-aurora/internal/platform/auth"
)

// RegisterRoutes monta as rotas do módulo integrations num router já
// protegido por auth.RequireAuthentication.
//
// Gap G-05 da auditoria de conformidade: as duas rotas só exigiam
// autenticação, embora auth.PermIntegrationsRead já existisse — a lista
// de integrações e seus status revelam detalhes operacionais internos
// (quais provedores externos a plataforma consome, se estão no ar), o
// mesmo argumento que já restringia a lista de feature flags. Agora
// exigem integrations:read (aurora-integration-manager, aurora-auditor ou
// aurora-admin).
func RegisterRoutes(r chi.Router, h *Handlers, logger *slog.Logger) {
	r.Group(func(protected chi.Router) {
		protected.Use(auth.RequirePermission(logger, auth.PermIntegrationsRead))
		protected.Get("/integrations", h.ListIntegrations)
		protected.Get("/integrations/{id}/status", h.GetIntegrationStatus)
	})
}
