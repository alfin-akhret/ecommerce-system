package events

import (
	"context"
	"errors"

	"github.com/alfin-akhret/ecommerce-system/internal/platform/database"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
)

type InboxRepository struct {
	db DB // pake DB interface biar bisa micro transaksi di dalam repo seprti pd function Claim
}

type InboxRepositoryManager interface {
	Save(ctx context.Context, event InboxEvent) error
	GetPending(ctx context.Context, limit int) ([]InboxEvent, error)
	MarkProcessed(ctx context.Context, id string) error
}

func NewInboxRepository(db database.DBTX) *InboxRepository {
	return &InboxRepository{db: db}
}

// Claim untuk mencegah multiple worker berebut event.
// menggunakan FOR UPDATE SKIP LOCKED.
// agar SKIP LOCKED efektif maka harus dilakukan di dalam transaksi.
// transaksi hanya dilakukan untuk proses claim.
// alur:
// worker A -> begin tx -> claim 1-100 -> update status to processing,
// selama proses claiming, worker B tidak bisa select atau update event 1-100,
// setelah proses claim oleh worker A, worker B tidak akan select event 1-100 karena
// statusnya sudah berubah jadi PROCESSING.
// kenapa kita butuh transaksi internal seperti ini?
// karena transaksi semacam ini adalah bagian dari proses implementasi query, bukan bagian
// dari business logic. dengan kata lain transaksi disini bukan untuk tujuan atomic tapi untuk
// tujuan locking cepat agar tidak ada race condition antar worker.
func (ir *InboxRepository) Claim(ctx context.Context, limit int) ([]InboxEvent, error) {

	query := `
	WITH claimed AS (
		SELECT id
		FROM inbox_events
		WHERE status = $1
		ORDER BY received_at
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	)
	UPDATE inbox_events
	SET status = $3
	FROM claimed
	WHERE inbox_events.id = claimed.id
	RETURNING
		inbox_events.id,
		inbox_events.event_type,
		inbox_events.playload,
		inbox_events.retry_count,
		inbox_events.created_at;
	`

	tx, err := ir.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, query, StatusPending, limit, StatusProcessing)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inboxEvents []InboxEvent
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
		inboxEvents = append(inboxEvents, inbox_event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return inboxEvents, nil
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
		return errors.New("event not found")
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

	var inboxEvents []InboxEvent
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
		inboxEvents = append(inboxEvents, inbox_event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return inboxEvents, nil
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
