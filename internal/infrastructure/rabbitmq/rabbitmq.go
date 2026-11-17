package rabbitmq

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Rambudhi/money-transfer/internal/config"
	"github.com/Rambudhi/money-transfer/internal/pkg/log"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/fx"
)

type RabbitMQ struct {
	Conn *amqp.Connection
}

func NewRabbitMQ(lc fx.Lifecycle, cfg *config.Config) (*RabbitMQ, error) {
	const (
		maxRetries  = 10
		retryDelay  = 3 * time.Second
		dialTimeout = 5 * time.Second
	)

	var conn *amqp.Connection
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		conn, err = amqp.DialConfig(
			cfg.RabbitMQ.URL,
			amqp.Config{
				Heartbeat: 10 * time.Second,
				Dial: func(network, addr string) (net.Conn, error) {
					return net.DialTimeout(
						network,
						addr,
						dialTimeout,
					)
				},
			},
		)

		if err == nil {
			log.Info("RabbitMQ connected")
			break
		}

		log.Warn(
			"Failed to connect to RabbitMQ",
			"attempt", attempt,
			"max_retries", maxRetries,
			"error", err,
		)

		if attempt == maxRetries {
			return nil, fmt.Errorf("rabbitmq connection failed after %d attempts: %w", maxRetries, err)
		}

		time.Sleep(retryDelay)
	}

	rmq := &RabbitMQ{
		Conn: conn,
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			if rmq.Conn == nil || rmq.Conn.IsClosed() {
				return nil
			}

			log.Info("Closing RabbitMQ connection")

			if err := rmq.Conn.Close(); err != nil {
				return fmt.Errorf("close rabbitmq connection: %w", err)
			}

			return nil
		},
	})

	return rmq, nil
}

func (r *RabbitMQ) NewChannel() (*amqp.Channel, error) {
	if r == nil || r.Conn == nil {
		return nil, fmt.Errorf("rabbitmq connection is not initialized")
	}

	if r.Conn.IsClosed() {
		return nil, fmt.Errorf("rabbitmq connection is closed")
	}

	channel, err := r.Conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("create rabbitmq channel: %w", err)
	}

	return channel, nil
}
