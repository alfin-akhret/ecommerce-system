package events

import "time"

type Event struct {
	Name      string
	Payload   any
	CreatedAt time.Time
}

type OrderCreatedPayload struct {
	OrderID string
	Email   string
}
