package app

import (
	"context"
	"log"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/auth"
	"github.com/alfin-akhret/ecommerce-system/internal/cart"
	"github.com/alfin-akhret/ecommerce-system/internal/config"
	"github.com/alfin-akhret/ecommerce-system/internal/order"
	"github.com/alfin-akhret/ecommerce-system/internal/payment"
	"github.com/alfin-akhret/ecommerce-system/internal/platform/database"
	"github.com/alfin-akhret/ecommerce-system/internal/product"
	"github.com/alfin-akhret/ecommerce-system/internal/user"
)

type App struct {
	Config *config.Config

	UserHandler                  *user.UserHandler
	AuthHandler                  *auth.AuthHandler
	ProductHandler               *product.Handler
	OrderHandler                 *order.Handler
	PaymentHandler               *payment.Handler
	CartHandler                  *cart.Handler
	PaymentExpirationWorker      *payment.PaymentExpirationWorker
	IdempotencyKeyDeletionWorker *order.IdempotencyKeyDeleteWorker
}

func New() (*App, error) {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg.DBUrl)
	if err != nil {
		return nil, err
	}

	rdb := database.NewRedis(cfg.RedisAddr)

	// user
	userRepo := user.NewUserRepository(db)
	userService := user.NewUserService(userRepo)
	userHandler := user.NewUserHandler(userService)
	authHandler := auth.NewAuthHandler(userService)

	// product
	productService := product.NewService(db)
	productHandler := product.NewHandler(productService)

	// cart
	cartRepo := cart.CreateNewCartRepository(rdb)
	cartService := cart.NewCartService(cartRepo, productService)
	cartHandler := cart.NewHandler(cartService)

	// payment
	paymentService := payment.NewService(db)

	// order
	orderService := order.NewService(db, productService, paymentService, cartService)
	orderHandler := order.NewHandler(orderService)

	paymentService.SetOrderStatusUpdater(orderService)

	paymentHandler := payment.NewHandler(paymentService)

	// publisher (sementara simple dulu)
	eventPublisher := payment.NewInMemoryPublisher()
	eventPublisher.Subscribe("payment.expired", func(ctx context.Context, payload any) {
		event, ok := payload.(payment.PaymentExpiredEvent)
		if !ok {
			log.Printf("[Event] invalid payload for payment.expired: %T\n", payload)
			return
		}

		orderService.CancelOrder(ctx, event.OrderID)
	})

	// worker
	expirationWorker := payment.NewPaymentExpirationWorker(
		paymentService,
		eventPublisher,
		10*time.Second,
		100,
	)

	// idempotency key delete worker
	iKeyDeletWorker := order.NewIdempotencyKeyDeleteWorker(
		orderService,
		10*time.Second,
		100,
	)

	return &App{
		Config:                       cfg,
		UserHandler:                  userHandler,
		AuthHandler:                  authHandler,
		ProductHandler:               productHandler,
		OrderHandler:                 orderHandler,
		PaymentHandler:               paymentHandler,
		CartHandler:                  cartHandler,
		PaymentExpirationWorker:      expirationWorker,
		IdempotencyKeyDeletionWorker: iKeyDeletWorker,
	}, nil
}
