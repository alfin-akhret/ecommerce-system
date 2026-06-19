package events

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repository *Repository
}

func NewService(db *pgxpool.Pool) *Service {
	repository := NewRepository(db)
	return &Service{repository: repository}
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
		event := Event{
			ID:        outEvnt.ID,
			Name:      outEvnt.EventType,
			Payload:   outEvnt.Payload,
			CreatedAt: outEvnt.CreatedAt,
		}

		events = append(events, event)
	}

	return events, nil
}
