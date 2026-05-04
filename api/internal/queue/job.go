package queue

import "context"

type Job struct {
	Type    string
	Payload []byte
	Retry   int
}

type Handler func(ctx context.Context, payload []byte) error
