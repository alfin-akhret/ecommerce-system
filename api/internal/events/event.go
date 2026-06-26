package events

import "time"

type Event struct {
	ID         string
	Name       string
	Payload    any
	CreatedAt  time.Time
	RetryCount int
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
