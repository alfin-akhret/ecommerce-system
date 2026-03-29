package contracts

import (
	"context"

	"github.com/google/uuid"
)

type CartItem struct {
	ProductID uuid.UUID
	Qty       int
}

type CartGetter interface {
	GetCart(ctx context.Context, ownerID uuid.UUID) ([]CartItem, error)
}
