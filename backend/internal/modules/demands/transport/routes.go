package transport

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registra as rotas de demandas no router principal.
func RegisterRoutes(r chi.Router, h *Handlers) {
	// Apenas usuários logados e com regra de leitura/escrita podem mover as demandas.
	// Regras ABAC (Attribute-Based Access Control) ou RBAC (Role-Based).
	r.Group(func(r chi.Router) {
		
		// Kanban
		r.Get("/demands/kanban", h.KanbanView)
		r.Post("/demands", h.CreateDemand)
		r.Get("/demands/{id}", h.GetByID)
		r.Patch("/demands/{id}/etapa", h.MoveKanbanCard)

		// Documentos / MinIO
		r.Get("/demands/{id}/documents/upload-url", h.GetUploadURL)
		r.Post("/demands/{id}/documents", h.ConfirmUpload)
	})
}
