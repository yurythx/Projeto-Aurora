package transport

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/modules/demands/application"
	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
	"github.com/yurythx/projeto-nova/pkg/httputil"
)

// Handlers agrupa os controladores REST das Demandas.
type Handlers struct {
	service *application.Service
	logger  *slog.Logger
}

// NewHandlers constrói a struct de Handlers.
func NewHandlers(service *application.Service, logger *slog.Logger) *Handlers {
	return &Handlers{service: service, logger: logger}
}

// CreateDemand cria uma nova demanda mensal (POST /api/v1/demands).
func (h *Handlers) CreateDemand(w http.ResponseWriter, r *http.Request) {
	var req CreateDemandRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	if req.ContratoID == uuid.Nil {
		httputil.WriteError(w, r, h.logger, apperrors.BadRequest("contrato_id é obrigatório"))
		return
	}
	if req.AnoMes == "" {
		httputil.WriteError(w, r, h.logger, apperrors.BadRequest("ano_mes é obrigatório (ex: 2026-08)"))
		return
	}

	demand, err := h.service.CreateDemand(r.Context(), application.CreateDemandInput{
		ContratoID:  req.ContratoID,
		AnoMes:      req.AnoMes,
		Observacoes: req.Observacoes,
	})
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	httputil.WriteCreated(w, toDemandResponse(demand))
}

// MoveKanbanCard trata o arrasto do card no Kanban. (PATCH /api/v1/demands/{id}/etapa)
func (h *Handlers) MoveKanbanCard(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, r, h.logger, apperrors.BadRequest("id da demanda inválido"))
		return
	}

	var req UpdateEtapaRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	// Executa a Máquina de Estados através do Serviço
	err = h.service.MoveKanbanCard(r.Context(), id, req.TargetEtapa)
	if err != nil {
		// Retornamos como erro de validação (400) para o Frontend exibir no Modal.
		httputil.WriteError(w, r, h.logger, apperrors.Validation(err.Error()))
		return
	}

	httputil.WriteOK(w, map[string]string{"message": "transição autorizada e efetivada"})
}

// KanbanView trata GET /api/v1/demands/kanban.
// Retorna todas as demandas em andamento estruturadas nas 6 colunas.
func (h *Handlers) KanbanView(w http.ResponseWriter, r *http.Request) {
	// Chamada ao service (precisamos implementar ListKanbanView no service)
	cols, err := h.service.KanbanView(r.Context())
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	// Montamos as 6 colunas estáticas
	columns := []map[string]interface{}{}
	labels := map[int]string{
		1: "Elaborar OF / Pré-Empenho",
		2: "Tramitar Planejamento",
		3: "Emitir OS / Envio Empresa",
		4: "Execução e Recepção",
		5: "Relatório Pgto / Certidões",
		6: "Contabilidade",
	}

	for i := 1; i <= 6; i++ {
		items := []DemandResponse{}
		for _, d := range cols[domain.EtapaKanban(i)] {
			items = append(items, toDemandResponse(d))
		}
		
		columns = append(columns, map[string]interface{}{
			"status": fmt.Sprintf("%d", i),
			"label":  labels[i],
			"total":  len(items),
			"items":  items,
		})
	}

	httputil.WriteOK(w, map[string]interface{}{
		"columns": columns,
	})
}

// GetUploadURL gera uma URL pré-assinada do MinIO para o frontend fazer o upload direto.
func (h *Handlers) GetUploadURL(w http.ResponseWriter, r *http.Request) {
	demandID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, r, h.logger, fmt.Errorf("uuid inválido: %w", err))
		return
	}

	docType := r.URL.Query().Get("doc_type")
	fileName := r.URL.Query().Get("file_name")
	if docType == "" || fileName == "" {
		httputil.WriteError(w, r, h.logger, fmt.Errorf("doc_type e file_name são obrigatórios"))
		return
	}

	uploadURL, objectPath, err := h.service.GenerateUploadURL(r.Context(), demandID, docType, fileName)
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	httputil.WriteOK(w, map[string]string{
		"upload_url":  uploadURL,
		"file_path":   objectPath,
	})
}

// ConfirmUpload registra que o documento foi salvo com sucesso no MinIO e adiciona a prova no banco de dados.
func (h *Handlers) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	demandID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, r, h.logger, fmt.Errorf("uuid inválido: %w", err))
		return
	}

	var req struct {
		DocType  string `json:"doc_type"`
		FilePath string `json:"file_path"`
		FileName string `json:"file_name"`
	}
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	if err := h.service.ConfirmUpload(r.Context(), demandID, req.DocType, req.FilePath, req.FileName); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	httputil.WriteOK(w, map[string]string{"message": "documento anexado com sucesso"})
}

// GetByID retorna os detalhes de uma demanda específica.
func (h *Handlers) GetByID(w http.ResponseWriter, r *http.Request) {
	demandID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, r, h.logger, fmt.Errorf("uuid inválido: %w", err))
		return
	}

	demand, err := h.service.GetDemandByID(r.Context(), demandID)
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	httputil.WriteOK(w, toDemandResponse(demand))
}
