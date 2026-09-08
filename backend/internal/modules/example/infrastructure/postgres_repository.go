package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yurythx/projeto-aurora/internal/modules/example/domain"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateTx grava item em tx (a transação de negócio de quem chama, nunca
// uma própria — ver domain.Repository.CreateTx) para que o INSERT seja
// atômico com o outbox.Write do evento "example.item.created" que
// application.Service.CreateItem grava na MESMA transação.
func (r *PostgresRepository) CreateTx(ctx context.Context, tx pgx.Tx, item *domain.Item) error {
	query := `
		INSERT INTO example_items (id, title, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := tx.Exec(ctx, query, item.ID, item.Title, item.Description, item.Status, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres.CreateTx: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	query := `
		SELECT id, title, description, status, created_at, updated_at
		FROM example_items
		WHERE id = $1
	`
	item := &domain.Item{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID, &item.Title, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrExampleNotFound
		}
		return nil, fmt.Errorf("postgres.GetByID: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]*domain.Item, int, error) {
	countQuery := `SELECT COUNT(*) FROM example_items`
	var total int
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres.List count: %w", err)
	}

	query := `
		SELECT id, title, description, status, created_at, updated_at
		FROM example_items
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.List query: %w", err)
	}
	defer rows.Close()

	var items []*domain.Item
	for rows.Next() {
		item := &domain.Item{}
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("postgres.List scan: %w", err)
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, item *domain.Item) error {
	query := `
		UPDATE example_items
		SET title = $1, description = $2, status = $3, updated_at = $4
		WHERE id = $5
	`
	res, err := r.db.Exec(ctx, query, item.Title, item.Description, item.Status, item.UpdatedAt, item.ID)
	if err != nil {
		return fmt.Errorf("postgres.Update: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrExampleNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM example_items WHERE id = $1`
	res, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("postgres.Delete: %w", err)
	}
	if res.RowsAffected() == 0 {
		return domain.ErrExampleNotFound
	}
	return nil
}
