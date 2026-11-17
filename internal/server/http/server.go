package http

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/Rambudhi/money-transfer/internal/config"
	"github.com/Rambudhi/money-transfer/internal/server/handler"
	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

func NewHTTPServer(cfg *config.Config, transferHandler *handler.TransferHandler) *http.Server {
	e := echo.New()

	SetupMiddleware(e)
	SetupRouter(e, transferHandler)

	addr := net.JoinHostPort(
		cfg.App.Address,
		cfg.App.Port,
	)

	return &http.Server{
		Addr:    addr,
		Handler: e,
	}
}

func Register(lc fx.Lifecycle, server *http.Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return err
			}

			slog.Info(
				"HTTP server started",
				"address", server.Addr,
			)

			go func() {
				if err := server.Serve(listener); err != nil &&
					err != http.ErrServerClosed {
					slog.Error(
						"HTTP server failed",
						"error", err,
					)
				}
			}()

			return nil
		},

		OnStop: func(ctx context.Context) error {
			slog.Info("Shutting down HTTP server")

			return server.Shutdown(ctx)
		},
	})
}
