package domain

import (
	"time"

	"github.com/google/uuid"
)

// DemandEtapaChangedEvent ocorre quando uma demanda avança (ou retrocede) no Kanban.
type DemandEtapaChangedEvent struct {
	DemandID   uuid.UUID   `json:"demand_id"`
	ContratoID uuid.UUID   `json:"contrato_id"`
	OldEtapa   EtapaKanban `json:"old_etapa"`
	NewEtapa   EtapaKanban `json:"new_etapa"`
	OccurredAt time.Time   `json:"occurred_at"`
}

// EventType retorna o tipo fixo do evento para o envelope do outbox.
func (e DemandEtapaChangedEvent) EventType() string {
	return "demand.etapa_changed"
}

// AggregateID retorna o ID da raiz de agregação que gerou o evento.
func (e DemandEtapaChangedEvent) AggregateID() string {
	return e.DemandID.String()
}

// AggregateType retorna o tipo da raiz de agregação.
func (e DemandEtapaChangedEvent) AggregateType() string {
	return "demand"
}
