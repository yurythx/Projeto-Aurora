package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Repository define a interface de persistência para o módulo de exemplo.
type Repository interface {
	// CreateTx grava item na transação de negócio de quem chama — nunca
	// abre a sua própria — para que o INSERT e o outbox.Write do evento
	// "example.item.created" (ver application.Service.CreateItem) sejam
	// atômicos: os dois commitam juntos, ou os dois revertem juntos. Este
	// é o blueprint de referência do Transactional Outbox (§16) que todo
	// módulo novo deveria seguir — os módulos anteriores a este só
	// mantinham a INTERFACE pronta, sem nenhum caso de uso de fato
	// chamando outbox.Write (achado de auditoria).
	CreateTx(ctx context.Context, tx pgx.Tx, item *Item) error
	GetByID(ctx context.Context, id uuid.UUID) (*Item, error)
	List(ctx context.Context, limit, offset int) ([]*Item, int, error)
	Update(ctx context.Context, item *Item) error
	Delete(ctx context.Context, id uuid.UUID) error
}
