package events

import (
	"context"
	"errors"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/platform/database"
)

const (
	StatusPending    = "PENDING"
	StatusProcessing = "PROCESSING"
	StatusPublished  = "PUBLISHED"
	StatusFailed     = "FAILED"
)

var ErrOrderNotFound = errors.New("event not found")

type Repository struct {
	db database.DBTX // can be either *pgxpool.Pool or pgx.Tx
}

type OutboxEvent struct {
	ID          string
	EventType   string
	Payload     []byte
	Status      string
	CreatedAt   time.Time
	PublishedAt *time.Time
}

func NewRepository(db database.DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpdateOutboxEventsStatus(ctx context.Context, ids []string, status string) error {
	if len(ids) == 0 {
		return nil
	}

	query := ``

	if status == StatusPublished {
		query = `
		UPDATE outbox
		SET status = $1,
		    published_at = NOW()
		`
	} else {
		query = `
		UPDATE outbox
		SET status = $1
		`
	}

	query += `
	WHERE status = $2
	AND id = ANY($3)
	`

	cmd, err := r.db.Exec(ctx, query, status, StatusProcessing, ids)
	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return ErrOrderNotFound
	}

	return nil
}

func (r *Repository) FindUnpublishedAndMarkProcessing(ctx context.Context, limit int) ([]OutboxEvent, error) {

	query := `
	UPDATE outbox
	SET status = $1
	WHERE id IN (
		SELECT id
		FROM outbox
		WHERE published_at IS NULL
		AND status = $2
		ORDER BY created_at ASC
		LIMIT $3
		FOR UPDATE SKIP LOCKED
	)
	RETURNING id, event_type, payload, status, created_at, published_at
	`

	rows, err := r.db.Query(ctx, query, StatusProcessing, StatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []OutboxEvent
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.Payload,
			&event.Status,
			&event.CreatedAt,
			&event.PublishedAt,
		); err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil

}
