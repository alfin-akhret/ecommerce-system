package product

import "github.com/google/uuid"

type Product struct {
	ID          uuid.UUID
	Name        string
	Description string
	Price       float64
}

func (p *Product) GetID() uuid.UUID {
	return p.ID
}

func (p *Product) GetPrice() float64 {
	return p.Price
}

type Inventory struct {
	ProductID uuid.UUID
	Stock     int
	Reserved  int
}
