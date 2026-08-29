// Package transport implementa o handler HTTP do módulo de contratos
// municipais: validar → autorizar → chamar service → serializar resposta.
package transport

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/modules/contratos/application"
	"github.com/yurythx/projeto-nova/internal/modules/contratos/domain"
	"github.com/yurythx/projeto-nova/internal/platform/auth"
	"github.com/yurythx/projeto-nova/pkg/httputil"
)

// Handlers agrupa os handlers HTTP do módulo de contratos.
type Handlers struct {
	service *application.Service
	logger  *slog.Logger
}

// NewHandlers constrói Handlers com as dependências injetadas.
func NewHandlers(service *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{service: service, logger: logger}
}

// CreateContrato trata POST /api/v1/contratos.
func (h *Handlers) CreateContrato(w http.ResponseWriter, r *http.Request) {
	var req CreateContratoRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	createdBy, subject, ok := auth.ActorUUID(r.Context())
	if !ok && subject != "" {
		h.logger.Warn("contratos: subject do token não é UUID — autoria não registrada", "subject", subject)
	}

	in := application.CreateInput{
		Numero:             req.Numero,
		Objeto:             req.Objeto,
		Contratante:        req.Contratante,
		Contratado:         req.Contratado,
		CNPJ:               req.CNPJ,
		Valor:              req.Valor,
		DataAssinatura:     parseDate(req.DataAssinatura),
		DataVigenciaInicio: parseDate(req.DataVigenciaInicio),
		DataVigenciaFim:    parseDate(req.DataVigenciaFim),
		CreatedBy:          createdBy,
	}

	c, err := h.service.Create(r.Context(), in)
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	httputil.WriteCreated(w, toContratoResponse(c))
}

// GetContrato trata GET /api/v1/contratos/{id}.
func (h *Handlers) GetContrato(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, r, h.logger, apperrors.BadRequest("id must be a valid UUID"))
		return
	}
	c, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	httputil.WriteOK(w, toContratoResponse(c))
}

// ListContratos trata GET /api/v1/contratos.
// Query params: status, busca, page, page_size.
func (h *Handlers) ListContratos(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	params := domain.ListParams{
		Status:   domain.Status(q.Get("status")),
		Busca:    q.Get("busca"),
		Page:     page,
		PageSize: pageSize,
	}

	contratos, total, err := h.service.List(r.Context(), params)
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	ps := params.PageSize
	if ps <= 0 {
		ps = 20
	}
	pages := int((total + int64(ps) - 1) / int64(ps))

	var items []ContratoResponse
	for _, c := range contratos {
		items = append(items, toContratoResponse(c))
	}
	httputil.WriteOK(w, ListContratosResponse{
		Data:  items,
		Total: total,
		Page:  params.Page,
		Pages: pages,
	})
}

// UpdateStatus trata PATCH /api/v1/contratos/{id}/status.
func (h *Handlers) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, r, h.logger, apperrors.BadRequest("id must be a valid UUID"))
		return
	}
	var req UpdateStatusRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	if err := h.service.UpdateStatus(r.Context(), id, domain.Status(req.Status)); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	httputil.WriteOK(w, struct{}{})
}

// KanbanView trata GET /api/v1/contratos/kanban.
// Query param: limit (por coluna, default 20).
func (h *Handlers) KanbanView(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	cols, err := h.service.KanbanView(r.Context(), limit)
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	httputil.WriteOK(w, toKanbanResponse(cols))
}

// AddDiarioRef trata POST /api/v1/contratos/{id}/diario-refs.
func (h *Handlers) AddDiarioRef(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, r, h.logger, apperrors.BadRequest("id must be a valid UUID"))
		return
	}
	var req AddDiarioRefRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	ref := domain.DiarioRef{
		ContratoID:    id,
		EditionNumber: req.EditionNumber,
		TipoEvento:    req.TipoEvento,
		PublicadoEm:   parseDate(req.PublicadoEm),
		Contexto:      req.Contexto,
		DocURL:        req.DocURL,
	}
	if err := h.service.AddDiarioRef(r.Context(), ref); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	httputil.WriteOK(w, struct{}{})
}

// GetDashboardStats trata GET /api/v1/contratos/dashboard.
func (h *Handlers) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetDashboardStats(r.Context())
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	httputil.WriteOK(w, stats)
}
