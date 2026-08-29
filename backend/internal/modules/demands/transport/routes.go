package transport

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova/internal/modules/demands/application"
	"github.com/yurythx/projeto-nova/internal/platform/auth"
)

// RegisterRoutes registra as rotas de demandas no router principal (já
// dentro do grupo autenticado — ver internal/app/router.go).
//
// Separação de papéis (IN SCL 01/2019): LER o quadro, o checklist e o
// histórico exige demands:read; MOVER um card de etapa e ANEXAR documentos
// de compliance são ações fiscais e exigem demands:manage. O nova-auditor
// enxerga tudo mas não move nada; o nova-admin tem as duas implicitamente.
func RegisterRoutes(r chi.Router, h *Handlers, logger *slog.Logger) {
	read := auth.RequirePermission(logger, auth.PermDemandsRead)
	manage := auth.RequirePermission(logger, auth.PermDemandsManage)

	// Leitura
	r.With(read).Get("/demands/kanban", h.KanbanView)
	r.With(read).Get("/demands/dashboard", h.Dashboard)
	r.With(read).Get("/demands/occurrences", h.ListOccurrences)
	r.With(read).Get("/demands/{id}", h.GetByID)
	r.With(read).Get("/demands/{id}/requirements", h.NextRequirements)
	r.With(read).Get("/demands/{id}/history", h.GetHistory)
	r.With(read).Get("/demands/{id}/package.zip", h.DownloadPackage)
	r.With(read).Get("/demands/{id}/oficio.pdf", h.DownloadPDF(application.PDFOficio))
	r.With(read).Get("/demands/{id}/ordem-servico.pdf", h.DownloadPDF(application.PDFOrdemServico))
	r.With(read).Get("/demands/{id}/relatorio.pdf", h.DownloadPDF(application.PDFRelatorio))

	// Ações fiscais (escrita)
	r.With(manage).Post("/demands", h.CreateDemand)
	r.With(manage).Patch("/demands/{id}/etapa", h.MoveKanbanCard)
	r.With(manage).Get("/demands/{id}/documents/upload-url", h.GetUploadURL)
	r.With(manage).Post("/demands/{id}/documents", h.ConfirmUpload)
}
