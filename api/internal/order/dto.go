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

type CreateOrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type CreateOrderRequest struct {
	Items         []CreateOrderItem `json:"items"`
	PaymentMethod string            `json:"payment_method"`
}

type CreateOrderResponse struct {
	OrderID     string  `json:"order_id"`
	TotalAmount float64 `json:"total_amount"`
	PaymentURL  string  `json:"payment_url"`
	ExpiredAt   *string `json:"expired_at"`
}

// CHECKOUT DTO
type CheckoutResponse struct {
	Items         []CheckoutItem        `json:"items"`
	TotalAmount   float64               `json:"total_amount"`
	GrandTotal    float64               `json:"grand_total"`
	PaymentMethod string                `json:"payment_method"`
	Shipping      *ShippingInfoResponse `json:"shipping"`
	Promo         *PromoInfoResponse    `json:"promo"`
}

type CheckoutItem struct {
	ProductID string `json:"product_id"`
	// Name      string  `json:"name"`
	Price float64 `json:"price"`
	Qty   int     `json:"qty"`
}

type ShippingInfoResponse struct {
	Method  string  `json:"method"`
	Address string  `json:"address"`
	Cost    float64 `json:"cost"`
}

type PromoInfoResponse struct {
	Code   string  `json:"code"`
	Amount float64 `json:"amount"`
}
