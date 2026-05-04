package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"go.uber.org/zap"
)

// worker Pool
func (q *Queue) StartWorkers(ctx context.Context, n int) {
	for i := 0; i < n; i++ {
		go q.worker(ctx, i)
	}

	// DLQ worker
	go func() {
		for job := range q.Dlq {
			fmt.Println("DLQ:", job.Type)
		}
	}()
}

// worker loop
func (q *Queue) worker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("worker stopped:", id)
			return
		case job := <-q.Jobs:
			q.process(job)
		}
	}
}

// process job + retry
func (q *Queue) process(job Job) {
	ctx := context.Background()
	logger := helper.LoggerFromCtx(ctx)

	handler, ok := q.Registry.Get(job.Type)
	if !ok {
		err := errors.New("unknown job")
		logger.Info("Processing job", zap.String("error_message", err.Error()))
		q.Dlq <- job
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	logger.Info("Processing job", zap.String("job", job.Type))

	err := handler(ctx, job.Payload)
	if err != nil {
		if job.Retry < 3 {
			job.Retry++

			// non-blocking retry using goroutine
			go func(j Job) {
				time.Sleep(backoff(job.Retry))
				q.Enqueue(job)
			}(job)

			return
		}
		q.Dlq <- job
	}
}

// exponential backoff
func backoff(attempt int) time.Duration {
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}
