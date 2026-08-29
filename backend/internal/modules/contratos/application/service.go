// Package application implementa os casos de uso do módulo de contratos
// municipais. Depende só de domain — nunca de infrastructure diretamente.
package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/modules/contratos/domain"
)

// Service implementa os casos de uso de gestão de contratos municipais.
type Service struct {
	repo   domain.Repository
	logger *slog.Logger
	// Casador automático Contrato <-> Diário Oficial (opcional; ligado por
	// WithDiarioMatching a partir de internal/app).
	diario DiarioMatchSource
	events ContratoEventEmitter
}

// NewService constrói um Service com as dependências injetadas.
func NewService(repo domain.Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// CreateInput são os dados necessários para criar um novo contrato.
type CreateInput struct {
	Numero             string
	Objeto             string
	Contratante        string
	Contratado         string
	CNPJ               string
	Valor              *float64
	DataAssinatura     *time.Time
	DataVigenciaInicio *time.Time
	DataVigenciaFim    *time.Time
	CreatedBy          *uuid.UUID
}

// Create valida e persiste um novo contrato com status inicial "rascunho".
func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Contrato, error) {
	if in.Numero == "" {
		return domain.Contrato{}, apperrors.Validation("numero é obrigatório")
	}
	if in.Objeto == "" {
		return domain.Contrato{}, apperrors.Validation("objeto é obrigatório")
	}
	if in.Contratante == "" {
		return domain.Contrato{}, apperrors.Validation("contratante é obrigatório")
	}
	if in.Contratado == "" {
		return domain.Contrato{}, apperrors.Validation("contratado é obrigatório")
	}

	c := domain.Contrato{
		ID:                 uuid.New(),
		Numero:             in.Numero,
		Objeto:             in.Objeto,
		Contratante:        in.Contratante,
		Contratado:         in.Contratado,
		CNPJ:               in.CNPJ,
		Valor:              in.Valor,
		DataAssinatura:     in.DataAssinatura,
		DataVigenciaInicio: in.DataVigenciaInicio,
		DataVigenciaFim:    in.DataVigenciaFim,
		Status:             domain.StatusRascunho,
		CreatedBy:          in.CreatedBy,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.repo.Create(ctx, c); err != nil {
		return domain.Contrato{}, fmt.Errorf("contratos: create: %w", err)
	}

	s.logger.Info("contrato criado",
		slog.String("id", c.ID.String()),
		slog.String("numero", c.Numero),
	)
	return c, nil
}

// GetByID retorna um contrato com seus detalhes (refs e aditivos).
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (domain.Contrato, error) {
	c, err := s.repo.GetWithDetails(ctx, id)
	if err != nil {
		return domain.Contrato{}, fmt.Errorf("contratos: get: %w", err)
	}
	return c, nil
}

// GetDashboardStats retorna os indicadores da tela inicial.
func (s *Service) GetDashboardStats(ctx context.Context) (domain.DashboardStats, error) {
	return s.repo.GetDashboardStats(ctx)
}

// List retorna uma lista paginada de contratos com filtros opcionais.
func (s *Service) List(ctx context.Context, params domain.ListParams) ([]domain.Contrato, int64, error) {
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	contratos, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("contratos: list: %w", err)
	}
	return contratos, total, nil
}

// UpdateStatus aplica uma transição de status, validando se ela é permitida.
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus domain.Status) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("contratos: update status get: %w", err)
	}

	if !c.Status.CanTransitionTo(newStatus) {
		return apperrors.Validation(fmt.Sprintf(
			"transição de status '%s' → '%s' não é permitida",
			c.Status, newStatus,
		))
	}

	if err := s.repo.UpdateStatus(ctx, id, newStatus); err != nil {
		return fmt.Errorf("contratos: update status: %w", err)
	}

	s.logger.Info("status de contrato atualizado",
		slog.String("id", id.String()),
		slog.String("de", string(c.Status)),
		slog.String("para", string(newStatus)),
	)
	return nil
}

// KanbanView retorna os contratos agrupados por status para a visão Kanban.
func (s *Service) KanbanView(ctx context.Context, limitPerCol int) (map[domain.Status][]domain.Contrato, error) {
	if limitPerCol <= 0 {
		limitPerCol = 20
	}
	result, err := s.repo.KanbanCols(ctx, limitPerCol)
	if err != nil {
		return nil, fmt.Errorf("contratos: kanban: %w", err)
	}
	return result, nil
}

// AddDiarioRef vincula uma publicação do Diário Oficial a um contrato.
// Chamado pelo worker de integração e também pode ser chamado manualmente
// via API.
func (s *Service) AddDiarioRef(ctx context.Context, ref domain.DiarioRef) error {
	if _, err := s.repo.GetByID(ctx, ref.ContratoID); err != nil {
		return fmt.Errorf("contratos: add diario ref: %w", err)
	}
	if err := s.repo.AddDiarioRef(ctx, ref); err != nil {
		return fmt.Errorf("contratos: add diario ref: %w", err)
	}
	return nil
}

// CreateFromDiario cria ou atualiza um contrato a partir de uma publicação
// do Diário Oficial. Usado pelo worker de integração.
// Se já existir um contrato com o mesmo número, vincula a publicação.
// Caso contrário, cria um novo contrato com status "vigente".
func (s *Service) CreateFromDiario(ctx context.Context, in CreateFromDiarioInput) error {
	// Tenta encontrar contrato existente pelo número
	existing, err := s.repo.FindByNumero(ctx, in.Numero)
	if err == nil {
		// Contrato já existe — só vincula a publicação
		ref := domain.DiarioRef{
			ContratoID:    existing.ID,
			EditionNumber: in.EditionNumber,
			TipoEvento:    in.TipoEvento,
			PublicadoEm:   in.PublicadoEm,
			Contexto:      in.Contexto,
			DocURL:        in.DocURL,
		}
		return s.repo.AddDiarioRef(ctx, ref)
	}

	// Contrato não existe — cria novo com status vigente
	now := time.Now()
	c := domain.Contrato{
		ID:          uuid.New(),
		Numero:      in.Numero,
		Objeto:      in.Objeto,
		Contratante: in.Contratante,
		Contratado:  in.Contratado,
		CNPJ:        in.CNPJ,
		Valor:       in.Valor,
		Status:      domain.StatusVigente,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if in.DataPublicacao != nil {
		c.DataAssinatura = in.DataPublicacao
	}

	if err := s.repo.Create(ctx, c); err != nil {
		return fmt.Errorf("contratos: create from diario: %w", err)
	}

	ref := domain.DiarioRef{
		ContratoID:    c.ID,
		EditionNumber: in.EditionNumber,
		TipoEvento:    in.TipoEvento,
		PublicadoEm:   in.PublicadoEm,
		Contexto:      in.Contexto,
		DocURL:        in.DocURL,
	}
	return s.repo.AddDiarioRef(ctx, ref)
}

// CreateFromDiarioInput são os dados extraídos de uma publicação do
// Diário Oficial para criar/atualizar um contrato.
type CreateFromDiarioInput struct {
	Numero         string
	Objeto         string
	Contratante    string
	Contratado     string
	CNPJ           string
	Valor          *float64
	TipoEvento     string
	EditionNumber  string
	DataPublicacao *time.Time
	PublicadoEm    *time.Time
	Contexto       string
	DocURL         string
}
