package payment

import (
	"context"
	"errors"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

var ErrPaymentNotFound = errors.New("payment not found")

type Repository struct {
	db database.DBTX
}

func NewRepository(db database.DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{db: tx}
}

func (r *Repository) CreatePayment(ctx context.Context, p *Payment) error {
	query := `
	INSERT INTO payments (id, order_id, amount, status, payment_method, paid_at, created_at, updated_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`

	_, err := r.db.Exec(ctx, query,
		p.ID,
		p.OrderID,
		p.Amount,
		p.Status,
		p.PaymentMethod,
		p.PaidAt,
		p.CreatedAt,
		p.UpdatedAt,
	)

	return err
}

// update payment status, optionally set PaidAt
func (r *Repository) UpdateStatus(ctx context.Context, paymentID string, status string, paidAt *time.Time) error {
	query := `
	UPDATE payments
	SET status = $1, paid_at = $2, updated_at = now()
	WHERE id = $3
	`

	cmd, err := r.db.Exec(ctx, query, status, paidAt, paymentID)
	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return ErrPaymentNotFound
	}

	return nil
}

// get payment by id
func (r *Repository) GetByID(ctx context.Context, paymentID string) (*Payment, error) {
	query := `
	SELECT id, order_id, amount, status, payment_method, paid_at, created_at, updated_at
	FROM payments
	WHERE id = $1
	`

	var p Payment
	err := r.db.QueryRow(ctx, query, paymentID).Scan(
		&p.ID,
		&p.OrderID,
		&p.Amount,
		&p.Status,
		&p.PaymentMethod,
		&p.PaidAt,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}

	return &p, nil
}

// get payment by orderID
func (r *Repository) GetByOrderID(ctx context.Context, orderID string) (*Payment, error) {
	query := `
	SELECT id, order_id, amount, status, payment_method, paid_at, created_at, updated_at
	FROM payments
	WHERE order_id = $1
	`
	var p Payment
	err := r.db.QueryRow(ctx, query, orderID).Scan(
		&p.ID,
		&p.OrderID,
		&p.Amount,
		&p.Status,
		&p.PaymentMethod,
		&p.PaidAt,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &p, nil
}
