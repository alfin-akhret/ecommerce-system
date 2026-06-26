package events

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Payload    json.RawMessage `json:"payload"`
	RawPayload []byte          `json:"-"`
	CreatedAt  time.Time       `json:"created_at"`
	RetryCount int             `json:"retry_count"`
}

type OrderCreatedPayload struct {
	OrderID string
	Email   string
}

type PaymentCallbackProcessedPayload struct {
	PaymentID string
	OrderID   string
	Status    string
	Email     string
}

type PaymentExpiredPayload struct {
	OrderID   string
	PaymentID string
	Email     string
}

type OutboxEvent struct {
	ID            string
	EventType     string
	Payload       []byte
	Status        string
	RetryCount    int
	CreatedAt     time.Time
	PublishedAt   *time.Time
	NextAttemptAt time.Time
	LockedAt      *time.Time
	LockedBy      *string
	LastError     *string
}

type InboxEvent struct {
	ID          string
	EventType   string
	Payload     []byte
	Status      string
	RetryCount  int
	CreatedAt   time.Time
	ProcessedAt *time.Time
	LastError   *string
}

const (
	InboxPending   = "PENDING"
	InboxProcessed = "PROCESSED"
)
