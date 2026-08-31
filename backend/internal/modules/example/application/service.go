package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/yurythx/projeto-aurora/internal/modules/example/domain"
	"github.com/yurythx/projeto-aurora/internal/platform/audit"
	"github.com/yurythx/projeto-aurora/internal/platform/outbox"
)

// Service gerencia as regras de negócio para o módulo de exemplo.
type Service struct {
	repo        domain.Repository
	outbox      *outbox.Writer
	auditWriter *audit.Writer
	logger      *slog.Logger
}

func NewService(repo domain.Repository, outbox *outbox.Writer, auditWriter *audit.Writer, logger *slog.Logger) *Service {
	return &Service{
		repo:        repo,
		outbox:      outbox,
		auditWriter: auditWriter,
		logger:      logger,
	}
}

func (s *Service) CreateItem(ctx context.Context, title, description string) (*domain.Item, error) {
	item, err := domain.NewItem(title, description)
	if err != nil {
		return nil, fmt.Errorf("example.CreateItem: %w", err)
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("example.CreateItem repo: %w", err)
	}

	s.logger.Info("item de exemplo criado com sucesso", slog.String("id", item.ID.String()))
	return item, nil
}

func (s *Service) GetItem(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListItems(ctx context.Context, page, pageSize int) ([]*domain.Item, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, pageSize, offset)
}
