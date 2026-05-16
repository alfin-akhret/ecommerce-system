package queue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	Client   *redis.Client
	Registry *Registry
}

const (
	queueJobsKey                = "queue:jobs"
	queueProcessingKey          = "queue:processing"
	queueProcessingDeadlinesKey = "queue:processing:deadlines"
	queueDelayedKey             = "queue:delayed"
	queueDLQKey                 = "queue:dlq"
	queueDedupPrefix            = "queue:job:"

	defaultJobTimeout    = 2 * time.Second
	defaultMaxRetry      = 3
	dedupTTL             = 24 * time.Hour
	recoveryBatchSize    = 100
	schedulerBatchSize   = 100
	recoveryInterval     = 10 * time.Second
	schedulerInterval    = 1 * time.Second
	minProcessingLease   = 30 * time.Second
	heartbeatMinInterval = 1 * time.Second
)

var (
	enqueueScript = redis.NewScript(`
if redis.call("SET", KEYS[2], "1", "NX", "EX", ARGV[2]) then
	redis.call("LPUSH", KEYS[1], ARGV[1])
	return 1
end
return 0
`)

	ackProcessingScript = redis.NewScript(`
local removed = redis.call("LREM", KEYS[1], 1, ARGV[1])
redis.call("ZREM", KEYS[2], ARGV[1])
return removed
`)

	requeueProcessingScript = redis.NewScript(`
local removed = redis.call("LREM", KEYS[1], 1, ARGV[1])
if removed > 0 then
	redis.call("ZREM", KEYS[2], ARGV[1])
	redis.call("LPUSH", KEYS[3], ARGV[1])
	return 1
end
redis.call("ZREM", KEYS[2], ARGV[1])
return 0
`)

	moveDueDelayedScript = redis.NewScript(`
local jobs = redis.call("ZRANGEBYSCORE", KEYS[1], "-inf", ARGV[1], "LIMIT", 0, ARGV[2])
local moved = 0
for _, job in ipairs(jobs) do
	if redis.call("ZREM", KEYS[1], job) == 1 then
		redis.call("LPUSH", KEYS[2], job)
		moved = moved + 1
	end
end
return moved
`)
)

func (q *RedisQueue) Enqueue(ctx context.Context, job Job) error {
	job = normalizeJob(job)

	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return enqueueScript.Run(ctx, q.Client,
		[]string{queueJobsKey, queueDedupPrefix + job.ID},
		data,
		int64(dedupTTL/time.Second),
	).Err()
}

func (q *RedisQueue) StartAll(ctx context.Context, n int) {
	q.StartWorkers(ctx, n)
	go q.StartScheduler(ctx)
	go q.StartRecoveryWorker(ctx)
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
			res, err := q.Client.BLMove(ctx,
				queueJobsKey,
				queueProcessingKey,
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
				q.ackProcessing(ctx, res)
				continue
			}

			job = normalizeJob(job)
			q.trackProcessing(ctx, res, job)
			stopHeartbeat := q.startHeartbeat(ctx, res, job)
			err = q.process(ctx, job)
			stopHeartbeat()

			// Always ack the processing entry after process() has either succeeded,
			// scheduled retry, or pushed the job to DLQ.
			q.ackProcessing(ctx, res)

			if err != nil {
				continue
			}
		}
	}
}

func (q *RedisQueue) StartRecoveryWorker(ctx context.Context) {
	ticker := time.NewTicker(recoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.recoverStaleJobs(ctx)
		}
	}
}

func (q *RedisQueue) recoverStaleJobs(ctx context.Context) {
	q.ensureProcessingDeadlines(ctx)

	now := strconv.FormatInt(time.Now().Unix(), 10)
	jobs, err := q.Client.ZRangeByScore(ctx, queueProcessingDeadlinesKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   now,
		Count: recoveryBatchSize,
	}).Result()
	if err != nil {
		return
	}

	for _, raw := range jobs {
		q.requeueProcessing(ctx, raw)
	}
}

