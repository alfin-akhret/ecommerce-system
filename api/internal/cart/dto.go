package cart

import (
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/google/uuid"
)

type AddCartItemRequest struct {
	ProductID string `json:"product_id"`
	Qty       int    `json:"qty"`
}

func parseCartItemRequest(r AddCartItemRequest) (uuid.UUID, int, error) {
	pid, err := uuid.Parse(r.ProductID)
	if err != nil {
		return uuid.Nil, 0, err

	}

	return pid, r.Qty, nil
}

type CartItemResponse struct {
	ProductID string  `json:"product_id"`
	Qty       int     `json:"qty"`
	Price     float64 `json:"price"`
}

func toCartItemResponse(item *CartItem) CartItemResponse {
	return CartItemResponse{
		ProductID: item.ProductID.String(),
		Qty:       item.Qty,
		Price:     helper.ToFloat(item.Price),
	}
}

type GetCartResponse struct {
	Items      []CartItemResponse `json:"items"`
	TotalPrice float64            `json:"total_price"`
}
