package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Rambudhi/money-transfer/internal/domain/entities"
	"github.com/Rambudhi/money-transfer/internal/domain/repositories"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

type TransferPublisher interface {
	PublishTransfer(
		ctx context.Context,
		transaction *entities.Transaction,
	) error
}

type OutboxPublisher struct {
	db                *pgxpool.Pool
	outboxRepo        repositories.MessageOutboxRepository
	transferPublisher TransferPublisher
}

func NewOutboxPublisher(
	db *pgxpool.Pool,
	outboxRepo repositories.MessageOutboxRepository,
	transferPublisher TransferPublisher,
) *OutboxPublisher {
	return &OutboxPublisher{
		db:                db,
		outboxRepo:        outboxRepo,
		transferPublisher: transferPublisher,
	}
}

func RegisterOutboxPublisher(lifecycle fx.Lifecycle, publisher *OutboxPublisher) {
	var cancel context.CancelFunc
	done := make(chan struct{})

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			runCtx, runCancel := context.WithCancel(context.Background())
			cancel = runCancel

			go func() {
				defer close(done)
				publisher.Run(runCtx)
			}()

			return nil
		},

		OnStop: func(ctx context.Context) error {
			if cancel != nil {
				cancel()
			}

			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
}

func (p *OutboxPublisher) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	slog.Info("Outbox publisher started")

	p.processPending(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Outbox publisher stopped")
			return

		case <-ticker.C:
			p.processPending(ctx)
		}
	}
}

func (p *OutboxPublisher) processPending(ctx context.Context) {
	const batchSize = 100

	messages, err := p.outboxRepo.GetPending(ctx, batchSize)
	if err != nil {
		slog.Error("Failed to get pending outbox messages", "error", err)
		return
	}

	for _, message := range messages {
		if ctx.Err() != nil {
			return
		}

		if err := p.publishMessage(ctx, message); err != nil {
			slog.Error("Failed to publish outbox message", "outbox_id", message.ID, "aggregate_id", message.AggregateID, "retry_count", message.RetryCount, "error", err)
		}
	}
}

func (p *OutboxPublisher) publishMessage(ctx context.Context, message entities.MessageOutbox) error {
	if message.EventType != "TRANSFER_CREATED" {
		return fmt.Errorf("unsupported outbox event type: %s", message.EventType)
	}

	transaction := &entities.Transaction{
		ID: message.AggregateID,
	}

	if err := p.transferPublisher.PublishTransfer(ctx, transaction); err != nil {
		if retryErr := p.incrementRetry(ctx, message.ID); retryErr != nil {
			return fmt.Errorf(
				"publish failed: %w; increment retry failed: %v",
				err,
				retryErr,
			)
		}

		return fmt.Errorf("publish transfer: %w", err)
	}

	if err := p.markPublished(ctx, message.ID); err != nil {
		return fmt.Errorf("mark outbox published: %w", err)
	}

	slog.Info(
		"Outbox message published",
		"outbox_id", message.ID,
		"transaction_id", message.AggregateID,
	)

	return nil
}

func (p *OutboxPublisher) markPublished(ctx context.Context, id int64) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin mark published transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	if err := p.outboxRepo.MarkPublished(ctx, tx, id); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit mark published: %w", err)
	}

	return nil
}

func (p *OutboxPublisher) incrementRetry(ctx context.Context, id int64) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin increment retry transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	if err := p.outboxRepo.IncrementRetry(ctx, tx, id); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit increment retry: %w", err)
	}

	return nil
}
