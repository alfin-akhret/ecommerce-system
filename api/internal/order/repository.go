package order

import (
	"context"

	"github.com/alfin-akhret/ecommerce-system/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db database.DBTX // can be either *pgxpool.Pool or pgx.Tx
}

func NewOrderRepository(db database.DBTX) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{
		db: tx,
	}
}

func (r *Repository) CreateOrder(ctx context.Context, o *Order) error {
	query := `
	INSERT INTO orders (id, user_id, status, total_amount)
	VALUES ($1,$2,$3,$4)
	`

	_, err := r.db.Exec(ctx, query,
		o.ID,
		o.UserID,
		o.Status,
		o.TotalAmount,
	)

	return err
}

func (r *Repository) CreateOrderItem(ctx context.Context, item *OrderItem) error {
	query := `
	INSERT INTO order_items (id, order_id, product_id, price, quantity)
	VALUES ($1,$2,$3,$4,$5)
	`

	_, err := r.db.Exec(ctx, query,
		item.ID,
		item.OrderID,
		item.ProductID,
		item.Price,
		item.Qty,
	)

	return err
}
