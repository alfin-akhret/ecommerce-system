package events

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type EventService interface {
	GetUnpublishedEvents(ctx context.Context, limit int, lockedBy string, leaseTimeout time.Duration) ([]Event, error) // todo: implement this
	UpdateOutboxEventsStatus(ctx context.Context, ids []string, status string) error
	MarkPublished(ctx context.Context, publishedIds []string) error
	MarkPublishFailed(ctx context.Context, eventID string, errString string, maxRetry int, backoff time.Duration) error
}

type EventPublisherWorker struct {
	service EventService
	broker  Broker

	interval     time.Duration // interval ticker workernya
	batchSize    int
	lockedBy     string
	leaseTimeout time.Duration // interval exponential backoff utk retry publish
	maxRetry     int

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewEventPublisherWorker(
	service EventService,
	broker Broker,
	interval time.Duration,
	batchSize int,
	lockedBy string,
	leaseTimeout time.Duration,
	maxRetry int,
) *EventPublisherWorker {
	return &EventPublisherWorker{
		service:      service,
		broker:       broker,
		interval:     interval,
		batchSize:    batchSize,
		lockedBy:     lockedBy,
		leaseTimeout: leaseTimeout,
		maxRetry:     maxRetry,
	}
}

func (w *EventPublisherWorker) Start(ctx context.Context) {
	logger := helper.LoggerFromCtx(ctx)

	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	ticker := time.NewTicker(w.interval)

	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		defer ticker.Stop()

		logger.Info("[Event Publisher Worker] Event Publisher worker started")

		for {
			select {
			case <-ctx.Done():
				logger.Info("[Event Publisher Worker] stopping Event Publisher worker...")
				return

			case <-ticker.C:
				w.run(ctx)
			}
		}
	}()
}

func (w *EventPublisherWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}

	w.wg.Wait()

	ctx := context.Background()
	logger := helper.LoggerFromCtx(ctx)
	logger.Info("[Event Publisher Worker] Event Publisher Worker stopped")
}

func (w *EventPublisherWorker) run(ctx context.Context) {
	logger := helper.LoggerFromCtx(ctx)
	logger.Info("[Event Publisher Worker] running publishing job...")

	events, err := w.service.GetUnpublishedEvents(
		ctx,
		w.batchSize,
		w.lockedBy,
		w.leaseTimeout,
	)
	if err != nil {
		logger.Error("[Event Publisher Worker] failed to get unpublished events: ", zap.Error(err))
		return
	}

	if len(events) == 0 {
		logger.Info("[Event Publisher Worker] No unpublished events found")
		return
	}

	logger.Info("[Event Publisher Worker]", zap.String("events published", strconv.Itoa(len(events))))

	// tracing
	// root span
	tr := otel.Tracer("event-publisher-worker")
	ctx, span := tr.Start(ctx, "event.publisher.worker.run")
	defer span.End()
	span.SetAttributes(
		attribute.Int("batch.size", len(events)),
	)

	var publishedIDs []string
	for _, ev := range events {
		// child span
		ctx, childSpan := tr.Start(ctx, "event.publisher.process")
		defer childSpan.End()
		childSpan.SetAttributes(
			attribute.String("event_id", ev.ID),
		)

		err := w.broker.Publish(ctx, ev, 0)
		if err != nil {
			backoff := calculateBackoff(ev.RetryCount)
			_ = w.service.MarkPublishFailed(
				ctx,
				ev.ID,
				err.Error(),
				w.maxRetry,
				backoff,
			)
			continue
		}

		publishedIDs = append(publishedIDs, ev.ID)

		//logger.Info("[Event Publisher Worker] event published for payment", zap.String("payment_id", ev.ID))
	}

	if len(publishedIDs) > 0 {
		_ = w.service.MarkPublished(ctx, publishedIDs)
	}

}

func calculateBackoff(retryCount int) time.Duration {
	base := time.Second
	delay := time.Duration(1<<retryCount) * base

	maxDelay := time.Minute
	if delay > maxDelay {
		return maxDelay
	}

	return delay
}
