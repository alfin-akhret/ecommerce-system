package contracts

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ProductView interface {
	GetID() uuid.UUID
	GetPrice() int64
}

type ProductManager interface {
	GetProductByIDWithTx(ctx context.Context, tx pgx.Tx, productID string) (ProductView, error)
	ReserveStockWithTx(ctx context.Context, tx pgx.Tx, productID string, qty int) error
	ReleaseStockWithTx(ctx context.Context, tx pgx.Tx, productID string, qty int) error
	ConfirmStockWithTx(ctx context.Context, tx pgx.Tx, productID string, qty int) error
	GetProductPrice(ctx context.Context, productID string) (ProductView, error)
}
