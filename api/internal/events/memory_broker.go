package events

import (
	"context"
	"sync"
)

type MemoryBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]Handler
}

func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{
		subscribers: make(map[string][]Handler),
	}
}

// implement Broker interface
func (b *MemoryBroker) Publish(ctx context.Context, event Event) {
	b.mu.RLock()
	handlers := b.subscribers[event.Name]
	b.mu.RUnlock()

	for _, h := range handlers {
		go h(ctx, event)
	}
}

func (b *MemoryBroker) Subscribe(eventName string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[eventName] = append(b.subscribers[eventName], handler)
}
