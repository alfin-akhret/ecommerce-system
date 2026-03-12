package product

import "github.com/google/uuid"

type Product struct {
	ID          uuid.UUID
	Name        string
	Description string
	Price       float64
}

type Inventory struct {
	ProductID uuid.UUID
	Stock     int
	Reserved  int
}
