package order

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/contracts"
	"github.com/alfin-akhret/ecommerce-system/internal/jobs"
	"github.com/alfin-akhret/ecommerce-system/internal/queue"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

var ErrOrderNotFound = errors.New("order not found")
var ErrKeyNotFound = errors.New("idempotency key not found")
var ErrCartItem = errors.New("there's no item in the cart or the total amount is 0")

type Service struct {
	db      *pgxpool.Pool
	repo    *Repository
	product contracts.ProductManager
	payment contracts.PaymentManager
	cart    contracts.CartManager
	queue   *queue.RedisQueue
}

func NewService(db *pgxpool.Pool,
	product contracts.ProductManager,
	payment contracts.PaymentManager,
	cart contracts.CartManager,
	queue *queue.RedisQueue) *Service {

	repo := NewOrderRepository(db)

	return &Service{
		db:      db,
		repo:    repo,
		product: product,
		payment: payment,
		cart:    cart,
		queue:   queue,
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
	tr := otel.Tracer("order.service")
	ctx, span := tr.Start(ctx, "order.service.UpdateOrderStatusWithTx")
	defer span.End()

	repo := s.repo.WithTx(tx)

	err := repo.UpdateOrderStatus(ctx, orderID, status)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// expire order idempotency key
	_ = repo.ExpireIdempotencyKey(ctx, orderID)

	return nil
}

func (s *Service) UpdateStatus(ctx context.Context, orderID string, status string) error {
	return s.repo.UpdateOrderStatus(ctx, orderID, status)
}

func (s *Service) ConfirmOrderStockWithTx(ctx context.Context, tx pgx.Tx, orderID string) error {

	tr := otel.Tracer("order.service")
	ctx, span := tr.Start(ctx, "order.service.ConfirmOrderStockWithTx")
	defer span.End()

	items, err := s.repo.ListOrderItems(ctx, orderID)
	if err != nil {
		return err
	}

	for _, item := range items {
		if err := s.product.ConfirmStockWithTx(ctx, tx, item.ProductID.String(), item.Qty); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	return nil
}

func (s *Service) ReleaseOrderStockWithTx(ctx context.Context, tx pgx.Tx, orderID string) error {

	tr := otel.Tracer("order.service")
	ctx, span := tr.Start(ctx, "order.service.ReleaseOrderStockWithTx")
	defer span.End()

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
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	return nil
}

// subscribe to topic: "payment.expired"
func (s *Service) CancelOrder(ctx context.Context, orderID string) {
	logger := helper.LoggerFromCtx(ctx)

	tr := otel.Tracer("order.service")
	ctx, span := tr.Start(ctx, "order.service.CancelOrder")
	defer span.End()

	log.Println("[Order] cancel order:", orderID)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error("[Order Service] cancel order: begin transaction",
			zap.String("error_message", err.Error()),
			zap.String("order_id", orderID),
		)
		return
	}
	defer tx.Rollback(ctx)

	if err := s.UpdateOrderStatusWithTx(ctx, tx, orderID, "CANCELLED"); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error("[Order Service] update order status",
			zap.String("error_message", err.Error()),
			zap.String("order_id", orderID),
		)
		return
	}

	if err := s.ReleaseOrderStockWithTx(ctx, tx, orderID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error("[Order Service] release order stock",
			zap.String("error_message", err.Error()),
			zap.String("order_id", orderID),
		)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error("[Order Service] cancler order: commit transaction",
			zap.String("error_message", err.Error()),
			zap.String("order_id", orderID),
		)
		return
	}

}

