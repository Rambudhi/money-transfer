package entities

import "time"

type TransactionStatus string

const (
	TransactionStatusPending TransactionStatus = "PENDING"
	TransactionStatusSuccess TransactionStatus = "SUCCESS"
	TransactionStatusFailed  TransactionStatus = "FAILED"
)

type Transaction struct {
	ID                 string
	ReferenceID        string
	MerchantID         int64
	Amount             int64
	DestinationAccount string
	Status             TransactionStatus
	FailureReason      string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
