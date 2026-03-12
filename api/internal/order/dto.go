package order

type CreateOrderRequest struct {
	Items []CreateOrderRequest `json:"items"`
}

type OrderItemRequest struct {
	ProductID string `json:"product_id"`
	Qty       int    `json:"qty"`
}
