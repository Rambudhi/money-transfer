package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rambudhi/money-transfer/internal/domain/entities"
	"github.com/jackc/pgx/v5"
)

type TransactionRepository interface {
	Create(ctx context.Context, tx pgx.Tx, transaction *entities.Transaction) error
	GetByID(ctx context.Context, tx pgx.Tx, transactionID string) (*entities.Transaction, error)
	MarkSuccess(ctx context.Context, tx pgx.Tx, transactionID string) error
	MarkFailed(ctx context.Context, tx pgx.Tx, transactionID string, reason string) (*entities.Transaction, error)
}

type transactionRepository struct{}

func NewTransactionRepository() TransactionRepository {
	return &transactionRepository{}
}

func (r *transactionRepository) Create(ctx context.Context, tx pgx.Tx, transaction *entities.Transaction) error {

	query := `
		INSERT INTO transactions (
			id,
			reference_id,
			merchant_id,
			amount,
			destination_account,
			status,
			created_at,
			updated_at
		)
		VALUES (
			gen_random_uuid(),
			gen_random_uuid()::TEXT,
			$1,
			$2,
			$3,
			$4,
			NOW(),
			NOW()
		)
		RETURNING
			id::TEXT,
			reference_id,
			created_at,
			updated_at
	`

	err := tx.QueryRow(
		ctx,
		query,
		transaction.MerchantID,
		transaction.Amount,
		transaction.DestinationAccount,
		entities.TransactionStatusPending,
	).Scan(
		&transaction.ID,
		&transaction.ReferenceID,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	transaction.Status = entities.TransactionStatusPending

	return nil
}

func (r *transactionRepository) GetByID(ctx context.Context, tx pgx.Tx, transactionID string) (*entities.Transaction, error) {

	query := `
		SELECT
			id::TEXT,
			reference_id,
			merchant_id,
			amount,
			destination_account,
			status,
			COALESCE(failure_reason, ''),
			created_at,
			updated_at
		FROM transactions
		WHERE id = $1
	`

	var transaction entities.Transaction

	err := tx.QueryRow(ctx, query, transactionID).Scan(
		&transaction.ID,
		&transaction.ReferenceID,
		&transaction.MerchantID,
		&transaction.Amount,
		&transaction.DestinationAccount,
		&transaction.Status,
		&transaction.FailureReason,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("transaction not found: %w", err)
		}

		return nil, fmt.Errorf("get transaction by id: %w", err)
	}

	return &transaction, nil
}

func (r *transactionRepository) MarkSuccess(ctx context.Context, tx pgx.Tx, transactionID string) error {

	query := `
		UPDATE transactions
		SET
			status = $1,
			updated_at = NOW()
		WHERE id = $2
		  AND status = $3
	`

	result, err := tx.Exec(
		ctx,
		query,
		entities.TransactionStatusSuccess,
		transactionID,
		entities.TransactionStatusPending,
	)

	if err != nil {
		return fmt.Errorf("mark transaction success: %w", err)
	}

	if result.RowsAffected() == 0 {
		return r.checkTransactionStatus(ctx, tx, transactionID)
	}

	return nil
}

func (r *transactionRepository) MarkFailed(ctx context.Context, tx pgx.Tx, transactionID string, reason string) (*entities.Transaction, error) {

	query := `
		UPDATE transactions
		SET
			status = $1,
			failure_reason = $2,
			updated_at = NOW()
		WHERE id = $3
		  AND status = $4
		RETURNING
			id::TEXT,
			reference_id,
			merchant_id,
			amount,
			destination_account,
			status,
			COALESCE(failure_reason, ''),
			created_at,
			updated_at
	`

	var transaction entities.Transaction

	err := tx.QueryRow(
		ctx,
		query,
		entities.TransactionStatusFailed,
		reason,
		transactionID,
		entities.TransactionStatusPending,
	).Scan(
		&transaction.ID,
		&transaction.ReferenceID,
		&transaction.MerchantID,
		&transaction.Amount,
		&transaction.DestinationAccount,
		&transaction.Status,
		&transaction.FailureReason,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, r.checkTransactionStatus(ctx, tx, transactionID)
		}

		return nil, fmt.Errorf("mark transaction failed: %w", err)
	}

	return &transaction, nil
}

func (r *transactionRepository) checkTransactionStatus(ctx context.Context, tx pgx.Tx, transactionID string) error {

	var status entities.TransactionStatus

	err := tx.QueryRow(ctx, `SELECT status FROM transactions WHERE id = $1`, transactionID).Scan(&status)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("transaction not found: %w", err)
		}

		return fmt.Errorf("check transaction status: %w", err)
	}

	return fmt.Errorf("transaction already processed: %w", err)
}
