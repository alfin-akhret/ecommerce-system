package order

import (
	"time"

	"github.com/google/uuid"
)

const (
	OrderStatusPending string = "PENDING"
	OrderStatusPaid    string = "PAID"
	OrderStatusFailed  string = "FAILED"
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

type IdempotencyKey struct {
	Key       uuid.UUID
	UserID    uuid.UUID
	Status    string
	ExpiredAt *time.Time
	Response  string
}
