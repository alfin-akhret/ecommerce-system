package payment

import "time"

type CreatePaymentRequest struct {
	OrderID       string  `json:"order_id"`
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"payment_method"`
}

type CreatePaymentResponse struct {
	ID            string  `json:"id"`
	OrderID       string  `json:"order_id"`
	Status        string  `json:"status"`
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"payment_method"`
	PaymentURL    string  `json:"payment_url,omitempty"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type UpdatePaymentStatusRequest struct {
	Status string `json:"status"`
}
