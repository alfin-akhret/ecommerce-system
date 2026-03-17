package payment

import (
	"context"
	"log"
	"sync"
	"time"
)

type PaymentService interface {
	ExpirePayments(ctx context.Context) ([]ExpiredPayment, error)
}
type EventPublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

type PaymentExpirationWorker struct {
	service   PaymentService
	publisher EventPublisher

	interval  time.Duration
	batchSzie int

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewPaymentExpirationWorker(
	service PaymentService,
	publisher EventPublisher,
	interval time.Duration,
	batchSize int,
) *PaymentExpirationWorker {
	return &PaymentExpirationWorker{
		service:   service,
		publisher: publisher,
		interval:  interval,
		batchSzie: batchSize,
	}
}

func (w *PaymentExpirationWorker) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	ticker := time.NewTicker(w.interval)

	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		defer ticker.Stop()

		log.Println("[Worker] Payment expiration worker started")

		for {
			select {
			case <-ctx.Done():
				log.Println("[Worker] stopping payment expiration worker...")
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
	log.Println("[Worker] Payment expiratin worker stopped")
}

func (w *PaymentExpirationWorker) run(ctx context.Context) {
	log.Println("[Worker] running expiration job...")

	payments, err := w.service.ExpirePayments(ctx)
	if err != nil {
		log.Println("[Worker] failed to expire payments: ", err)
		return
	}

	if len(payments) == 0 {
		log.Println("[Worker] No expired payments found")
		return
	}

	log.Printf("[Worker] %d payments expired\n", len(payments))

	for _, p := range payments {
		event := PaymentExpiredEvent{
			PaymentID: p.ID.String(),
			OrderID:   p.OrderID.String(),
			ExpiredAt: time.Now(),
		}

		err := w.publisher.Publish(ctx, "payment.expired", event)
		if err != nil {
			log.Printf("[Worker] failed publish event for payment %s: %v\n", p.ID, err)

			// NOTE:
			// di production → masukin ke retry / outbox
			continue
		}

		log.Printf("[Worker] event published for payments %s\n", p.ID)
	}
}
