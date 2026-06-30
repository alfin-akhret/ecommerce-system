package mail

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alfin-akhret/ecommerce-system/internal/events"
)

const (
	inboxBatchSize = 10
	maxRetryCount  = 5
)

func (s *Service) ProcessInbox(ctx context.Context) error {
	inboxEvents, err := s.inboxRepo.Claim(ctx, inboxBatchSize, consumerName)
	if err != nil {
		return err
	}

	for _, inboxEvent := range inboxEvents {

		if inboxEvent.RetryCount >= maxRetryCount {
			if markFailedErr := s.inboxRepo.MarkFailed(ctx, inboxEvent.ID,
				inboxEvent.Consumer); markFailedErr != nil {
				return markFailedErr
			}
			continue
		}

		if err := s.handleInboxEvent(ctx, inboxEvent); err != nil {
			if markErr := s.inboxRepo.MarkPending(ctx, inboxEvent.ID,
				inboxEvent.RetryCount,
				err,
				inboxEvent.Consumer); markErr != nil {
				return markErr
			}
			continue
		}

		if err := s.inboxRepo.MarkProcessed(ctx, inboxEvent.ID, inboxEvent.Consumer); err != nil {
			return err
		}

	}

	return nil
}

func (s *Service) handleInboxEvent(ctx context.Context, inboxEvent events.InboxEvent) error {

	var event events.Event

	if err := json.Unmarshal(inboxEvent.Payload, &event); err != nil {
		return err
	}

	switch event.Name {
	case events.OrderCreated:
		return s.handleOrderCreated(ctx, event)
	case events.PaymentExpired:
		return s.handlePaymentExpired(ctx, event)
	case events.PaymentCallbackProcessed:
		return s.handlePaymentCallbackProcessed(ctx, event)
	default:
		return fmt.Errorf("unknown event: %s", event.Name)
	}

}

func (s *Service) handleOrderCreated(ctx context.Context, event events.Event) error {

	var payload events.OrderCreatedPayload

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	emailPayload := EmailPayload{
		To:      payload.Email,
		Subject: "Order Confirmation",
		Body: fmt.Sprintf(
			"Your order %s has been created",
			payload.OrderID,
		),
	}

	return s.sendMail(ctx, emailPayload, event.ID)
}

func (s *Service) handlePaymentExpired(ctx context.Context, event events.Event) error {

	var payload events.PaymentExpiredPayload

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	emailPayload := EmailPayload{
		To:      payload.Email,
		Subject: "Payment Expired",
		Body: fmt.Sprintf(
			`Your order information:

		Order ID: %s
		Payment ID: %s
		Order Status: CANCELED
		Payment Status: %s
		`,
			payload.OrderID,
			payload.PaymentID,
			"EXPIRED",
		),
	}

	return s.sendMail(ctx, emailPayload, event.ID)

}

func (s *Service) handlePaymentCallbackProcessed(ctx context.Context, event events.Event) error {

	var payload events.PaymentCallbackProcessedPayload

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	emailPayload := EmailPayload{
		To:      payload.Email,
		Subject: "Payment Status",
		Body: fmt.Sprintf("Your payment status with ID: %s for Order: %s was %s",
			payload.PaymentID,
			payload.OrderID,
			payload.Status),
	}

	return s.sendMail(ctx, emailPayload, event.ID)

}
