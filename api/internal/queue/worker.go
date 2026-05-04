package queue

import (
	"context"
	"fmt"
	"time"
)

// worker loop
func (q *Queue) StartWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-q.Jobs:
			q.process(job)
		}
	}
}

// process job + retry
func (q *Queue) process(job Job) {
	handler, ok := q.Registry.Get(job.Type)
	if !ok {
		fmt.Println("unknown job:", job.Type)
		return
	}

	err := handler(context.Background(), job.Payload)
	if err != nil {
		if job.Retry < 3 {
			job.Retry++
			time.Sleep(500 * time.Millisecond)
			q.Enqueue(job)
		}
	}
}
