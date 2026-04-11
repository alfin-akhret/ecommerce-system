package payment

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID            uuid.UUID  `json:"id"`
	OrderID       uuid.UUID  `json:"order_id"`
	Amount        int64      `json:"amount"`
	Status        string     `json:"status"`
	PaymentMethod string     `json:"payment_method"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
	ExpiredAt     *time.Time `json:"expired_at,omitempty"`
}

type ExpiredPayment struct {
	ID      uuid.UUID
	OrderID uuid.UUID
}

type ReconData struct {
	PaymentID uuid.UUID
	Status    string
	Remark    string
	CreatedAt time.Time
}
