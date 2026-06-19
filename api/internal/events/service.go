package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repository *Repository
}

func NewService(db *pgxpool.Pool) *Service {
	repository := NewRepository(db)
	return &Service{repository: repository}
}

func (s *Service) UpdateOutboxEventsStatus(ctx context.Context, ids []string, status string) error {
	err := s.repository.UpdateOutboxEventsStatus(ctx, ids, status)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) GetUnpublishedEvents(ctx context.Context, limit int) ([]Event, error) {
	// get events from DB
	outboxEvents, err := s.repository.FindUnpublishedAndMarkProcessing(ctx, limit)
	if err != nil {
		return nil, err
	}

	// convert outbox events to Events
	var events []Event
	for _, outEvnt := range outboxEvents {
		payload, err := decodeOutboxPayload(outEvnt.EventType, outEvnt.Payload)
		if err != nil {
			return nil, err
		}

		event := Event{
			ID:        outEvnt.ID,
			Name:      outEvnt.EventType,
			Payload:   payload,
			CreatedAt: outEvnt.CreatedAt,
		}

		events = append(events, event)
	}

	return events, nil
}

func decodeOutboxPayload(eventType string, rawPayload []byte) (any, error) {
	switch eventType {
	case "order.created":
		var payload OrderCreatedPayload
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case "payment.callback.processed":
		var payload PaymentCallbackProcessedPayload
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case "payment.expired":
		var payload PaymentExpiredPayload
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
}
