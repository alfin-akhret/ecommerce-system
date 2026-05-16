package queue

import "time"

type Job struct {
	Type      string
	Payload   []byte
	Retry     int
	MaxRetry  int
	Timeout   time.Duration
	CreatedAt time.Time
}
