package queue

import "time"

type Job struct {
	Type    string
	Payload []byte
	Retry   int
	Timeout time.Duration
}
