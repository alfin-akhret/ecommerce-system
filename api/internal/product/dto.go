package product

import "github.com/google/uuid"

type CreateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

type ProductResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type ProductListItem struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Price          float64 `json:"price"`
	AvailableStock int     `json:"available_stock"`
}

func toProduct(r CreateProductRequest) *Product {
	return &Product{
		ID:          uuid.New(),
		Name:        r.Name,
		Description: r.Description,
		Price:       r.Price,
	}
}

func toProductResponse(p *Product) *ProductResponse {
	return &ProductResponse{
		ID:          p.ID.String(),
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
	}
}
