package order

type OrderListItem struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	TotalAmount float64 `json:"total_amount"`
	CreatedAt   string  `json:"created_at"`
}

type OrderItemResponse struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	Price     float64 `json:"price"`
	Qty       int     `json:"quantity"`
}

type OrderResponse struct {
	ID          string              `json:"id"`
	UserID      string              `json:"user_id"`
	Status      string              `json:"status"`
	TotalAmount float64             `json:"total_amount"`
	CreatedAt   string              `json:"created_at"`
	Items       []OrderItemResponse `json:"items"`
}

type CheckoutItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type CheckoutRequest struct {
	Items         []CheckoutItem `json:"items"`
	PaymentMethod string         `json:"payment_method"`
}

type CheckoutResponse struct {
	OrderID     string  `json:"order_id"`
	TotalAmount float64 `json:"total_amount"`
	PaymentURL  string  `json:"payment_url"`
	ExpiredAt   string  `json:"expired_at"`
}
