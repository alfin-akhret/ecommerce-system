package contracts

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type OrderSnapshot struct {
	ID          string
	UserID      string
	Status      string
	TotalAmount float64
	CreatedAt   time.Time
}

type OrderUpdater interface {
	UpdateOrderStatusWithTx(ctx context.Context, tx pgx.Tx, orderID string, status string) error
	ConfirmOrderStockWithTx(ctx context.Context, tx pgx.Tx, orderID string) error
	ReleaseOrderStockWithTx(ctx context.Context, tx pgx.Tx, orderID string) error
	GetOrderWithTx(ctx context.Context, tx pgx.Tx, orderID string) (*OrderSnapshot, error)
}
