package contracts

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type PaymentCreateResult struct {
	ID         string
	PaymentURL string
	ExpiredAt  *time.Time
}

type PaymentUpdater interface {
	CreatePaymentWithTx(ctx context.Context, tx pgx.Tx, orderID string, amount float64, method string) (*PaymentCreateResult, error)
}
