package worker

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Rambudhi/money-transfer/internal/messaging/rabbitmq"
	"github.com/Rambudhi/money-transfer/internal/worker/handler"
	"go.uber.org/fx"
)

type Worker struct {
	consumer *rabbitmq.Consumer
	handler  *handler.TransferHandler

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewWorker(consumer *rabbitmq.Consumer, transferHandler *handler.TransferHandler) *Worker {
	return &Worker{
		consumer: consumer,
		handler:  transferHandler,
	}
}

func Register(lc fx.Lifecycle, worker *Worker) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return worker.Start()
		},
		OnStop: func(ctx context.Context) error {
			return worker.Shutdown(ctx)
		},
	})
}

func (w *Worker) Start() error {
	workerCtx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel

	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		slog.Info("Transfer worker started")

		if err := w.consumer.Consume(
			workerCtx,
			w.handler.HandleTransfer,
		); err != nil {
			if workerCtx.Err() == nil {
				slog.Error("Transfer worker stopped unexpectedly", "error", err)
			}
		}

		slog.Info("Transfer worker stopped")
	}()

	return nil
}

func (w *Worker) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down transfer worker")

	if w.cancel != nil {
		w.cancel()
	}

	done := make(chan struct{})

	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Transfer worker shutdown completed")
		return nil

	case <-ctx.Done():
		slog.Warn("Transfer worker shutdown timeout", "error", ctx.Err())
		return ctx.Err()
	}
}
