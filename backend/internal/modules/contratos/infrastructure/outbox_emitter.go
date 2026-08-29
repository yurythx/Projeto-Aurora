package infrastructure

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova/internal/platform/database"
	"github.com/yurythx/projeto-nova/internal/platform/outbox"
)

// OutboxEmitter implementa application.ContratoEventEmitter: grava o evento
// em outbox_events dentro de uma transação própria (o casador roda num loop
// de worker, fora de qualquer transação de negócio). O Publisher do outbox
// entrega ao RabbitMQ e daí ao Hub de WebSocket do frontend.
type OutboxEmitter struct {
	db     *pgxpool.Pool
	writer *outbox.Writer
}

func NewOutboxEmitter(db *pgxpool.Pool, writer *outbox.Writer) *OutboxEmitter {
	return &OutboxEmitter{db: db, writer: writer}
}

func (e *OutboxEmitter) EmitContratoEvent(ctx context.Context, eventType, aggregateID string, payload any) error {
	if e.db == nil || e.writer == nil {
		return nil
	}
	correlationID := uuid.New()
	return database.WithTx(ctx, e.db, func(ctx context.Context, tx pgx.Tx) error {
		if err := e.writer.Write(ctx, tx, eventType, "contrato", aggregateID, correlationID, payload); err != nil {
			return fmt.Errorf("contratos: emit %s: %w", eventType, err)
		}
		return nil
	})
}
