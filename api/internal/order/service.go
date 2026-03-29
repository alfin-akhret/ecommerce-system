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
	db      *pgxpool.Pool
	repo    *Repository
	product contracts.ProductManager
	payment contracts.PaymentManager
	cart    contracts.CartManager
}

func NewService(db *pgxpool.Pool,
	product contracts.ProductManager,
	payment contracts.PaymentManager,
	cart contracts.CartManager) *Service {

	repo := NewOrderRepository(db)

	return &Service{
		db:      db,
		repo:    repo,
		product: product,
		payment: payment,
		cart:    cart,
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
			TotalAmount: helper.ToFloat(o.TotalAmount),
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
			Price:     helper.ToFloat(item.Price),
			Qty:       item.Qty,
		})
	}

	return &OrderResponse{
		ID:          order.ID.String(),
		UserID:      order.UserID.String(),
		Status:      order.Status,
		TotalAmount: helper.ToFloat(order.TotalAmount),
		CreatedAt:   order.CreatedAt.UTC().Format(time.RFC3339),
		Items:       items,
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
		if err := s.product.ConfirmStockWithTx(ctx, tx, item.ProductID.String(), item.Qty); err != nil {
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
		if err := s.product.ReleaseStockWithTx(
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

func (s *Service) getCart(ctx context.Context, userID string) (*Cart, error) {
	ownerID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	cartData, err := s.cart.GetCart(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	cart := &Cart{}
	for _, val := range cartData {

		// get latest price from product
		product, err := s.product.GetProductPrice(ctx, val.ProductID.String())
		if err != nil {
			return nil, err
		}

		cartItem := CartItem{
			ProductID: val.ProductID,
			Qty:       val.Qty,
			Price:     product.GetPrice(),
		}

		cart.Items = append(cart.Items, cartItem)
		cart.TotalAmount += product.GetPrice() * int64(val.Qty)
	}

	return cart, nil

}

func (s *Service) CreateOrder(ctx context.Context, userID string, req CreateOrderRequest) (*CreateOrderResponse, error) {
	// 1. get cart

	cart, err := s.getCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	// start DB transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	repo := s.repo.WithTx(tx)

	// create order
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	orderID := uuid.New()

	order := &Order{
		ID:          orderID,
		UserID:      uid,
		Status:      OrderStatusPending,
		TotalAmount: cart.TotalAmount,
		CreatedAt:   now,
	}

	if err := repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	for _, item := range cart.Items {
		// reserve stock
		err := s.product.ReserveStockWithTx(ctx, tx, item.ProductID.String(), item.Qty)
		if err != nil {
			return nil, err
		}

		// add order item, snapshot price
		orderItem := &OrderItem{
			ID:        uuid.New(),
			OrderID:   orderID,
			ProductID: item.ProductID,
			Price:     item.Price,
			Qty:       item.Qty,
			CreatedAt: now,
		}

		if err := repo.CreateOrderItem(ctx, orderItem); err != nil {
			return nil, err
		}

	}

	// create payment
	paymentResult, err := s.payment.CreatePaymentWithTx(ctx, tx, orderID.String(), order.TotalAmount, req.PaymentMethod)
	if err != nil {
		return nil, err
	}

	// commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// remove cart
	if _, err := s.cart.DeleteCart(ctx, uid); err != nil {
		return nil, err
	}

	// create order response
	orderRespnse := &CreateOrderResponse{
		OrderID:     orderID.String(),
		TotalAmount: helper.ToFloat(order.TotalAmount),
		PaymentURL:  paymentResult.PaymentURL,
		ExpiredAt:   helper.FormatOptionalTime(paymentResult.ExpiredAt),
	}
	return orderRespnse, nil

}

func (s *Service) Checkout(ctx context.Context, userID string) (*CheckoutResponse, error) {

	cart, err := s.getCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	coResp := &CheckoutResponse{}
	for _, val := range cart.Items {
		coItem := CheckoutItem{
			ProductID: val.ProductID.String(),
			Qty:       val.Qty,
			Price:     helper.ToFloat(val.Price),
		}
		coResp.Items = append(coResp.Items, coItem)
	}

	coResp.TotalAmount = helper.ToFloat(cart.TotalAmount)
	coResp.GrandTotal = helper.ToFloat(cart.TotalAmount)
	coResp.PaymentMethod = "" // temp hardcoded, should implemen later
	coResp.Shipping = nil     // temp hardcoded, should implemen later
	coResp.Promo = nil        // temp hardcoded, should implemen later

	return coResp, nil
}
