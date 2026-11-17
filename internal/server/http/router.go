package http

import (
	"net/http"

	"github.com/Rambudhi/money-transfer/internal/server/handler"
	"github.com/labstack/echo/v4"
)

func SetupRouter(
	e *echo.Echo,
	transferHandler *handler.TransferHandler,
) {
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	e.POST("/transfer", transferHandler.CreateTransfer)
}
