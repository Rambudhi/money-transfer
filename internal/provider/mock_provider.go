package provider

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/Rambudhi/money-transfer/internal/domain/entities"
)

type MockProvider struct {
	minDelay time.Duration
	maxDelay time.Duration
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		minDelay: 100 * time.Millisecond,
		maxDelay: 1000 * time.Millisecond,
	}
}

func (p *MockProvider) Transfer(ctx context.Context, transaction *entities.Transaction) error {

	delayRange := p.maxDelay - p.minDelay

	var delay time.Duration

	if delayRange > 0 {
		delay = p.minDelay + time.Duration(
			rand.Int63n(int64(delayRange)),
		)
	} else {
		delay = p.minDelay
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-timer.C:
	}

	// Simulate 20% provider failure.
	if rand.Intn(100) < 20 {
		return errors.New("mock provider: transfer rejected")
	}

	return nil
}
