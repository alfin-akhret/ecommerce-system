package order

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Status      string
	TotalAmount int64
	CreatedAt   time.Time
}

type OrderItem struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	ProductID uuid.UUID
	Price     int64
	Qty       int
	CreatedAt time.Time
}

type Cart struct {
	Items       []CartItem
	TotalAmount int64
}

type CartItem struct {
	ProductID uuid.UUID
	Price     int64
	Qty       int
}

type ShippingInfo struct {
	Method  string
	Address string
	Cost    int64
}

type PromoInfo struct {
	Code   uuid.UUID
	Amount int64
}
