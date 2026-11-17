package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/Rambudhi/money-transfer/internal/domain/entities"
	"github.com/jackc/pgx/v5"
)

type MerchantRepository interface {
	GetByID(ctx context.Context, tx pgx.Tx, id int64) (*entities.Merchant, error)
	Debit(ctx context.Context, tx pgx.Tx, id int64, amount int64) error
	Credit(ctx context.Context, tx pgx.Tx, id int64, amount int64) error
}

type merchantRepository struct {
}

func NewMerchantRepository() MerchantRepository {
	return &merchantRepository{}
}

// GetByID mengambil data merchant berdasarkan ID.
func (r *merchantRepository) GetByID(ctx context.Context, tx pgx.Tx, merchantID int64) (*entities.Merchant, error) {
	query := `
		SELECT
			id,
			name,
			balance,
			created_at,
			updated_at
		FROM merchants
		WHERE id = $1
		FOR UPDATE
	`

	var merchant entities.Merchant

	err := tx.QueryRow(ctx, query, merchantID).Scan(
		&merchant.ID,
		&merchant.Name,
		&merchant.Balance,
		&merchant.CreatedAt,
		&merchant.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("merchant not found: %w", err)
		}

		return nil, fmt.Errorf("get merchant by id: %w", err)
	}

	return &merchant, nil
}

func (r *merchantRepository) Debit(ctx context.Context, tx pgx.Tx, id int64, amount int64) error {
	query := `
		UPDATE merchants
		SET balance = balance - $1,
		    updated_at = NOW()
		WHERE id = $2
		  AND balance >= $1
		  AND $1 > 0
	`

	result, err := tx.Exec(ctx, query, amount, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("merchant not found or insufficient balance")
	}

	return nil
}

func (r *merchantRepository) Credit(ctx context.Context, tx pgx.Tx, id int64, amount int64) error {
	query := `
		UPDATE merchants
		SET balance = balance + $1,
		    updated_at = NOW()
		WHERE id = $2
		  AND $1 > 0
	`

	result, err := tx.Exec(ctx, query, amount, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("merchant not found or invalid amount")
	}

	return nil
}
