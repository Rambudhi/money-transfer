package transfer

import (
	"context"

	"github.com/Rambudhi/money-transfer/internal/domain/entities"
)

type Service interface {
	Transfer(ctx context.Context, request TransferRequest) (*entities.Transaction, error)
}

type WorkerService interface {
	ProcessTransfer(ctx context.Context, transactionID string) error
}

type Provider interface {
	Transfer(ctx context.Context, transaction *entities.Transaction) error
}
