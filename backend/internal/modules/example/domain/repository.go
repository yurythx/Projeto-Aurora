package domain

import (
	"context"

	"github.com/google/uuid"
)

// Repository define a interface de persistência para o módulo de exemplo.
type Repository interface {
	Create(ctx context.Context, item *Item) error
	GetByID(ctx context.Context, id uuid.UUID) (*Item, error)
	List(ctx context.Context, limit, offset int) ([]*Item, int, error)
	Update(ctx context.Context, item *Item) error
	Delete(ctx context.Context, id uuid.UUID) error
}
