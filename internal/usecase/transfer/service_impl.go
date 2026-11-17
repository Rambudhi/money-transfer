package transfer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Rambudhi/money-transfer/internal/domain/entities"
	repository "github.com/Rambudhi/money-transfer/internal/domain/repositories"
	"github.com/Rambudhi/money-transfer/internal/infrastructure/postgres"
	"github.com/Rambudhi/money-transfer/internal/pkg/log"
)

type serviceImpl struct {
	db              *postgres.SQLDB
	merchantRepo    repository.MerchantRepository
	transactionRepo repository.TransactionRepository
	// publisher       *messagingrabbitmq.Publisher
	provider   Provider
	outboxRepo repository.MessageOutboxRepository
}

var (
	_ Service       = (*serviceImpl)(nil)
	_ WorkerService = (*serviceImpl)(nil)
)

func NewService(
	db *postgres.SQLDB,
	merchantRepo repository.MerchantRepository,
	transactionRepo repository.TransactionRepository,
	// publisher *messagingrabbitmq.Publisher,
	provider Provider,
	outboxRepo repository.MessageOutboxRepository,
) *serviceImpl {
	return &serviceImpl{
		db:              db,
		merchantRepo:    merchantRepo,
		transactionRepo: transactionRepo,
		// publisher:       publisher,
		provider:   provider,
		outboxRepo: outboxRepo,
	}
}

func (s *serviceImpl) Transfer(ctx context.Context, request TransferRequest) (*entities.Transaction, error) {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	merchant, err := s.merchantRepo.GetByID(ctx, tx, request.MerchantID)
	if err != nil {
		log.Error("Failed to get merchant", "error", err)
		return nil, err
	}

	if merchant.Balance < request.Amount {
		return nil, errors.New("insufficient merchant balance")
	}

	if err := s.merchantRepo.Debit(ctx, tx, request.MerchantID, request.Amount); err != nil {
		log.Error("Failed to debit merchant balance", "error", err)
		return nil, err
	}

	transaction := &entities.Transaction{
		MerchantID:         request.MerchantID,
		Amount:             request.Amount,
		DestinationAccount: request.DestinationAccount,
		Status:             entities.TransactionStatusPending,
	}

	if err := s.transactionRepo.Create(ctx, tx, transaction); err != nil {
		log.Error("Failed to create transaction", "error", err)
		return nil, err
	}

	payload, err := json.Marshal(map[string]string{
		"transaction_id": transaction.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal outbox payload: %w", err)
	}

	outboxMessage := &entities.MessageOutbox{
		AggregateID: transaction.ID,
		EventType:   "TRANSFER_CREATED",
		Payload:     payload,
		Status:      entities.OutboxStatusPending,
	}

	if err := s.outboxRepo.Create(ctx, tx, outboxMessage); err != nil {
		log.Error("Failed to create outbox message", "error", err)
		return nil, fmt.Errorf("create outbox message: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error(
			"Failed to commit transfer transaction",
			"transaction_id", transaction.ID,
			"error", err,
		)
		return nil, fmt.Errorf("commit transfer transaction: %w", err)
	}

	// if err := s.publisher.PublishTransfer(ctx, transaction); err != nil {
	// 	log.Error("Failed to publish transfer message", "transaction_id", transaction.ID, "error", err)
	// 	return transaction, fmt.Errorf("publish transfer message: %w", err)
	// }

	log.Info("Transfer created successfully", "transaction_id", transaction.ID, "merchant_id", request.MerchantID, "amount", request.Amount)
	return transaction, nil
}

func (s *serviceImpl) ProcessTransfer(ctx context.Context, transactionID string) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	transaction, err := s.transactionRepo.GetByID(
		ctx,
		tx,
		transactionID,
	)
	if err != nil {
		_ = tx.Rollback(context.Background())
		return fmt.Errorf("get transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit read transaction: %w", err)
	}

	if transaction.Status != entities.TransactionStatusPending {
		log.Info("Transaction already processed", "transaction_id", transactionID, "status", transaction.Status)
		return nil
	}

	providerErr := s.provider.Transfer(ctx, transaction)

	if errors.Is(providerErr, context.Canceled) ||
		errors.Is(providerErr, context.DeadlineExceeded) {
		return providerErr
	}

	if providerErr != nil {
		return s.refundTransfer(ctx, transactionID, providerErr)
	}

	return s.markTransferSuccess(ctx, transactionID)
}

func (s *serviceImpl) markTransferSuccess(ctx context.Context, transactionID string) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin success transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	if err := s.transactionRepo.MarkSuccess(
		ctx,
		tx,
		transactionID,
	); err != nil {
		return fmt.Errorf("mark transaction success: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit success transaction: %w", err)
	}

	log.Info(
		"Transfer completed successfully",
		"transaction_id", transactionID,
	)

	return nil
}

func (s *serviceImpl) refundTransfer(ctx context.Context, transactionID string, providerErr error) error {

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin refund transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	// Change PENDING to FAILED.
	transaction, err := s.transactionRepo.MarkFailed(
		ctx,
		tx,
		transactionID,
		providerErr.Error(),
	)
	if err != nil {
		return fmt.Errorf("mark transaction failed: %w", err)
	}

	// Refund merchant balance in the same DB transaction.
	if err := s.merchantRepo.Credit(
		ctx,
		tx,
		transaction.MerchantID,
		transaction.Amount,
	); err != nil {
		return fmt.Errorf("refund merchant balance: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit refund transaction: %w", err)
	}

	log.Info(
		"Transfer failed and merchant refunded",
		"transaction_id", transactionID,
		"merchant_id", transaction.MerchantID,
		"amount", transaction.Amount,
		"reason", providerErr.Error(),
	)

	return nil
}
