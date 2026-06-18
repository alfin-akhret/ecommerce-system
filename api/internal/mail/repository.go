package mail

import (
	"context"
	"errors"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/platform/database"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct {
	db database.DBTX // can be either *pgxpool.Pool or pgx.Tx
}

func NewEmailRepository(db database.DBTX) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) InsertProcessedMessage(ctx context.Context, eventID string) error {
	query := `
	INSERT INTO processed_messages (message_id, created_at)
	VALUES ($1,$2)
	`

	now := time.Now().UTC()

	_, err := r.db.Exec(ctx, query,
		eventID,
		now,
	)

	return err
}

// duplicate error helper
func IsDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
