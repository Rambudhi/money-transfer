package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	infrastructure "github.com/Rambudhi/money-transfer/internal/infrastructure/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TransferConsumerTag = "money-transfer-worker"
	ConsumerPrefetch    = 10
	ConsumerConcurrency = 10
)

type MessageHandler func(
	ctx context.Context,
	body []byte,
) error

type Consumer struct {
	channel *amqp.Channel
}

func NewConsumer(
	rmq *infrastructure.RabbitMQ,
) (*Consumer, error) {
	channel, err := rmq.NewChannel()
	if err != nil {
		return nil, fmt.Errorf("create rabbitmq channel: %w", err)
	}

	_, err = channel.QueueDeclare(
		TransferQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("declare transfer queue: %w", err)
	}

	if err := channel.Qos(
		ConsumerPrefetch,
		0,
		false,
	); err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("set consumer qos: %w", err)
	}

	return &Consumer{
		channel: channel,
	}, nil
}

func (c *Consumer) Consume(
	ctx context.Context,
	handler MessageHandler,
) error {
	if handler == nil {
		return errors.New("message handler is required")
	}

	if c.channel == nil || c.channel.IsClosed() {
		return errors.New("rabbitmq channel is closed")
	}

	deliveries, err := c.channel.Consume(
		TransferQueue,
		TransferConsumerTag,
		false, // manual ACK
		false, // exclusive
		false, // no local
		false, // no wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("start rabbitmq consumer: %w", err)
	}

	semaphore := make(chan struct{}, ConsumerConcurrency)

	var wg sync.WaitGroup

	slog.Info(
		"RabbitMQ consumer started",
		"queue", TransferQueue,
		"prefetch", ConsumerPrefetch,
		"concurrency", ConsumerConcurrency,
	)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping RabbitMQ consumer")

			if err := c.channel.Cancel(
				TransferConsumerTag,
				false,
			); err != nil {
				slog.Error(
					"Failed to cancel RabbitMQ consumer",
					"error", err,
				)
			}

			wg.Wait()

			slog.Info("RabbitMQ consumer stopped")
			return nil

		case delivery, ok := <-deliveries:
			if !ok {
				wg.Wait()

				if ctx.Err() != nil {
					return nil
				}

				return errors.New("rabbitmq delivery channel closed")
			}

			select {
			case semaphore <- struct{}{}:

			case <-ctx.Done():
				_ = delivery.Nack(false, true)

				if err := c.channel.Cancel(
					TransferConsumerTag,
					false,
				); err != nil {
					slog.Error(
						"Failed to cancel RabbitMQ consumer",
						"error", err,
					)
				}

				wg.Wait()
				return nil
			}

			wg.Add(1)

			go func(msg amqp.Delivery) {
				defer wg.Done()

				defer func() {
					<-semaphore
				}()

				if err := handler(ctx, msg.Body); err != nil {
					slog.Error(
						"Failed to process transfer message",
						"message_id", msg.MessageId,
						"error", err,
					)

					if nackErr := msg.Nack(false, true); nackErr != nil {
						slog.Error(
							"Failed to nack RabbitMQ message",
							"message_id", msg.MessageId,
							"error", nackErr,
						)
					}

					return
				}

				if err := msg.Ack(false); err != nil {
					slog.Error(
						"Failed to ack RabbitMQ message",
						"message_id", msg.MessageId,
						"error", err,
					)
					return
				}

				slog.Info(
					"RabbitMQ message processed successfully",
					"message_id", msg.MessageId,
				)
			}(delivery)
		}
	}
}

func (c *Consumer) Close() error {
	if c.channel == nil || c.channel.IsClosed() {
		return nil
	}

	return c.channel.Close()
}
