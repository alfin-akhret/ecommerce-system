package order

import (
	"context"
	"log"
	"sync"
	"time"
)

type OrderService interface {
	DeleteIdempotencyKey(ctx context.Context, limit int) ([]DeletedKeys, error)
}

type IdempotencyKeyDeleteWorker struct {
	service   OrderService
	interval  time.Duration
	batchSize int
	wg        sync.WaitGroup
	cancel    context.CancelFunc
}

func NewIdempotencyKeyDeleteWorker(
	service OrderService,
	interval time.Duration,
	batchSize int,
) *IdempotencyKeyDeleteWorker {
	return &IdempotencyKeyDeleteWorker{
		service:   service,
		interval:  interval,
		batchSize: batchSize,
	}
}

func (w *IdempotencyKeyDeleteWorker) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	ticker := time.NewTicker(w.interval)

	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		defer ticker.Stop()

		log.Println("[Order iKey Worker] Order Idempotency Key deletion worker starter")

		for {
			select {
			case <-ctx.Done():
				log.Println("[Order iKey Worker] Stoping Order Idempotency Key deletion worker")
				return
			case <-ticker.C:
				w.run(ctx)
			}

		}
	}()
}

func (w *IdempotencyKeyDeleteWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	log.Println("[Order iKey Worker] Order Idempotency Deletion Worker stopped")
}

func (w *IdempotencyKeyDeleteWorker) run(ctx context.Context) {
	log.Println("[Order iKey Worker] Running key deletion job...")

	keys, err := w.service.DeleteIdempotencyKey(ctx, w.batchSize)
	if err != nil {
		log.Println("[Order iKey Worker] Failed to delete order idempotency keys")
		return
	}

	if len(keys) == 0 {
		log.Println("[Order iKey Worker] No expire idempotency keys found")
		return
	}

	log.Println("[Order iKey Worker] Keys deleted: ", len(keys))

	for _, val := range keys {
		log.Printf("Key: %v\n", val)
	}
}
