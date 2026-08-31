package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrExampleNotFound = errors.New("example entity not found")
	ErrInvalidInput    = errors.New("invalid example input")
)

// Item representa uma entidade de exemplo genérica no Projeto Aurora.
// Serve como modelo para novos módulos a serem criados no sistema.
type Item struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewItem cria e valida um novo Item de exemplo.
func NewItem(title, description string) (*Item, error) {
	if title == "" {
		return nil, ErrInvalidInput
	}

	now := time.Now().UTC()
	return &Item{
		ID:          uuid.New(),
		Title:       title,
		Description: description,
		Status:      "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}
