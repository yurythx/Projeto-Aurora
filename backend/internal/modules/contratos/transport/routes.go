package transport

import (
	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova/internal/platform/auth"
	"log/slog"
)

// RegisterRoutes monta as rotas do módulo de contratos no roteador.
// As rotas já estão dentro de um grupo autenticado (ver internal/app/router.go).
func RegisterRoutes(r chi.Router, h *Handlers, logger *slog.Logger) {
	// Visão Kanban — GET /api/v1/contratos/kanban (antes das rotas com {id})
	r.With(
		auth.RequirePermission(logger, auth.PermContratosRead),
	).Get("/contratos/kanban", h.KanbanView)

	// Dashboard Stats
	r.With(
		auth.RequirePermission(logger, auth.PermContratosRead),
	).Get("/contratos/dashboard", h.GetDashboardStats)

	// CRUD básico
	r.With(
		auth.RequirePermission(logger, auth.PermContratosManage),
	).Post("/contratos", h.CreateContrato)

	r.With(
		auth.RequirePermission(logger, auth.PermContratosRead),
	).Get("/contratos", h.ListContratos)

	r.With(
		auth.RequirePermission(logger, auth.PermContratosRead),
	).Get("/contratos/{id}", h.GetContrato)

	r.With(
		auth.RequirePermission(logger, auth.PermContratosManage),
	).Patch("/contratos/{id}/status", h.UpdateStatus)

	// Referências ao Diário Oficial
	r.With(
		auth.RequirePermission(logger, auth.PermContratosManage),
	).Post("/contratos/{id}/diario-refs", h.AddDiarioRef)
}
