package contracts

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type PaymentCreateResult struct {
	ID         string
	PaymentURL string
}

type PaymentUpdater interface {
	CreatePaymentWithTx(ctx context.Context, tx pgx.Tx, orderID string, amount float64, method string) (*PaymentCreateResult, error)
}
