package transport

import (
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova/internal/modules/contratos/domain"
)

// --- Request DTOs ---

// CreateContratoRequest é o payload de POST /contratos.
type CreateContratoRequest struct {
	Numero             string   `json:"numero"`
	Objeto             string   `json:"objeto"`
	Contratante        string   `json:"contratante"`
	Contratado         string   `json:"contratado"`
	CNPJ               string   `json:"cnpj,omitempty"`
	Valor              *float64 `json:"valor,omitempty"`
	DataAssinatura     *string  `json:"data_assinatura,omitempty"` // "2006-01-02"
	DataVigenciaInicio *string  `json:"data_vigencia_inicio,omitempty"`
	DataVigenciaFim    *string  `json:"data_vigencia_fim,omitempty"`
}

// UpdateStatusRequest é o payload de PATCH /contratos/{id}/status.
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// AddDiarioRefRequest é o payload de POST /contratos/{id}/diario-refs.
type AddDiarioRefRequest struct {
	EditionNumber string  `json:"edition_number"`
	TipoEvento    string  `json:"tipo_evento,omitempty"`
	PublicadoEm   *string `json:"publicado_em,omitempty"` // "2006-01-02"
	Contexto      string  `json:"contexto,omitempty"`
	DocURL        string  `json:"doc_url,omitempty"`
}

// --- Response DTOs ---

// ContratoResponse é a representação de um contrato na API.
type ContratoResponse struct {
	ID                 uuid.UUID           `json:"id"`
	Numero             string              `json:"numero"`
	Objeto             string              `json:"objeto"`
	Contratante        string              `json:"contratante"`
	Contratado         string              `json:"contratado"`
	CNPJ               string              `json:"cnpj,omitempty"`
	Valor              *float64            `json:"valor,omitempty"`
	DataAssinatura     *string             `json:"data_assinatura,omitempty"`
	DataVigenciaInicio *string             `json:"data_vigencia_inicio,omitempty"`
	DataVigenciaFim    *string             `json:"data_vigencia_fim,omitempty"`
	Status             string              `json:"status"`
	StatusLabel        string              `json:"status_label"`
	DiarioRefs         []DiarioRefResponse `json:"diario_refs,omitempty"`
	Aditivos           []AditivoResponse   `json:"aditivos,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

// DiarioRefResponse é a representação de uma referência do Diário Oficial.
type DiarioRefResponse struct {
	EditionNumber string  `json:"edition_number"`
	TipoEvento    string  `json:"tipo_evento,omitempty"`
	PublicadoEm   *string `json:"publicado_em,omitempty"`
	Contexto      string  `json:"contexto,omitempty"`
	DocURL        string  `json:"doc_url,omitempty"`
}

// AditivoResponse é a representação de um aditivo de contrato.
type AditivoResponse struct {
	ID             uuid.UUID `json:"id"`
	Numero         string    `json:"numero,omitempty"`
	Objeto         string    `json:"objeto,omitempty"`
	ValorAdicional *float64  `json:"valor_adicional,omitempty"`
	DataAssinatura *string   `json:"data_assinatura,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// KanbanColumnResponse agrupa contratos de uma coluna do Kanban.
type KanbanColumnResponse struct {
	Status string             `json:"status"`
	Label  string             `json:"label"`
	Total  int                `json:"total"`
	Items  []ContratoResponse `json:"items"`
}

// KanbanResponse é a resposta completa da visão Kanban.
type KanbanResponse struct {
	Columns []KanbanColumnResponse `json:"columns"`
}

// ListContratosResponse é a resposta paginada de listagem.
type ListContratosResponse struct {
	Data  []ContratoResponse `json:"data"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Pages int                `json:"pages"`
}

// --- Conversão domain → DTO ---

const dateLayout = "2006-01-02"

func toContratoResponse(c domain.Contrato) ContratoResponse {
	resp := ContratoResponse{
		ID:          c.ID,
		Numero:      c.Numero,
		Objeto:      c.Objeto,
		Contratante: c.Contratante,
		Contratado:  c.Contratado,
		CNPJ:        c.CNPJ,
		Valor:       c.Valor,
		Status:      string(c.Status),
		StatusLabel: c.Status.KanbanLabel(),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
	if c.DataAssinatura != nil {
		s := c.DataAssinatura.Format(dateLayout)
		resp.DataAssinatura = &s
	}
	if c.DataVigenciaInicio != nil {
		s := c.DataVigenciaInicio.Format(dateLayout)
		resp.DataVigenciaInicio = &s
	}
	if c.DataVigenciaFim != nil {
		s := c.DataVigenciaFim.Format(dateLayout)
		resp.DataVigenciaFim = &s
	}

	for _, ref := range c.DiarioRefs {
		resp.DiarioRefs = append(resp.DiarioRefs, toDiarioRefResponse(ref))
	}
	for _, a := range c.Aditivos {
		resp.Aditivos = append(resp.Aditivos, toAditivoResponse(a))
	}
	return resp
}

func toDiarioRefResponse(ref domain.DiarioRef) DiarioRefResponse {
	r := DiarioRefResponse{
		EditionNumber: ref.EditionNumber,
		TipoEvento:    ref.TipoEvento,
		Contexto:      ref.Contexto,
		DocURL:        ref.DocURL,
	}
	if ref.PublicadoEm != nil {
		s := ref.PublicadoEm.Format(dateLayout)
		r.PublicadoEm = &s
	}
	return r
}

func toAditivoResponse(a domain.Aditivo) AditivoResponse {
	r := AditivoResponse{
		ID:             a.ID,
		Numero:         a.Numero,
		Objeto:         a.Objeto,
		ValorAdicional: a.ValorAdicional,
		CreatedAt:      a.CreatedAt,
	}
	if a.DataAssinatura != nil {
		s := a.DataAssinatura.Format(dateLayout)
		r.DataAssinatura = &s
	}
	return r
}

func toKanbanResponse(cols map[domain.Status][]domain.Contrato) KanbanResponse {
	resp := KanbanResponse{}
	for _, s := range domain.AllStatuses() {
		items := cols[s]
		col := KanbanColumnResponse{
			Status: string(s),
			Label:  s.KanbanLabel(),
			Total:  len(items),
		}
		for _, c := range items {
			col.Items = append(col.Items, toContratoResponse(c))
		}
		resp.Columns = append(resp.Columns, col)
	}
	return resp
}

// parseDate parseia uma data opcional no formato "2006-01-02".
func parseDate(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(dateLayout, *s)
	if err != nil {
		return nil
	}
	return &t
}