// retry with delay
func (q *RedisQueue) scheduleRetry(ctx context.Context, job Job) {
	job = normalizeJob(job)
	data, _ := json.Marshal(job)

	score := float64(time.Now().Add(backoff(job.Retry)).Unix())

	q.Client.ZAdd(ctx, queueDelayedKey, redis.Z{
		Score:  score,
		Member: data,
	})
}

// shceduler
func (q *RedisQueue) StartScheduler(ctx context.Context) {
	ticker := time.NewTicker(schedulerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := float64(time.Now().Unix())
			moveDueDelayedScript.Run(ctx, q.Client,
				[]string{queueDelayedKey, queueJobsKey},
				strconv.FormatFloat(now, 'f', -1, 64),
				schedulerBatchSize,
			)
		}
	}
}

// DLQ
func (q *RedisQueue) pushDLQ(ctx context.Context, job Job) {
	job = normalizeJob(job)
	data, _ := json.Marshal(job)
	q.Client.LPush(ctx, queueDLQKey, data)
}

// main process
func (q *RedisQueue) process(ctx context.Context, job Job) error {
	job = normalizeJob(job)

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

	errCh := make(chan error, 1)
	go func() {
		errCh <- handler(jobCtx, job.Payload)
	}()

	var err error
	select {
	case err = <-errCh:
	case <-jobCtx.Done():
		err = jobCtx.Err()
	}

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			q.pushDLQ(ctx, job)
			return err
		}

		// push to schedule retry
		if job.Retry < job.MaxRetry {

			// biar struct tidak mutasi langsung,
			// bahaya kalau concurrency naik
			// bisa race condition
			// job := job  // copy dulu job nya
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

func normalizeJob(job Job) Job {
	if job.ID == "" {
		job.ID = jobFingerprint(job)
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.Timeout <= 0 {
		job.Timeout = defaultJobTimeout
	}
	if job.MaxRetry <= 0 {
		job.MaxRetry = defaultMaxRetry
	}
	return job
}

func jobFingerprint(job Job) string {
	sum := sha256.Sum256(append([]byte(job.Type), job.Payload...))
	return hex.EncodeToString(sum[:])
}

func processingLease(job Job) time.Duration {
	lease := job.Timeout * 3
	if lease < minProcessingLease {
		return minProcessingLease
	}
	return lease
}

func heartbeatInterval(job Job) time.Duration {
	interval := processingLease(job) / 3
	if interval < heartbeatMinInterval {
		return heartbeatMinInterval
	}
	return interval
}

func (q *RedisQueue) processingDeadline(job Job) float64 {
	return float64(time.Now().Add(processingLease(job)).Unix())
}

func (q *RedisQueue) trackProcessing(ctx context.Context, raw string, job Job) {
	q.Client.ZAdd(ctx, queueProcessingDeadlinesKey, redis.Z{
		Score:  q.processingDeadline(job),
		Member: raw,
	})
}

func (q *RedisQueue) startHeartbeat(ctx context.Context, raw string, job Job) func() {
	heartbeatCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(heartbeatInterval(job))
		defer ticker.Stop()

		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				q.trackProcessing(heartbeatCtx, raw, job)
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}

func (q *RedisQueue) ackProcessing(ctx context.Context, raw string) {
	ackProcessingScript.Run(ctx, q.Client,
		[]string{queueProcessingKey, queueProcessingDeadlinesKey},
		raw,
	)
}

func (q *RedisQueue) requeueProcessing(ctx context.Context, raw string) {
	requeueProcessingScript.Run(ctx, q.Client,
		[]string{queueProcessingKey, queueProcessingDeadlinesKey, queueJobsKey},
		raw,
	)
}

func (q *RedisQueue) ensureProcessingDeadlines(ctx context.Context) {
	jobs, err := q.Client.LRange(ctx, queueProcessingKey, 0, -1).Result()
	if err != nil {
		return
	}

	for _, raw := range jobs {
		if err := q.Client.ZScore(ctx, queueProcessingDeadlinesKey, raw).Err(); err == nil {
			continue
		}

		var job Job
		if err := json.Unmarshal([]byte(raw), &job); err != nil {
			q.ackProcessing(ctx, raw)
			continue
		}

		q.trackProcessing(ctx, raw, normalizeJob(job))
	}
}