func (s *Service) getCart(ctx context.Context, userID string) (*Cart, error) {
	log := helper.LoggerFromCtx(ctx)

	tr := otel.Tracer("order.service")
	ctx, span := tr.Start(ctx, "order.service.getCart")
	defer span.End()

	ownerID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	cartData, err := s.cart.GetCart(ctx, ownerID)
	if err != nil {
		log.Error("Order: cart not found",
			zap.String("user_id", userID),
			zap.String("error_message", err.Error()))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	cart := &Cart{}
	for _, val := range cartData {

		// get latest price from product
		product, err := s.product.GetProductPrice(ctx, val.ProductID.String())
		if err != nil {
			log.Error("Order: failed getting product price",
				zap.String("product_id", product.GetID().String()),
				zap.String("error_message", err.Error()))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
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

func (s *Service) CreateOrder(ctx context.Context, userID string,
	req CreateOrderRequest, key string) (*CreateOrderResponse, error) {
	// set timeout
	// ini untuk menjaga external call seperti ke: db, redis,
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// logger
	log := helper.LoggerFromCtx(ctx)
	log.Info("Order: Creating order", zap.String("user_id", userID))

	// tracer
	tr := otel.Tracer("order.service")
	ctx, span := tr.Start(ctx, "order.service.CreateOrder")
	defer span.End()

	// 1. get cart
	cart, err := s.getCart(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error("Order: Failed to get cart",
			zap.String("user_id", userID),
			zap.String("error_message", err.Error()),
		)
		return nil, err
	}

	if cart.Items == nil || cart.TotalAmount == 0 {
		log.Warn("Order: Cart is empty or total amount is zero",
			zap.String("user_id", userID),
			zap.String("error_message",
				ErrCartItem.Error()),
		)
		return nil, ErrCartItem
	}

	// start DB transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Error("Order: Failed to start DB transaction",
			zap.String("user_id", userID),
			zap.String("error_message",
				err.Error()),
		)
		return nil, err
	}
	defer tx.Rollback(ctx)

	repo := s.repo.WithTx(tx)

	// create order
	uid, err := uuid.Parse(userID)
	if err != nil {
		log.Error("Order: Failed to parse user ID",
			zap.String("user_id", userID),
			zap.String("error_message",
				err.Error()),
		)
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

	orderItems := make([]*OrderItem, 0, len(cart.Items))

	for _, item := range cart.Items {
		// reserve stock
		err := s.product.ReserveStockWithTx(ctx, tx, item.ProductID.String(), item.Qty)
		if err != nil {
			log.Error(
				"Order: Failed to reserve stock",
				zap.String("user_id", userID),
				zap.String("product_id", item.ProductID.String()),
				zap.String("error_message", err.Error()),
			)

			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		// add order item, snapshot price
		orderItems = append(orderItems, &OrderItem{
			ID:        uuid.New(),
			OrderID:   orderID,
			ProductID: item.ProductID,
			Price:     item.Price,
			Qty:       item.Qty,
			CreatedAt: now,
		})
	}

	if err := repo.CreateOrder(ctx, order); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error("Order: Failed to create order",
			zap.String("user_id", userID),
			zap.Stack(err.Error()),
		)
		return nil, err
	}

	for _, orderItem := range orderItems {
		if err := repo.CreateOrderItem(ctx, orderItem); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.Error("Order: Failed to create order item",
				zap.String("user_id", userID),
				zap.Stack(err.Error()),
			)
			return nil, err
		}
	}

	// create payment
	paymentResult, err := s.payment.CreatePaymentWithTx(ctx, tx, orderID.String(), order.TotalAmount, req.PaymentMethod)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error("Order: Failed to create payment",
			zap.String("user_id", userID),
			zap.Stack(err.Error()),
		)
		return nil, err
	}

	// create order response
	orderRespnse := &CreateOrderResponse{
		OrderID:     orderID.String(),
		TotalAmount: helper.ToFloat(order.TotalAmount),
		PaymentURL:  paymentResult.PaymentURL,
		ExpiredAt:   helper.FormatOptionalTime(paymentResult.ExpiredAt),
	}

	// save idempotency
	iKey, err := uuid.Parse(key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error(
			"Order: Failed to parse idempotency key",
			zap.String("user_id", userID),
			zap.String("order_id", orderID.String()),
			zap.String("error_message", err.Error()),
		)
		return nil, err
	}

	jsonResponse, err := json.Marshal(orderRespnse)
	if err != nil {
		log.Error(
			"Order: Failed to encode idempotency response",
			zap.String("user_id", userID),
			zap.String("order_id", orderID.String()),
			zap.String("error_message", err.Error()),
		)
		return nil, err
	}

	if err := s.repo.SaveIdempotencyKey(ctx,
		iKey,
		uid,
		orderID,
		OrderStatusPending,
		jsonResponse,
		*paymentResult.ExpiredAt,
	); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error("Order: Failed to save idempotency key",
			zap.String("user_id", userID),
			zap.String("order_id", orderID.String()),
			zap.String("error_message", err.Error()),
		)
		return nil, err
	}

	// commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Error("Order: Failed to commit transaction",
			zap.String("user_id", userID),
			zap.String("error_message", err.Error()),
		)
		return nil, err
	}

	span.SetAttributes(
		attribute.Bool("order.created", true),
		attribute.String("order.id", orderRespnse.OrderID),
		attribute.String("user.id", userID),
	)

	log.Info("Order: Order created", zap.String("user_id", userID), zap.String("order_id", orderID.String()))

	// remove cart
	if _, err := s.cart.DeleteCart(ctx, uid); err != nil {
		log.Error("Order: Failed to delete cart after creating order",
			zap.String("user_id", userID),
			zap.String("order_id", orderID.String()),
			zap.String("error_message", err.Error()),
		)
		return nil, err
	}

	// create send email job
	payload, _ := json.Marshal(jobs.SendEmailPayload{
		OrderID: orderID.String(),
		Email:   "testingemail@gmail.com",
	})

	s.queue.Enqueue(queue.Job{
		Type:    "send_email",
		Payload: payload,
		Timeout: 5 * time.Second,
	})

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

func (s *Service) CheckIdempotency(ctx context.Context, userID string, key string) (*CheckIdempotencyResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	iKey, err := uuid.Parse(key)
	if err != nil {
		return nil, err
	}

	result, err := s.repo.GetIdempotencyKey(ctx, uid, iKey)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	expiredAt := ""
	if formatted := helper.FormatOptionalTime(result.ExpiredAt); formatted != nil {
		expiredAt = *formatted
	}

	reponse := &CheckIdempotencyResponse{
		Key:       result.Key.String(),
		UserID:    result.UserID.String(),
		Status:    result.Status,
		ExpiredAt: expiredAt,
		Response:  result.Response,
	}

	return reponse, nil
}

func (s *Service) DeleteIdempotencyKey(ctx context.Context, limit int) ([]DeletedKeys, error) {
	result, err := s.repo.DeleteIdempotencyKey(ctx, limit)
	if err != nil {
		return nil, err
	}
	return result, nil
}
