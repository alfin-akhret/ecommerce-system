package contracts

import (
	"context"

	"github.com/google/uuid"
)

type CartItem struct {
	ProductID uuid.UUID
	Qty       int
}

type CartManager interface {
	GetCart(ctx context.Context, ownerID uuid.UUID) ([]CartItem, error)
	DeleteCart(ctx context.Context, ownerID uuid.UUID) (string, error)
}
