package order

import (
	"context"
	"errors"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/product"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidQty = errors.New("invalid quantity")
var ErrOrderNotFound = errors.New("order not found")

type Service struct {
	db             *pgxpool.Pool // for queries that require transactions, we create a new repository with the transaction as DBTX
	repo           *Repository   // for simple queries that don't require transactions, we can use the repository with the main DB connection
	productService *product.Service
}

func NewService(db *pgxpool.Pool, productService *product.Service) *Service {

	repo := NewOrderRepository(db)

	return &Service{
		db:             db,
		repo:           repo,
		productService: productService,
	}
}

func (s *Service) CreateOrder(ctx context.Context, userID string, req CreateOrderRequest) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	orderRepo := NewOrderRepository(tx)

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	order := &Order{
		ID:     uuid.New(),
		UserID: userUUID,
		Status: "PENDING_PAYMENT",
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
		p, err := s.productService.GetProductByIDWithTx(ctx, tx, item.ProductID)
		if err != nil {
			return err
		}

		// reserve stock
		if err := s.productService.ReserveInventoryWithTx(ctx, tx, item.ProductID, item.Qty); err != nil {
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

func (s *Service) ListOrders(ctx context.Context, userID string) ([]OrderListItem, error) {
	orders, err := s.repo.ListOrdersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]OrderListItem, 0, len(orders))
	for _, o := range orders {
		items = append(items, OrderListItem{
			ID:          o.ID.String(),
			Status:      o.Status,
			TotalAmount: o.TotalAmount,
			CreatedAt:   o.CreatedAt.Format(time.RFC3339),
		})
	}

	return items, nil
}

func (s *Service) GetOrder(ctx context.Context, userID string, orderID string) (*OrderResponse, error) {
	order, err := s.repo.GetOrderByID(ctx, userID, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	orderItems, err := s.repo.ListOrderItems(ctx, orderID)
	if err != nil {
		return nil, err
	}

	items := make([]OrderItemResponse, 0, len(orderItems))
	for _, item := range orderItems {
		items = append(items, OrderItemResponse{
			ID:        item.ID.String(),
			ProductID: item.ProductID.String(),
			Price:     item.Price,
			Qty:       item.Qty,
		})
	}

	return &OrderResponse{
		ID:          order.ID.String(),
		UserID:      order.UserID.String(),
		Status:      order.Status,
		TotalAmount: order.TotalAmount,
		CreatedAt:   order.CreatedAt.Format(time.RFC3339),
		Items:       items,
	}, nil
}

func (s *Service) Checkout(ctx context.Context, userID string, req CheckoutRequest) (*CheckoutResponse, error) {
	// begin transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// create orderRepo with transaction
	orderRepo := s.repo.WithTx(tx)

	// parse userID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	// create new order object
	order := &Order{
		ID:     uuid.New(),
		UserID: userUUID,
		Status: "PENDING",
	}

	var total float64
	var orderItems []*OrderItem

	for _, item := range req.Items {
		product, err := s.productService.GetProductByIDWithTx(ctx, tx, item.ProductID)
		if err != nil {
			return nil, err
		}

		if err := s.productService.ReserveInventoryWithTx(ctx, tx, item.ProductID, item.Quantity); err != nil {
			return nil, err
		}

		itemQty := item.Quantity
		subtotal := product.Price * float64(itemQty)
		total += subtotal

		// parse product ID
		productUUID, err := uuid.Parse(product.ID)
		if err != nil {
			return nil, err
		}

		OrderItem := &OrderItem{
			ID:        uuid.New(),
			OrderID:   order.ID,
			ProductID: productUUID,
			Price:     product.Price,
			Qty:       itemQty,
		}

		orderItems = append(orderItems, OrderItem)
	}

	order.TotalAmount = total

	if err := orderRepo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	for _, orderItem := range orderItems {
		if err := orderRepo.CreateOrderItem(ctx, orderItem); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &CheckoutResponse{
		OrderID:     order.ID.String(),
		TotalAmount: total,
	}, nil
}
