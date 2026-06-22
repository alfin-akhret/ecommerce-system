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
	StatusDead       = "DEAD"
)

var ErrOrderNotFound = errors.New("event not found")

type Repository struct {
	db database.DBTX // can be either *pgxpool.Pool or pgx.Tx
}

type OutboxEvent struct {
	ID            string
	EventType     string
	Payload       []byte
	Status        string
	RetryCount    int
	CreatedAt     time.Time
	PublishedAt   *time.Time
	NextAttemptAt time.Time
	LockedAt      *time.Time
	LockedBy      *string
	LastError     *string
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
		    published_at = NOW(),
			locked_at = NULL,
			locked_by = NULL
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

func (r *Repository) FindUnpublishedAndMarkProcessing(
	ctx context.Context,
	limit int,
	lockedBy string,
	leaseTimeout time.Duration,
) ([]OutboxEvent, error) {

	query := `
	UPDATE outbox
	SET status = $1,
		locked_at = NOW(),
		locked_by = $2
	WHERE id IN (
		SELECT id
		FROM outbox
		WHERE published_at IS NULL
		AND (
			(status = $3 AND next_attempt_at <= NOW())
			OR
			(status = $4 AND locked_at < NOW() - $5::interval)
		)
		ORDER BY created_at ASC
		LIMIT $6
		FOR UPDATE SKIP LOCKED
	)
	RETURNING 
	id, 
	event_type,
	payload, 
	status, 
	retry_count,
	created_at, 
	published_at,
	next_attempt_at,
	locked_at,
	locked_by,
	last_error
	`

	rows, err := r.db.Query(
		ctx,
		query,
		StatusProcessing,
		lockedBy,
		StatusPending,
		StatusProcessing,
		leaseTimeout.String(),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []OutboxEvent
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.Payload,
			&event.Status,
			&event.RetryCount,
			&event.CreatedAt,
			&event.PublishedAt,
			&event.NextAttemptAt,
			&event.LockedAt,
			&event.LockedBy,
			&event.LastError,
		); err != nil {
			return nil, err
		}

		result = append(result, event)
	}

	return result, rows.Err()

}

func (r *Repository) MarkPublished(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	query := `
	UPDATE outbox
	SET status = $1,
		published_at = NOW(),
		locked_at = NULL,
		locked_by = NULL,
		last_error = NULL
	WHERE status = $2
	AND id = ANY($3)
	`

	_, err := r.db.Exec(ctx, query, StatusPublished, StatusProcessing, ids)
	return err
}

// mark published yang gagal
func (r *Repository) MarkPublishFailed(
	ctx context.Context,
	id string,
	errMessage string,
	maxRetry int,
	backoff time.Duration,
) error {

	query := `
	UPDATE outbox
	SET status = CASE
			WHEN retry_count + 1 >= $2 THEN $3
			ELSE $4
		END,
			retry_count = retry_count + 1,
			next_attempt_at = CASE
			WHEN retry_count + 1 >= $2 THEN next_attempt_at
			ELSE $5
		END,
			last_error = $6,
			locked_at = NULL,
			locked_by = NULL
	WHERE id = $1
		AND status = $7 
	`

	NextAttemptAt := time.Now().UTC().Add(backoff)

	_, err := r.db.Exec(
		ctx,
		query,
		id,
		maxRetry,
		StatusDead,
		StatusPending,
		NextAttemptAt,
		errMessage,
		StatusProcessing,
	)

	return err

}
