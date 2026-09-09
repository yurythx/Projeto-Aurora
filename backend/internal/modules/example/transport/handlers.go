package transport

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-aurora/internal/domain/errors"
	"github.com/yurythx/projeto-aurora/internal/modules/example/application"
	"github.com/yurythx/projeto-aurora/internal/modules/example/domain"
	"github.com/yurythx/projeto-aurora/pkg/httputil"
)

type Handlers struct {
	service *application.Service
	logger  *slog.Logger
}

func NewHandlers(service *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{
		service: service,
		logger:  logger,
	}
}

func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateItemRequest
	// Blueprint (gap G-06): DecodeJSON aplica MaxBytesReader (1 MiB) +
	// DisallowUnknownFields; Validate confere as struct-tags. Nenhum
	// módulo deve usar json.NewDecoder cru como este handler fazia.
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	if err := httputil.Validate(req); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	item, err := h.service.CreateItem(r.Context(), req.Title, req.Description)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			httputil.WriteError(w, r, h.logger, apperrors.BadRequest(err.Error()))
			return
		}
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	httputil.WriteCreated(w, item)
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.WriteError(w, r, h.logger, apperrors.BadRequest("ID inválido"))
		return
	}

	item, err := h.service.GetItem(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrExampleNotFound) {
			httputil.WriteError(w, r, h.logger, apperrors.NotFound("Item não encontrado"))
			return
		}
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	httputil.WriteOK(w, item)
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	items, total, err := h.service.ListItems(r.Context(), page, pageSize)
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	httputil.WriteOKWithMeta(w, items, map[string]interface{}{
		"total": total,
		"page":  page,
	})
}
