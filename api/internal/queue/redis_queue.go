package queue

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client   *redis.Client
	Registry *Registry
}

func (q *RedisQueue) Enqueue(ctx context.Context, job Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return q.client.LPush(ctx, "queue:jobs", data).Err()
}

func (q *RedisQueue) StartWorkers(ctx context.Context, n int) {
	for i := 0; i < n; i++ {
		go q.worker(ctx, i)
	}
}

// worker (blocking)
func (q *RedisQueue) worker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			res, err := q.client.BLMove(ctx,
				"queue:jobs",
				"queue:processing",
				"RIGHT",
				"LEFT",
				0).Result()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			var job Job
			if err := json.Unmarshal([]byte(res), &job); err != nil {
				q.client.LRem(ctx, "queue:processing", 1, res)
				continue
			}

			err = q.process(ctx, job)
			if err != nil {
				continue
			}

			q.client.LRem(ctx, "queue:processing", 1, res)
		}
	}
}

func (q *RedisQueue) StartRecoveryWorker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			q.recoverStaleJobs(ctx)
		}
	}
}

func (q *RedisQueue) recoverStaleJobs(ctx context.Context) {
	// ambil semua job di queue:processing
	jobs, err := q.client.LRange(ctx, "queue:processing", 0, -1).Result()
	if err != nil {
		return
	}

	for _, raw := range jobs {
		var job Job
		if err := json.Unmarshal([]byte(raw), &job); err != nil {
			// kalau corrupt buang dari processing
			q.client.LRem(ctx, "queue:processing", 1, raw)
			continue
		}

		// cek apakah job udah terlalu lama nyangkut
		if time.Since(job.CreatedAt) > job.Timeout*2 {
			// requeue ke jobs
			q.client.LPush(ctx, "queue:jobs", raw)
			// hapus dari processing
			q.client.LRem(ctx, "queueu:processing", 1, raw)
		}
	}
}

// retry with delay
func (q *RedisQueue) scheduleRetry(ctx context.Context, job Job) {
	data, _ := json.Marshal(job)

	score := float64(time.Now().Add(backoff(job.Retry)).Unix())

	q.client.ZAdd(ctx, "queue:delayed", redis.Z{
		Score:  score,
		Member: data,
	})
}

// shceduler
func (q *RedisQueue) StartScheduler(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := float64(time.Now().Unix())

			jobs, _ := q.client.ZRangeArgs(ctx, redis.ZRangeArgs{
				Key:     "queue:delayed",
				Start:   "0",
				Stop:    strconv.FormatFloat(now, 'f', -1, 64),
				ByScore: true,
			}).Result()

			for _, j := range jobs {
				q.client.LPush(ctx, "queue:jobs", j)
				q.client.ZRem(ctx, "queue:delayed", j)
			}
		}
	}
}

// DLQ
func (q *RedisQueue) pushDLQ(ctx context.Context, job Job) {
	data, _ := json.Marshal(job)
	q.client.LPush(ctx, "queue:dlq", data)
}

// main process
func (q *RedisQueue) process(ctx context.Context, job Job) error {
	handler, ok := q.Registry.Get(job.Type)
	if !ok {
		// todo: log error

		// push to DLQ
		q.pushDLQ(ctx, job)
		err := errors.New("failed to get job")
		return err
	}

	jobCtx, cancel := context.WithTimeout(ctx, job.Timeout)
	defer cancel()

	err := handler(jobCtx, job.Payload)
	if err != nil {
		// push to schedule retry
		if job.Retry < job.MaxRetry {

			// biar struct tidak mutasi langsung,
			// bahaya kalau concurrency naik
			// bisa race condition
			job := job  // copy dulu job nya
			job.Retry++ // baru mutasi

			// bikin konteks baru khusus untuk retry
			retryCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			q.scheduleRetry(retryCtx, job)
			return err
		}

		q.pushDLQ(ctx, job)
		return err
	}

	return nil
}
