package payment

import (
	"context"
	"log"
)

type EventHandler func(ctx context.Context, payload any)

type InMemoryPublisher struct {
	handlers map[string][]EventHandler
}

func NewInMemoryPublisher() *InMemoryPublisher {
	return &InMemoryPublisher{
		handlers: make(map[string][]EventHandler),
	}
}

func (p *InMemoryPublisher) Publish(ctx context.Context, topic string, payload any) error {
	if hs, ok := p.handlers[topic]; ok {
		for _, h := range hs {
			go func(handler EventHandler) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[EventPublisher] handler panic: %v\n", r)
					}
				}()
				handler(ctx, payload)
			}(h) // async
		}
	}
	return nil
}

func (p *InMemoryPublisher) Subscribe(topic string, handler EventHandler) {
	p.handlers[topic] = append(p.handlers[topic], handler)
}
