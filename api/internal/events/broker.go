package events

import "context"

type Handler func(ctx context.Context, event Event) error

type Broker interface {
	Publish(ctx context.Context, event Event)
	Subscribe(eventName string, handler Handler)
	Close() error
}
