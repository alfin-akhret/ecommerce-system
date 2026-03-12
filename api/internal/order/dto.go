package order

type CreateOrderRequest struct {
	Items []OrderItemRequest `json:"items"`
}

type OrderItemRequest struct {
	ProductID string `json:"product_id"`
	Qty       int    `json:"qty"`
}

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
	Qty       int     `json:"qty"`
}

type OrderResponse struct {
	ID          string              `json:"id"`
	UserID      string              `json:"user_id"`
	Status      string              `json:"status"`
	TotalAmount float64             `json:"total_amount"`
	CreatedAt   string              `json:"created_at"`
	Items       []OrderItemResponse `json:"items"`
}
