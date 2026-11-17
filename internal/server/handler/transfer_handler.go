package handler

import (
	"net/http"

	"github.com/Rambudhi/money-transfer/internal/pkg/log"
	"github.com/Rambudhi/money-transfer/internal/pkg/validation"
	"github.com/Rambudhi/money-transfer/internal/usecase/transfer"

	"github.com/labstack/echo/v4"
)

type TransferHandler struct {
	transferService transfer.Service
}

func NewTransferHandler(transferService transfer.Service) *TransferHandler {
	return &TransferHandler{
		transferService: transferService,
	}
}

func (h *TransferHandler) CreateTransfer(c echo.Context) error {
	var request transfer.TransferRequest

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
	}

	if err := validation.Validate(request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
	}

	transaction, err := h.transferService.Transfer(c.Request().Context(), request)
	if err != nil {
		log.Error("Failed to create transfer", "merchant_id", request.MerchantID, "amount", request.Amount, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{"message": err.Error()})
	}

	return c.JSON(http.StatusAccepted, transfer.TransferResponse{
		ID:          transaction.ID,
		MerchantID:  transaction.MerchantID,
		Amount:      transaction.Amount,
		Status:      transaction.Status,
		ReferenceID: transaction.ReferenceID,
	})
}
