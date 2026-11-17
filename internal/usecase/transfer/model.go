package transfer

import "github.com/Rambudhi/money-transfer/internal/domain/entities"

type TransferRequest struct {
	MerchantID         int64  `json:"merchant_id" validate:"required,gt=0"`
	Amount             int64  `json:"amount" validate:"required,gt=0"`
	DestinationAccount string `json:"destination_account" validate:"required"`
}

type TransferResponse struct {
	ID          string                     `json:"id"`
	MerchantID  int64                      `json:"merchant_id"`
	Amount      int64                      `json:"amount"`
	Status      entities.TransactionStatus `json:"status"`
	ReferenceID string                     `json:"reference_id"`
}
