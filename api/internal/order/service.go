package order

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/contracts"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrOrderNotFound = errors.New("order not found")

type Service struct {
	db             *pgxpool.Pool
	repo           *Repository
	productUpdater contracts.ProductUpdater
	paymentUpdater contracts.PaymentUpdater
}

func NewService(db *pgxpool.Pool, productUpdater contracts.ProductUpdater, paymentUpdater contracts.PaymentUpdater) *Service {

	repo := NewOrderRepository(db)

	return &Service{
		db:             db,
		repo:           repo,
		productUpdater: productUpdater,
		paymentUpdater: paymentUpdater,
	}
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
			CreatedAt:   o.CreatedAt.UTC().Format(time.RFC3339),
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
		CreatedAt:   order.CreatedAt.UTC().Format(time.RFC3339),
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
		product, err := s.productUpdater.GetProductByIDWithTx(ctx, tx, item.ProductID)
		if err != nil {
			return nil, err
		}

		if err := s.productUpdater.ReserveStockWithTx(ctx, tx, item.ProductID, item.Quantity); err != nil {
			return nil, err
		}

		itemQty := item.Quantity
		subtotal := product.GetPrice() * float64(itemQty)
		total += subtotal

		// parse product ID
		OrderItem := &OrderItem{
			ID:        uuid.New(),
			OrderID:   order.ID,
			ProductID: product.GetID(),
			Price:     product.GetPrice(),
			Qty:       itemQty,
		}

		orderItems = append(orderItems, OrderItem)
	}

	order.TotalAmount = total

	if err := orderRepo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	paymentResp, err := s.paymentUpdater.CreatePaymentWithTx(ctx, tx, order.ID.String(), total, req.PaymentMethod)
	if err != nil {
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
		PaymentURL:  paymentResp.PaymentURL,
		ExpiredAt:   helper.FormatOptionalTime(paymentResp.ExpiredAt),
	}, nil
}

func (s *Service) UpdateOrderStatusWithTx(ctx context.Context, tx pgx.Tx, orderID string, status string) error {
	repo := s.repo.WithTx(tx)
	return repo.UpdateOrderStatus(ctx, orderID, status)
}

func (s *Service) UpdateStatus(ctx context.Context, orderID string, status string) error {
	return s.repo.UpdateOrderStatus(ctx, orderID, status)
}

func (s *Service) ConfirmOrderStockWithTx(ctx context.Context, tx pgx.Tx, orderID string) error {
	items, err := s.repo.ListOrderItems(ctx, orderID)
	if err != nil {
		return err
	}

	for _, item := range items {
		if err := s.productUpdater.ConfirmStockWithTx(ctx, tx, item.ProductID.String(), item.Qty); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) ReleaseOrderStockWithTx(ctx context.Context, tx pgx.Tx, orderID string) error {
	repo := s.repo.WithTx(tx)

	items, err := repo.ListOrderItems(ctx, orderID)
	if err != nil {
		return err
	}

	for _, item := range items {
		if err := s.productUpdater.ReleaseStockWithTx(
			ctx,
			tx,
			item.ProductID.String(),
			item.Qty,
		); err != nil {
			return err
		}
	}

	return nil
}

// subscribe to topic: "payment.expired"
func (s *Service) CancelOrder(ctx context.Context, orderID string) {
	log.Println("[Order] cancel order:", orderID)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("[Order Service] cancel order error: %v\n", err)
		return
	}
	defer tx.Rollback(ctx)

	if err := s.UpdateOrderStatusWithTx(ctx, tx, orderID, "CANCELLED"); err != nil {
		log.Printf("[Order Service] cancel order error: %v\n", err)
		return
	}

	if err := s.ReleaseOrderStockWithTx(ctx, tx, orderID); err != nil {
		log.Printf("[Order Service] cancel order error: %v\n", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("[Order Service] cancel order commit error: %v\n", err)
		return
	}

}
