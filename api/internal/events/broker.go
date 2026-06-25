package events

import "context"

type Handler func(ctx context.Context, event Event) error

type Broker interface {
	Publish(ctx context.Context, event Event, retryCount int) (error, string)
	Subscribe(ctx context.Context, eventName string, subscriberName string, handler Handler)
	Close() error
}
