package entities

import "time"

type OutboxStatus string

const (
    OutboxStatusPending   OutboxStatus = "PENDING"
    OutboxStatusPublished OutboxStatus = "PUBLISHED"
)

type MessageOutbox struct {
    ID          int64
    AggregateID string
    EventType   string
    Payload     []byte
    Status      OutboxStatus
    RetryCount  int
    PublishedAt *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
