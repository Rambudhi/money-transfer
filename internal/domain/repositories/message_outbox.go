package repositories

import (
	"context"
	"fmt"

	"github.com/Rambudhi/money-transfer/internal/domain/entities"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageOutboxRepository interface {
	Create(ctx context.Context, tx pgx.Tx, message *entities.MessageOutbox) error
	GetPending(ctx context.Context, limit int) ([]entities.MessageOutbox, error)
	MarkPublished(ctx context.Context, tx pgx.Tx, id int64) error
	IncrementRetry(ctx context.Context, tx pgx.Tx, id int64) error
}

type messageOutboxRepository struct {
	DB *pgxpool.Pool
}

func NewMessageOutboxRepository(db *pgxpool.Pool) MessageOutboxRepository {
	return &messageOutboxRepository{
		DB: db,
	}
}

func (r *messageOutboxRepository) Create(ctx context.Context, tx pgx.Tx, message *entities.MessageOutbox) error {
	query := `
        INSERT INTO message_outbox (
            aggregate_id,
            event_type,
            payload,
            status
        )
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at, updated_at
    `

	return tx.QueryRow(
		ctx,
		query,
		message.AggregateID,
		message.EventType,
		message.Payload,
		message.Status,
	).Scan(
		&message.ID,
		&message.CreatedAt,
		&message.UpdatedAt,
	)
}

func (r *messageOutboxRepository) GetPending(ctx context.Context, limit int) ([]entities.MessageOutbox, error) {
	query := `
        SELECT
            id,
            aggregate_id,
            event_type,
            payload,
            status,
            retry_count,
            published_at,
            created_at,
            updated_at
        FROM message_outbox
        WHERE status = 'PENDING'
        ORDER BY created_at ASC, id ASC
        LIMIT $1
    `

	rows, err := r.DB.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]entities.MessageOutbox, 0)

	for rows.Next() {
		var message entities.MessageOutbox

		err := rows.Scan(
			&message.ID,
			&message.AggregateID,
			&message.EventType,
			&message.Payload,
			&message.Status,
			&message.RetryCount,
			&message.PublishedAt,
			&message.CreatedAt,
			&message.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *messageOutboxRepository) MarkPublished(ctx context.Context, tx pgx.Tx, id int64) error {
	query := `
        UPDATE message_outbox
        SET
            status = 'PUBLISHED',
            published_at = NOW(),
            updated_at = NOW()
        WHERE id = $1
          AND status = 'PENDING'
    `

	result, err := tx.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("outbox message %d not found or already published", id)
	}

	return nil
}

func (r *messageOutboxRepository) IncrementRetry(ctx context.Context, tx pgx.Tx, id int64) error {
	query := `
        UPDATE message_outbox
        SET
            retry_count = retry_count + 1,
            updated_at = NOW()
        WHERE id = $1
          AND status = 'PENDING'
    `

	result, err := tx.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("outbox message %d not found or already published", id)
	}

	return nil
}
