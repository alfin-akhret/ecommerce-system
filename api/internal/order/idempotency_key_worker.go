package order

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"go.uber.org/zap"
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
	logger := helper.LoggerFromCtx(ctx)

	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	ticker := time.NewTicker(w.interval)

	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		defer ticker.Stop()

		logger.Info("[Order iKey Worker] Order Idempotency Key deletion worker started")

		for {
			select {
			case <-ctx.Done():
				logger.Info("[Order iKey Worker] Stoping Order Idempotency Key deletion worker")
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
}

func (w *IdempotencyKeyDeleteWorker) run(ctx context.Context) {
	logger := helper.LoggerFromCtx(ctx)

	logger.Info("[Order iKey Worker] Running key deletion job...")

	keys, err := w.service.DeleteIdempotencyKey(ctx, w.batchSize)
	if err != nil {
		logger.Error("[Order iKey Worker] Failed to delete order idempotency keys")
		return
	}

	if len(keys) == 0 {
		logger.Info("[Order iKey Worker] No expire idempotency keys found")
		return
	}

	logger.Info("[Order iKey Worker] Keys deleted", zap.String("Key num", strconv.Itoa(len(keys))))

	for _, val := range keys {
		logger.Info("[Order iKey Worker]", zap.String("key", val.Key.String()))
	}
}
