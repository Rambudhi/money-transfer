package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Rambudhi/money-transfer/internal/usecase/transfer"
)

type TransferMessage struct {
	TransactionID string `json:"transaction_id"`
}

type TransferHandler struct {
	transferService transfer.WorkerService
}

func NewTransferHandler(
	transferService transfer.WorkerService,
) *TransferHandler {
	return &TransferHandler{
		transferService: transferService,
	}
}

// Handle processes transfer messages received from RabbitMQ.
func (h *TransferHandler) HandleTransfer(ctx context.Context, body []byte) error {
	var msg TransferMessage

	if err := json.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("decode transfer message: %w", err)
	}

	if msg.TransactionID == "" {
		return errors.New("transaction_id is required")
	}

	if err := h.transferService.ProcessTransfer(ctx, msg.TransactionID); err != nil {
		return fmt.Errorf("process transfer %s: %w", msg.TransactionID, err)
	}

	return nil
}
