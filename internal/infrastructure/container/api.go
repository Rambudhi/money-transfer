package container

import (
	"github.com/Rambudhi/money-transfer/internal/config"
	repository "github.com/Rambudhi/money-transfer/internal/domain/repositories"
	"github.com/Rambudhi/money-transfer/internal/infrastructure/postgres"
	"github.com/Rambudhi/money-transfer/internal/infrastructure/rabbitmq"
	messagingrabbitmq "github.com/Rambudhi/money-transfer/internal/messaging/rabbitmq"
	"github.com/Rambudhi/money-transfer/internal/pkg/database"
	"github.com/Rambudhi/money-transfer/internal/provider"
	handler "github.com/Rambudhi/money-transfer/internal/server/handler"
	httpserver "github.com/Rambudhi/money-transfer/internal/server/http"
	"github.com/Rambudhi/money-transfer/internal/usecase/transfer"

	"go.uber.org/fx"
)

var APIModule = fx.Options(
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
		fx.Annotate(
			messagingrabbitmq.NewPublisher,
			fx.As(new(messagingrabbitmq.TransferPublisher)),
		),
		messagingrabbitmq.NewOutboxPublisher,
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
			fx.As(new(transfer.Service)),
		),
	),

	fx.Provide(
		handler.NewTransferHandler,
		httpserver.NewHTTPServer,
	),

	fx.Invoke(database.Initialize),
	fx.Invoke(messagingrabbitmq.RegisterOutboxPublisher),
	fx.Invoke(httpserver.Register),
)
