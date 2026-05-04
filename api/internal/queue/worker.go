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
	// main worker
	for i := 0; i < n; i++ {
		go q.worker(ctx, i)
	}

	// DLQ worker
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job, ok := <-q.Dlq:
				if !ok {
					return
				}
				fmt.Println("DLQ:", job.Type)
			}
		}
	}()
}

// worker loop
func (q *Queue) worker(ctx context.Context, id int) {
	// recover jika worker panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("worker panic recovered:", id, r)
			time.Sleep(1 * time.Second)
			go q.worker(ctx, id) // restart worker
		}
	}()

	// proses job
	for {
		select {
		case <-ctx.Done():
			fmt.Println("worker stopped:", id)
			return
		case job, ok := <-q.Jobs:
			if !ok {
				return
			}
			q.process(ctx, job)
		}
	}
}

// process job + retry
func (q *Queue) process(ctx context.Context, job Job) {

	logger := helper.LoggerFromCtx(ctx)
	if logger == nil {
		logger = zap.NewNop()
	}

	handler, ok := q.Registry.Get(job.Type)
	if !ok {
		err := errors.New("unknown job")
		logger.Error("Unknown job type",
			zap.String("job", job.Type),
			zap.String("error_message", err.Error()))

		select {
		case q.Dlq <- job:
		default:
			logger.Error("DLQ full, dropping job", zap.String("job", job.Type))
		}

		return
	}

	// determine timeout per job (fallback to default 2s)
	timeout := 2 * time.Second
	if job.Timeout > 0 {
		timeout = job.Timeout
	}

	jobCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	logger.Info("Start job", zap.String("job", job.Type))

	err := handler(jobCtx, job.Payload)
	if err != nil {
		if job.Retry < 3 {
			job.Retry++

			logger.Info("Retry job",
				zap.String("job", job.Type),
				zap.Int("retry", job.Retry),
			)

			// non-blocking retry using goroutine
			// context aware retry
			go func(j Job) {
				select {
				case <-time.After(backoff(j.Retry)):
					q.Enqueue(j)
				case <-ctx.Done():
					return
				}
			}(job)

			return
		}

		logger.Warn("Failed job", zap.String("job", job.Type))

		select {
		case q.Dlq <- job:
		default:
			logger.Error("DLQ full, dropping job", zap.String("job", job.Type))
		}

		return
	}

	logger.Info("Success job", zap.String("job", job.Type))
}

// exponential backoff
func backoff(attempt int) time.Duration {
	return time.Duration(1<<attempt) * 500 * time.Millisecond
}
