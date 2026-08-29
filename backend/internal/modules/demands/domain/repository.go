package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository define a interface de persistência para as demandas mensais e documentos.
type Repository interface {
	// Create inicia uma nova demanda mensal para um contrato.
	Create(ctx context.Context, demand MonthlyDemand) error

	// GetByID busca uma demanda com todos os seus documentos anexados.
	GetByID(ctx context.Context, id uuid.UUID) (MonthlyDemand, error)

	// Update atualiza os dados da demanda (etapa, status).
	Update(ctx context.Context, demand MonthlyDemand) error

	// ListByContract retorna todas as demandas de um contrato específico.
	ListByContract(ctx context.Context, contratoID uuid.UUID) ([]MonthlyDemand, error)

	// ListAllKanban retorna todas as demandas ativas em todas as etapas.
	ListAllKanban(ctx context.Context) (map[EtapaKanban][]MonthlyDemand, error)

	// AddDocument anexa um novo documento exigido (compliance).
	AddDocument(ctx context.Context, doc DemandDocument) error

	// RemoveDocument remove um documento anexado caso haja erro de upload.
	RemoveDocument(ctx context.Context, docID uuid.UUID) error

	// ExistsForContractMonth reporta se já existe uma demanda para o contrato
	// no mês dado — idempotência da geração automática mensal.
	ExistsForContractMonth(ctx context.Context, contratoID uuid.UUID, anoMes string) (bool, error)

	// ListStale devolve as demandas cuja etapa começou antes de `olderThan`
	// (candidatas a pendência de SLA). Traz contrato_id e etapa_started_at.
	ListStale(ctx context.Context, olderThan time.Time) ([]MonthlyDemand, error)

	// EnsureSLAOccurrence grava uma pendência de SLA se ainda não houver uma
	// aberta (resolvida=false) do mesmo tipo para a demanda. Retorna se criou.
	EnsureSLAOccurrence(ctx context.Context, o Occurrence) (bool, error)

	// ListOpenOccurrences lista as pendências não resolvidas (dashboard/alertas).
	ListOpenOccurrences(ctx context.Context, limit int) ([]Occurrence, error)

	// CountByEtapa devolve quantas demandas há em cada etapa (funil do
	// dashboard). Etapas sem demanda podem vir ausentes do mapa.
	CountByEtapa(ctx context.Context) (map[EtapaKanban]int, error)

	// ListExpiringCertidoes lista, por demanda ainda em andamento (etapa < 6),
	// a certidão de compliance mais recente de cada tipo cuja validade é
	// anterior a `before` — as vencidas e as a vencer, da mais crítica para a
	// menos. Traz o número do contrato para exibição.
	ListExpiringCertidoes(ctx context.Context, before time.Time, limit int) ([]ExpiringCertidao, error)
}

// ExpiringCertidao é uma linha do painel "certidões a vencer" do dashboard.
type ExpiringCertidao struct {
	DemandaID      uuid.UUID
	ContratoNumero string
	DocType        DocumentType
	ValidadeAte    time.Time
	Etapa          EtapaKanban
}
