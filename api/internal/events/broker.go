package events

import "context"

type Handler func(ctx context.Context, event Event) error

type Broker interface {
	Publish(ctx context.Context, event Event, retryCount int) error
	Subscribe(eventName string, subscriberName string, handler Handler)
	Close() error
}
