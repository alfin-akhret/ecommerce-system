package product

import (
	"context"

	"github.com/alfin-akhret/ecommerce-system/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db database.DBTX // can be either *pgxpool.Pool or pgx.Tx
}

func NewProductRepository(db database.DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{
		db: tx,
	}
}

func (r *Repository) CreateProduct(ctx context.Context, p *Product) error {
	query := "INSERT INTO products (id, name, description, price) VALUES ($1, $2, $3, $4)"
	_, err := r.db.Exec(ctx, query,
		p.ID, p.Name, p.Description, p.Price)
	return err
}

func (r *Repository) CreateInventory(ctx context.Context, inv *Inventory) error {
	query := "INSERT INTO product_inventory (product_id, stock, reserved) VALUES ($1, $2, $3)"
	_, err := r.db.Exec(ctx, query,
		inv.ProductID, inv.Stock, inv.Reserved)
	return err
}

func (r *Repository) ListProducts(ctx context.Context) ([]ProductListItem, error) {
	query := `SELECT p.id, p.name, p.price, (i.stock - i.reserved) AS available_stock
			  FROM products p
			  JOIN product_inventory i ON p.id = i.product_id`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []ProductListItem
	for rows.Next() {
		var p ProductListItem
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.AvailableStock); err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, nil
}

func (r *Repository) GetProductByID(ctx context.Context, id string) (*ProductDetailResponse, error) {
	query := `SELECT
		p.id,
		p.name,
		p.description,
		p.price,
		i.stock,
		i.reserved,
		(i.stock - i.reserved) AS available
	FROM products p
	JOIN product_inventory i
		ON p.id = i.product_id
	WHERE p.id = $1`

	var p ProductDetailResponse

	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.Price,
		&p.Stock,
		&p.Reserved,
		&p.Available,
	)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *Repository) UpdateStock(ctx context.Context, productID string, qty int) error {

	query := `
	UPDATE product_inventory
	SET stock = stock + $1,
	    updated_at = now()
	WHERE product_id = $2
	`

	_, err := r.db.Exec(ctx, query, qty, productID)
	return err
}

func (r *Repository) GetInventoryForUpdate(ctx context.Context, productID string) (*Inventory, error) {
	// FOR UPDATE --> lock row until transaction finished
	query := `
	SELECT product_id, stock, reserved
	FROM product_inventory
	WHERE product_id = $1
	FOR UPDATE
	`

	var inv Inventory
	err := r.db.QueryRow(ctx, query, productID).Scan(
		&inv.ProductID,
		&inv.Stock,
		&inv.Reserved,
	)
	if err != nil {
		return nil, err
	}

	return &inv, nil
}

func (r *Repository) UpdateReserved(ctx context.Context, productID string, qty int) error {
	query := `
	UPDATE product_inventory
	SET reserved = reserved + $1, updated_at = now()
	WHERE product_id = $2
	`

	_, err := r.db.Exec(ctx, query, qty, productID)

	return err
}

func (r *Repository) ReleasedReserved(ctx context.Context, productID string, qty int) error {
	query := `
	UPDATE product_inventory
	SET reserved = reserved - $1,
	    updated_at = now()
	WHERE product_id = $2
	AND reserved >= $1
	`
	_, err := r.db.Exec(ctx, query, qty, productID)

	return err
}

func (r *Repository) ConfirmStock(ctx context.Context, productID string, qty int) error {

	query := `
	UPDATE product_inventory
	SET stock = stock - $1,
	    reserved = reserved - $1,
	    updated_at = now()
	WHERE product_id = $2
	`

	_, err := r.db.Exec(ctx, query, qty, productID)

	return err
}
