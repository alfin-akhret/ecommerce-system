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
