package order

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/platform/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db database.DBTX // can be either *pgxpool.Pool or pgx.Tx
}

func (r *Repository) DeleteIdempotencyKey(ctx context.Context, limit int) ([]DeletedKeys, error) {
	query := `
	DELETE FROM idempotency_keys
	WHERE key IN (
		SELECT key FROM idempotency_keys
		WHERE expired_at < NOW()
		OR status = 'EXPIRED'
		LIMIT $1)
	RETURNING key, user_id, order_id
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DeletedKeys
	for rows.Next() {
		var d DeletedKeys
		if err := rows.Scan(&d.Key, &d.UserID, &d.OrderID); err != nil {
			return nil, err
		}
		result = append(result, d)
	}

	return result, nil

}

func (r *Repository) ExpireIdempotencyKey(ctx context.Context, orderID string) error {
	query := `
	UPDATE idempotency_keys
	SET status = 'EXPIRED'
	WHERE order_id = $1 AND status = 'PENDING'
	`

	cmd, err := r.db.Exec(ctx, query, orderID)
	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return ErrKeyNotFound
	}

	return nil
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
	AND status = 'PENDING'
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

func (r *Repository) GetIdempotencyKey(ctx context.Context, userID uuid.UUID, iKey uuid.UUID) (*IdempotencyKey, error) {

	query := `
	SELECT key, user_id, status, expired_at, response
	FROM idempotency_keys
	WHERE key = $1 AND user_id = $2 AND status <> 'EXPIRED'
	`

	var result IdempotencyKey
	if err := r.db.QueryRow(ctx, query, iKey, userID).Scan(
		&result.Key,
		&result.UserID,
		&result.Status,
		&result.ExpiredAt,
		&result.Response,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	return &result, nil
}

func (r *Repository) SaveIdempotencyKey(ctx context.Context,
	iKey uuid.UUID,
	userID uuid.UUID,
	orderID uuid.UUID,
	status string,
	jsonResponse []byte,
	expiredAt time.Time) error {

	query := `
	INSERT INTO idempotency_keys (key, user_id, order_id, status, response, expired_at)
	VALUES ($1,$2,$3,$4,$5,$6)
	`

	_, err := r.db.Exec(ctx, query,
		iKey,
		userID,
		orderID,
		status,
		jsonResponse,
		expiredAt,
	)

	return err
}
