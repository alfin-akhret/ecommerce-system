package queue

import (
	"context"
	"errors"
	"time"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"go.uber.org/zap"
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
	ctx := context.Background()
	logger := helper.LoggerFromCtx(ctx)

	handler, ok := q.Registry.Get(job.Type)
	if !ok {
		err := errors.New("unknown job")
		logger.Info("Processing job", zap.String("error_message", err.Error()))
		return
	}

	logger.Info("Processing job", zap.String("job", job.Type))

	err := handler(context.Background(), job.Payload)
	if err != nil {
		if job.Retry < 3 {
			job.Retry++
			time.Sleep(500 * time.Millisecond)
			q.Enqueue(job)
		}
	}
}
