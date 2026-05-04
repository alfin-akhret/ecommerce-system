package queue

import "context"

// Mapping job -> handler

type Registry struct {
	Handlers map[string]Handler
}

type Handler func(ctx context.Context, payload []byte) error

func (r *Registry) Register(name string, h Handler) {
	r.Handlers[name] = h
}

func (r *Registry) Get(name string) (Handler, bool) {
	h, ok := r.Handlers[name]
	return h, ok
}
