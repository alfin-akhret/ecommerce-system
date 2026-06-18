package payment

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/events"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type PaymentService interface {
	ExpirePayments(ctx context.Context) ([]ExpiredPayment, error)
}
type EventPublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

type PaymentExpirationWorker struct {
	service PaymentService
	broker  events.Broker

	interval  time.Duration
	batchSzie int

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewPaymentExpirationWorker(
	service PaymentService,
	broker events.Broker,
	interval time.Duration,
	batchSize int,
) *PaymentExpirationWorker {
	return &PaymentExpirationWorker{
		service:   service,
		broker:    broker,
		interval:  interval,
		batchSzie: batchSize,
	}
}

func (w *PaymentExpirationWorker) Start(ctx context.Context) {
	logger := helper.LoggerFromCtx(ctx)

	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	ticker := time.NewTicker(w.interval)

	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		defer ticker.Stop()

		logger.Info("[Payment Worker] Payment expiration worker started")

		for {
			select {
			case <-ctx.Done():
				logger.Info("[Payment Worker] stopping payment expiration worker...")
				return

			case <-ticker.C:
				w.run(ctx)
			}
		}
	}()
}

func (w *PaymentExpirationWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}

	w.wg.Wait()

	ctx := context.Background()
	logger := helper.LoggerFromCtx(ctx)
	logger.Info("[Payment Worker] Payment expiratin worker stopped")
}

func (w *PaymentExpirationWorker) run(ctx context.Context) {
	logger := helper.LoggerFromCtx(ctx)
	logger.Info("[Payment Worker] running expiration job...")

	payments, err := w.service.ExpirePayments(ctx)
	if err != nil {
		logger.Error("[Payment Worker] failed to expire payments: ", zap.Error(err))
		return
	}

	if len(payments) == 0 {
		logger.Info("[Payment Worker] No expired payments found")
		return
	}

	logger.Info("[Payment Worker]", zap.String("payment expired", strconv.Itoa(len(payments))))

	// tracing
	// root span
	tr := otel.Tracer("payment-expiration-worker")
	ctx, span := tr.Start(ctx, "payment.expiration.worker.run")
	defer span.End()
	span.SetAttributes(
		attribute.Int("batch.size", len(payments)),
	)

	for _, p := range payments {
		// child span
		ctx, childSpan := tr.Start(ctx, "payment.expire.process")
		defer childSpan.End()
		childSpan.SetAttributes(
			attribute.String("payment_id", p.ID.String()),
			attribute.String("order_id", p.OrderID.String()),
		)

		event := events.Event{
			ID:   uuid.NewString(),
			Name: "payment.expired",
			Payload: events.PaymentExpiredPayload{
				OrderID:   p.OrderID.String(),
				PaymentID: p.ID.String(),
				Email:     "testingemail@gmail.com",
			},
		}

		w.broker.Publish(ctx, event, 0)

		logger.Info("[Payment Worker] event published for payment", zap.String("payment_id", p.ID.String()))
	}
}
