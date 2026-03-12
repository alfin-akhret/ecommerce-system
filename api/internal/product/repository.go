package product

import (
	"context"

	"github.com/alfin-akhret/ecommerce-system/internal/platform/database"
)

type Repository interface {
	CreateProduct(ctx context.Context, p *Product) error
	CreateInventory(ctx context.Context, inv *Inventory) error
}

type repository struct {
	db database.DBTX
}

func NewProductRepository(db database.DBTX) Repository {
	return &repository{db: db}
}

func (r *repository) CreateProduct(ctx context.Context, p *Product) error {
	query := "INSERT INTO products (id, name, description, price) VALUES ($1, $2, $3, $4)"
	_, err := r.db.Exec(ctx, query,
		p.ID, p.Name, p.Description, p.Price)
	return err
}

func (r *repository) CreateInventory(ctx context.Context, inv *Inventory) error {
	query := "INSERT INTO product_inventory (product_id, stock, reserved) VALUES ($1, $2, $3)"
	_, err := r.db.Exec(ctx, query,
		inv.ProductID, inv.Stock, inv.Reserved)
	return err
}
