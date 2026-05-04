package queue

// Mapping job -> handler

type Registry struct {
	Handlers map[string]Handler
}

func (r *Registry) Register(name string, h Handler) {
	r.Handlers[name] = h
}

func (r *Registry) Get(name string) (Handler, bool) {
	h, ok := r.Handlers[name]
	return h, ok
}
