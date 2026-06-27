package events

import (
	"context"
	"errors"

	"github.com/alfin-akhret/ecommerce-system/internal/platform/database"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
)

type InboxRepository struct {
	db database.DBTX // can be either *pgxpool.Pool or pgx.Tx
}

type InboxRepositoryManager interface {
	Save(ctx context.Context, event InboxEvent) error
	GetPending(ctx context.Context, limit int) ([]InboxEvent, error)
	MarkProcessed(ctx context.Context, id string) error
}

func NewInboxRepository(db database.DBTX) *InboxRepository {
	return &InboxRepository{db: db}
}

func (ir *InboxRepository) MarkProcessed(ctx context.Context, id string) error {
	query := `
	UPDATE inbox_events
	SET 
		status = 'PROCESSED',
		processed_at = NOW()
	WHERE id = $1;
	`

	cmd, err := ir.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return errors.New("Event not found.")
	}

	return nil
}

func (ir *InboxRepository) GetPending(ctx context.Context, limit int) ([]InboxEvent, error) {
	query := `
	SELECT
		id,
		event_type,
		payload,
		created_at,
		retry_count
	FROM inbox_events
	WHERE status = 'PENDING'
	ORDER BY received_at
	LIMIT $1;
	`

	rows, err := ir.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inbox_events []InboxEvent
	for rows.Next() {
		var inbox_event InboxEvent
		if err := rows.Scan(
			&inbox_event.ID,
			&inbox_event.EventType,
			&inbox_event.Payload,
			&inbox_event.CreatedAt,
			&inbox_event.RetryCount,
		); err != nil {
			return nil, err
		}
		inbox_events = append(inbox_events, inbox_event)
	}

	return inbox_events, nil
}

func (ir *InboxRepository) Save(ctx context.Context, event InboxEvent) error {

	query := `
	INSERT INTO inbox_events (id, event_type, payload, status, created_at)
	values ($1, $2, $3, $4, $5)
	`

	_, err := ir.db.Exec(ctx, query,
		event.ID,
		event.EventType,
		event.Payload,
		event.Status,
		event.CreatedAt,
	)
	if err != nil {
		if helper.IsDuplicateKeyError(err) {
			return ErrDuplicateInboxEvent
		}
		return err
	}

	return nil
}
