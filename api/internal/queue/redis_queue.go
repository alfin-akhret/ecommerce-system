package queue

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client   *redis.Client
	Registry *Registry
}

func (q *RedisQueue) Enqueue(ctx context.Context, job Job) error {
	data, _ := json.Marshal(job)

	return q.client.LPush(ctx, "queue:jobs", data).Err()
}

// worker (blocking)
func (q *RedisQueue) StartWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			res, err := q.client.BRPopLPush(ctx, "queue:jobs", "queue:processing", 0).Result()
			if err != nil {
				continue
			}

			var job Job
			json.Unmarshal([]byte(res), &job)

			q.process(ctx, job)

			q.client.LRem(ctx, "queue:processing", 1, res)
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

			jobs, _ := q.client.ZRangeByScore(ctx, "queue:delayed", &redis.ZRangeBy{
				Min: "0",
				Max: strconv.FormatFloat(now, 'f', -1, 64),
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
func (q *RedisQueue) process(ctx context.Context, job Job) {
	handler, ok := q.Registry.Get(job.Type)
	if !ok {
		// todo: log error

		// push to DLQ
		q.pushDLQ(ctx, job)
		return
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

			q.scheduleRetry(ctx, job)
			return
		}

		q.pushDLQ(ctx, job)
		return
	}
}
