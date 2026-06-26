package events

import (
	"context"

	"github.com/alfin-akhret/ecommerce-system/internal/platform/database"
)

type InboxRepository struct {
	db database.DBTX // can be either *pgxpool.Pool or pgx.Tx
}

type InboxRepositoryManager interface {
	Save(ctx context.Context, event InboxEvent) error
}

func NewInboxRepository(db database.DBTX) *InboxRepository {
	return &InboxRepository{db: db}
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

	return err
}
