package order

import (
	"context"
	"errors"

	"github.com/alfin-akhret/ecommerce-system/internal/product"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidQty = errors.New("invalid quantity")

type Service struct {
	db   *pgxpool.Pool // for queries that require transactions, we create a new repository with the transaction as DBTX
	repo *Repository   // for simple queries that don't require transactions, we can use the repository with the main DB connection
}

func NewService(db *pgxpool.Pool) *Service {

	repo := NewOrderRepository(db)

	return &Service{
		db:   db,
		repo: repo,
	}
}

func (s *Service) CreateOrder(ctx context.Context, userID string, req CreateOrderRequest) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	orderRepo := NewOrderRepository(tx)
	productRepo := product.NewProductRepository(tx)

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	order := &Order{
		ID:     uuid.New(),
		UserID: userUUID,
		Status: "pending_payment",
	}

	var (
		total      float64
		orderItems []*OrderItem
	)

	for _, item := range req.Items {
		if item.Qty <= 0 {
			return ErrInvalidQty
		}

		// get products
		p, err := productRepo.GetProductByID(ctx, item.ProductID)
		if err != nil {
			return err
		}

		inv, err := productRepo.GetInventoryForUpdate(ctx, item.ProductID)
		if err != nil {
			return err
		}

		available := inv.Stock - inv.Reserved
		if available < item.Qty {
			return product.ErrNotEnoughStock
		}

		// reserve stock
		if err := productRepo.UpdateReserved(ctx, item.ProductID, item.Qty); err != nil {
			return err
		}

		productUUID, err := uuid.Parse(p.ID)
		if err != nil {
			return err
		}

		orderItem := &OrderItem{
			ID:        uuid.New(),
			OrderID:   order.ID,
			ProductID: productUUID,
			Price:     p.Price,
			Qty:       item.Qty,
		}

		orderItems = append(orderItems, orderItem)
		total += p.Price * float64(item.Qty)
	}

	order.TotalAmount = total

	if err := orderRepo.CreateOrder(ctx, order); err != nil {
		return err
	}

	for _, orderItem := range orderItems {
		if err := orderRepo.CreateOrderItem(ctx, orderItem); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)

}
