package transport

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

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

	now := time.Now()
	for i := 1; i <= 6; i++ {
		items := []DemandResponse{}
		for _, d := range cols[domain.EtapaKanban(i)] {
			resp := toDemandResponseAt(d, now)
			if d.Etapa < domain.Etapa6Contabilidade {
				chk := application.CheckAdvance(d, d.Etapa+1, now)
				resp.NextRequirements = &chk
			}
			items = append(items, resp)
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
		"upload_url": uploadURL,
		"file_path":  objectPath,
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
		DocType     string `json:"doc_type"`
		FilePath    string `json:"file_path"`
		FileName    string `json:"file_name"`
		ValidadeAte string `json:"validade_ate,omitempty"` // YYYY-MM-DD (certidões)
	}
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	var validade *time.Time
	if s := strings.TrimSpace(req.ValidadeAte); s != "" {
		t, perr := time.Parse("2006-01-02", s)
		if perr != nil {
			httputil.WriteError(w, r, h.logger, apperrors.BadRequest("validade_ate deve estar no formato AAAA-MM-DD"))
			return
		}
		validade = &t
	}

	if err := h.service.ConfirmUpload(r.Context(), demandID, req.DocType, req.FilePath, req.FileName, validade); err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	httputil.WriteOK(w, map[string]string{"message": "documento anexado com sucesso"})
}

// ListOccurrences devolve as pendências abertas (GET /demands/occurrences).
func (h *Handlers) ListOccurrences(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.OpenOccurrences(r.Context(), 200)
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	httputil.WriteOK(w, items)
}

// NextRequirements devolve o checklist da próxima etapa (GET /demands/{id}/requirements).
func (h *Handlers) NextRequirements(w http.ResponseWriter, r *http.Request) {
	demandID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, r, h.logger, apperrors.BadRequest("id da demanda inválido"))
		return
	}
	check, err := h.service.NextRequirements(r.Context(), demandID)
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	httputil.WriteOK(w, check)
}

// DownloadPDF gera e transmite um documento oficial da demanda em PDF
// (GET /demands/{id}/{oficio|ordem-servico|relatorio}.pdf).
func (h *Handlers) DownloadPDF(kind application.PDFKind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		demandID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			httputil.WriteError(w, r, h.logger, apperrors.BadRequest("id da demanda inválido"))
			return
		}
		demand, fileName, err := h.service.LoadDemandForPDF(r.Context(), demandID, kind)
		if err != nil {
			httputil.WriteError(w, r, h.logger, err)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename=%q`, fileName))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if err := application.RenderDemandPDF(kind, demand, w); err != nil {
			h.logger.Error("falha ao gerar PDF da demanda",
				slog.String("demanda_id", demandID.String()), slog.String("kind", string(kind)), slog.Any("error", err))
		}
	}
}

// DownloadPackage transmite um .zip com todos os anexos da demanda, na
// ordem das etapas, mais um índice (GET /demands/{id}/package.zip).
func (h *Handlers) DownloadPackage(w http.ResponseWriter, r *http.Request) {
	demandID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, r, h.logger, apperrors.BadRequest("id da demanda inválido"))
		return
	}

	// Carrega e valida ANTES de escrever qualquer cabeçalho, para poder
	// responder 400/404 em JSON em vez de um zip quebrado.
	demand, err := h.service.LoadDemandForPackage(r.Context(), demandID)
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}

	fileName := application.PackageFileName(demand)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, fileName))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if err := h.service.WriteDemandPackage(r.Context(), demand, w); err != nil {
		// Cabeçalhos (e talvez parte do corpo) já foram enviados — não dá
		// para trocar por um 500 JSON. Só registra; o cliente recebe um
		// zip truncado e tenta de novo.
		h.logger.Error("falha ao gerar pacote da demanda",
			slog.String("demanda_id", demandID.String()), slog.Any("error", err))
	}
}

// Dashboard devolve o painel de Demandas Mensais (GET /demands/dashboard).
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Dashboard(r.Context(), time.Now())
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	httputil.WriteOK(w, data)
}

// GetHistory devolve a trilha de auditoria da demanda (GET /demands/{id}/history).
func (h *Handlers) GetHistory(w http.ResponseWriter, r *http.Request) {
	demandID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, r, h.logger, apperrors.BadRequest("id da demanda inválido"))
		return
	}
	rows, err := h.service.DemandHistory(r.Context(), demandID)
	if err != nil {
		httputil.WriteError(w, r, h.logger, err)
		return
	}
	httputil.WriteOK(w, rows)
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
