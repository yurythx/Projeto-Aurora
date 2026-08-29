// Package domain define as entidades e contratos do módulo de contratos
// municipais. Segue o mesmo padrão de domain-driven design dos demais
// módulos da plataforma: entidades puras sem dependência de infraestrutura,
// interfaces de repositório como pontos de extensão.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// Status representa o estado atual de um contrato municipal no fluxo
// de tramitação. A progressão normal é:
//
//	Rascunho → EmAnalise → Aprovado → Vigente → Encerrado
//
// Cancelado pode ocorrer a partir de qualquer estado exceto Encerrado.
type Status string

const (
	StatusRascunho  Status = "rascunho"
	StatusEmAnalise Status = "em_analise"
	StatusAprovado  Status = "aprovado"
	StatusVigente   Status = "vigente"
	StatusEncerrado Status = "encerrado"
	StatusCancelado Status = "cancelado"
)

// AllStatuses retorna todos os status na ordem do Kanban.
func AllStatuses() []Status {
	return []Status{
		StatusRascunho,
		StatusEmAnalise,
		StatusAprovado,
		StatusVigente,
		StatusEncerrado,
		StatusCancelado,
	}
}

// KanbanLabel retorna o rótulo de exibição de cada coluna do Kanban.
func (s Status) KanbanLabel() string {
	switch s {
	case StatusRascunho:
		return "Rascunho"
	case StatusEmAnalise:
		return "Em Análise"
	case StatusAprovado:
		return "Aprovado"
	case StatusVigente:
		return "Vigente"
	case StatusEncerrado:
		return "Encerrado"
	case StatusCancelado:
		return "Cancelado"
	default:
		return string(s)
	}
}

// ValidTransitions define quais transições de status são permitidas.
// A chave é o status ATUAL; o valor é a lista de status para os quais
// é possível avançar. Encerrado e Cancelado são estados terminais.
var ValidTransitions = map[Status][]Status{
	StatusRascunho:  {StatusEmAnalise, StatusCancelado},
	StatusEmAnalise: {StatusAprovado, StatusRascunho, StatusCancelado},
	StatusAprovado:  {StatusVigente, StatusEmAnalise, StatusCancelado},
	StatusVigente:   {StatusEncerrado, StatusCancelado},
	StatusEncerrado: {},
	StatusCancelado: {},
}

// CanTransitionTo reporta se a transição de status atual → next é permitida.
func (s Status) CanTransitionTo(next Status) bool {
	for _, allowed := range ValidTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

// Contrato é a entidade principal do módulo. Representa um contrato
// administrativo municipal, com seu ciclo de vida de tramitação.
type Contrato struct {
	ID                 uuid.UUID
	Numero             string
	Objeto             string
	Contratante        string // secretaria/órgão municipal contratante
	Contratado         string // empresa/fornecedor contratado
	CNPJ               string
	Valor              *float64 // nil quando não informado
	DataAssinatura     *time.Time
	DataVigenciaInicio *time.Time
	DataVigenciaFim    *time.Time
	Status             Status
	// DiarioRefs lista as publicações do Diário Oficial vinculadas.
	// Preenchido somente quando carregado com WithRefs=true.
	DiarioRefs []DiarioRef
	// Aditivos lista os aditivos vinculados.
	// Preenchido somente quando carregado com WithAditivos=true.
	Aditivos  []Aditivo
	CreatedBy *uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DiarioRef representa uma publicação do Diário Oficial vinculada
// a um contrato — seja a publicação original de criação, um aditivo,
// uma rescisão ou um extrato de fiscalização.
type DiarioRef struct {
	ContratoID    uuid.UUID
	EditionNumber string
	TipoEvento    string // ex.: CONTRATO, ADITIVO, RESCISAO, EXTRATO
	PublicadoEm   *time.Time
	Contexto      string // trecho relevante do texto da publicação
	DocURL        string
}

// Aditivo representa um termo aditivo de um contrato.
type Aditivo struct {
	ID             uuid.UUID
	ContratoID     uuid.UUID
	Numero         string
	Objeto         string
	ValorAdicional *float64
	DataAssinatura *time.Time
	CreatedAt      time.Time
}

// ListParams agrupa os filtros de listagem de contratos.
type ListParams struct {
	Status   Status // vazio = todos
	Busca    string // busca textual em objeto/contratado
	Page     int
	PageSize int
}
