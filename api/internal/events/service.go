package events

import (
	"context"
	"time"

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

func (s *Service) GetUnpublishedEvents(ctx context.Context, limit int, lockedBy string, leaseTimeout time.Duration) ([]Event, error) {
	// get events from DB
	outboxEvents, err := s.repository.FindUnpublishedAndMarkProcessing(ctx, limit, lockedBy, leaseTimeout)
	if err != nil {
		return nil, err
	}

	// convert outbox events to Events
	var events []Event
	for _, outEvnt := range outboxEvents {
		event := Event{
			ID:         outEvnt.ID,
			Name:       outEvnt.EventType,
			Payload:    outEvnt.Payload,
			CreatedAt:  outEvnt.CreatedAt,
			RetryCount: outEvnt.RetryCount,
		}

		events = append(events, event)
	}

	return events, nil
}

func (s *Service) MarkPublished(ctx context.Context, publishedIds []string) error {
	err := s.repository.MarkPublished(ctx, publishedIds)
	return err
}

func (s *Service) MarkPublishFailed(ctx context.Context, eventID string, errString string, maxRetry int, backoff time.Duration) error {
	err := s.repository.MarkPublishFailed(ctx, eventID, errString, maxRetry, backoff)
	return err
}
