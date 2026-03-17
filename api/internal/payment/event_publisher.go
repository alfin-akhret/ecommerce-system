package payment

import "context"

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
						// best-effort safety: avoid panic crashing worker goroutine
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
