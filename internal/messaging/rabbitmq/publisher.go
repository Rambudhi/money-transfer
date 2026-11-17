package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Rambudhi/money-transfer/internal/domain/entities"
	infrastructure "github.com/Rambudhi/money-transfer/internal/infrastructure/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TransferQueue       = "transfer.process"
	ConfirmationTimeout = 5 * time.Second
)

type Publisher struct {
	channel  *amqp.Channel
	confirms chan amqp.Confirmation
	mu       sync.Mutex
}

func NewPublisher(rmq *infrastructure.RabbitMQ) (*Publisher, error) {
	channel, err := rmq.NewChannel()
	if err != nil {
		return nil, err
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
		return nil, err
	}

	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("enable publisher confirmation: %w", err)
	}

	confirms := channel.NotifyPublish(
		make(chan amqp.Confirmation, 1),
	)

	return &Publisher{
		channel:  channel,
		confirms: confirms,
	}, nil
}

func (p *Publisher) PublishTransfer(ctx context.Context, transaction *entities.Transaction) error {
	payload := struct {
		TransactionID string `json:"transaction_id"`
	}{
		TransactionID: transaction.ID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal transfer message: %w", err)
	}

	return p.Publish(ctx, body)
}

func (p *Publisher) Publish(ctx context.Context, body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel == nil || p.channel.IsClosed() {
		return fmt.Errorf("rabbitmq publisher channel is closed")
	}

	err := p.channel.PublishWithContext(
		ctx,
		"",
		TransferQueue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish rabbitmq message: %w", err)
	}

	timer := time.NewTimer(ConfirmationTimeout)
	defer timer.Stop()

	select {
	case confirmation, ok := <-p.confirms:
		if !ok {
			return fmt.Errorf("rabbitmq confirmation channel closed")
		}

		if !confirmation.Ack {
			return fmt.Errorf("rabbitmq rejected published message")
		}

		return nil

	case <-ctx.Done():
		_ = p.channel.Close()
		return ctx.Err()

	case <-timer.C:
		_ = p.channel.Close()
		return fmt.Errorf("rabbitmq publisher confirmation timeout")
	}
}

func (p *Publisher) Close() error {
	if p.channel == nil || p.channel.IsClosed() {
		return nil
	}

	return p.channel.Close()
}
