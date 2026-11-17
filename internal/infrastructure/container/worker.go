package container

import (
	"github.com/Rambudhi/money-transfer/internal/config"
	repository "github.com/Rambudhi/money-transfer/internal/domain/repositories"
	"github.com/Rambudhi/money-transfer/internal/infrastructure/postgres"
	"github.com/Rambudhi/money-transfer/internal/infrastructure/rabbitmq"
	messagingrabbitmq "github.com/Rambudhi/money-transfer/internal/messaging/rabbitmq"
	"github.com/Rambudhi/money-transfer/internal/pkg/database"
	"github.com/Rambudhi/money-transfer/internal/provider"
	"github.com/Rambudhi/money-transfer/internal/usecase/transfer"
	worker "github.com/Rambudhi/money-transfer/internal/worker"
	handler "github.com/Rambudhi/money-transfer/internal/worker/handler"
	"go.uber.org/fx"
)

var WorkerModule = fx.Options(
	fx.Provide(
		config.LoadConfig,
		postgres.NewDB,
		postgres.ProvidePostgresPool,
		rabbitmq.NewRabbitMQ,
	),

	fx.Provide(
		repository.NewMerchantRepository,
		repository.NewTransactionRepository,
		repository.NewMessageOutboxRepository,
	),

	fx.Provide(
		messagingrabbitmq.NewConsumer,
	),

	fx.Provide(
		fx.Annotate(
			provider.NewMockProvider,
			fx.As(new(transfer.Provider)),
		),
	),

	fx.Provide(
		fx.Annotate(
			transfer.NewService,
			fx.As(new(transfer.WorkerService)),
		),
		handler.NewTransferHandler,
		worker.NewWorker,
	),

	fx.Invoke(database.Initialize),
	fx.Invoke(worker.Register),
)
