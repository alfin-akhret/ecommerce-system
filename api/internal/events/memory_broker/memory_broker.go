package memorybroker

import (
	"context"
	"sync"

	"github.com/alfin-akhret/ecommerce-system/internal/events"
)

type MemoryBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]events.Handler
}

func NewMemoryBroker() *MemoryBroker {
	return &MemoryBroker{
		subscribers: make(map[string][]events.Handler),
	}
}

// implement Broker interface
func (b *MemoryBroker) Publish(ctx context.Context, event events.Event) {
	b.mu.RLock()
	handlers := b.subscribers[event.Name]
	b.mu.RUnlock()

	for _, h := range handlers {
		go h(ctx, event)
	}
}

func (b *MemoryBroker) Subscribe(eventName string, handler events.Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[eventName] = append(b.subscribers[eventName], handler)
}

func (b *MemoryBroker) Close() error {
	// not implemented
	return nil
}
