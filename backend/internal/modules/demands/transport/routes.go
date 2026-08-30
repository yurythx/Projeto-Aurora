package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova/internal/modules/demands/application"
	"github.com/yurythx/projeto-nova/internal/platform/auth"
	"github.com/yurythx/projeto-nova/internal/platform/httpserver"
)

// RegisterRoutes registra as rotas de demandas no router principal (já
// dentro do grupo autenticado — ver internal/app/router.go).
//
// Separação de papéis (IN SCL 01/2019): LER o quadro, o checklist e o
// histórico exige demands:read; MOVER um card de etapa e ANEXAR documentos
// de compliance são ações fiscais e exigem demands:manage. O nova-auditor
// enxerga tudo mas não move nada; o nova-admin tem as duas implicitamente.
//
// As escritas passam ainda por rate limiting por usuário (limiter), para
// que um token comprometido não consiga varrer/spammar o Kanban.
func RegisterRoutes(r chi.Router, h *Handlers, logger *slog.Logger, limiter httpserver.Limiter) {
	read := auth.RequirePermission(logger, auth.PermDemandsRead)
	manage := auth.RequirePermission(logger, auth.PermDemandsManage)
	rate := httpserver.RateLimit(logger, limiter, RateLimitKey)

	// Leitura
	r.With(read).Get("/demands/kanban", h.KanbanView)
	r.With(read).Get("/demands/dashboard", h.Dashboard)
	r.With(read).Get("/demands/occurrences", h.ListOccurrences)
	r.With(read).Get("/demands/{id}", h.GetByID)
	r.With(read).Get("/demands/{id}/requirements", h.NextRequirements)
	r.With(read).Get("/demands/{id}/history", h.GetHistory)
	r.With(read).Get("/demands/{id}/package.zip", h.DownloadPackage)
	// O PDF unificado concatena anexos + docs gerados via pdfcpu — mais
	// pesado que o .zip, então entra com rate limit como os outros PDFs.
	r.With(read, rate).Get("/demands/{id}/package.pdf", h.DownloadPackagePDF)
	// PDFs são gerados na hora (fpdf monta o doc em memória): leitura, mas
	// com rate limit para não virar vetor de CPU.
	r.With(read, rate).Get("/demands/{id}/oficio.pdf", h.DownloadPDF(application.PDFOficio))
	r.With(read, rate).Get("/demands/{id}/ordem-servico.pdf", h.DownloadPDF(application.PDFOrdemServico))
	r.With(read, rate).Get("/demands/{id}/relatorio.pdf", h.DownloadPDF(application.PDFRelatorio))

	// Ações fiscais (escrita) — manage + rate limit por usuário
	r.With(manage, rate).Post("/demands", h.CreateDemand)
	r.With(manage, rate).Patch("/demands/{id}/etapa", h.MoveKanbanCard)
	r.With(manage, rate).Get("/demands/{id}/documents/upload-url", h.GetUploadURL)
	r.With(manage, rate).Post("/demands/{id}/documents", h.ConfirmUpload)
}

// RateLimitKey limita por usuário autenticado; cai para o IP se anônimo
// (não deveria ocorrer atrás do middleware de auth).
func RateLimitKey(r *http.Request) string {
	if identity, ok := auth.IdentityFromContext(r.Context()); ok && identity.Subject != "" {
		return identity.Subject
	}
	return httpserver.ClientIPKey(r)
}
