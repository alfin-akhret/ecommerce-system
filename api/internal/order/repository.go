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

func (r *Repository) ListOrdersByUser(ctx context.Context, userID string) ([]Order, error) {
	query := `
	SELECT id, user_id, status, total_amount, created_at
	FROM orders
	WHERE user_id = $1
	ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(
			&o.ID,
			&o.UserID,
			&o.Status,
			&o.TotalAmount,
			&o.CreatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}

	return orders, nil
}

func (r *Repository) GetOrderByID(ctx context.Context, userID string, orderID string) (*Order, error) {
	query := `
	SELECT id, user_id, status, total_amount, created_at
	FROM orders
	WHERE id = $1 AND user_id = $2
	`

	var o Order
	if err := r.db.QueryRow(ctx, query, orderID, userID).Scan(
		&o.ID,
		&o.UserID,
		&o.Status,
		&o.TotalAmount,
		&o.CreatedAt,
	); err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *Repository) UpdateOrderStatus(ctx context.Context, orderID string, status string) error {
	query := `
	UPDATE orders
	SET status = $1
	WHERE id = $2
	`

	cmd, err := r.db.Exec(ctx, query, status, orderID)
	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return ErrOrderNotFound
	}

	return nil
}

func (r *Repository) ListOrderItems(ctx context.Context, orderID string) ([]OrderItem, error) {
	query := `
	SELECT id, order_id, product_id, price, quantity, created_at
	FROM order_items
	WHERE order_id = $1
	ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.Price,
			&item.Qty,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}
