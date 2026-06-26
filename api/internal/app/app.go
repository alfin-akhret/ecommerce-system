package app

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/auth"
	"github.com/alfin-akhret/ecommerce-system/internal/cart"
	"github.com/alfin-akhret/ecommerce-system/internal/config"
	"github.com/alfin-akhret/ecommerce-system/internal/events"
	rabbitmqbroker "github.com/alfin-akhret/ecommerce-system/internal/events/rabbitmq_broker"
	"github.com/alfin-akhret/ecommerce-system/internal/mail"
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
	Broker                       *rabbitmqbroker.RabbitMQBroker
	EventPublisherWorker         *events.EventPublisherWorker
}

func New(ctx context.Context) (*App, error) {
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

	/*
		// in-memory message broker
		broker := events.NewMemoryBroker()
		// 2. analytic service
		broker.Subscribe("order.created", func(ctx context.Context, event events.Event) {
			payload := event.Payload.(events.OrderCreatedPayload)
			log.Printf("[Analytics] order_created order_id=%s", payload.OrderID)
		})
	*/

	// rabbitMQ message broker
	broker := rabbitmqbroker.CreateNewBroker(cfg.RabbitMQHost)

	// event service
	eventService := events.NewService(db)
	// event publisher worker
	hostname, _ := os.Hostname()
	lockedBy := fmt.Sprintf("%s:%d", hostname, os.Getpid())
	eventPublisherWorker := events.NewEventPublisherWorker(
		eventService,
		broker,
		10*time.Second,
		100,
		lockedBy,
		5*time.Second,
		3,
	)

	// event inbox repository
	inboxRepo := events.NewInboxRepository(db)

	// email service
	smtpPort, _ := strconv.Atoi(cfg.SMTPPort)
	mailConfig := &mail.EmailConfig{
		SMTPHost:      cfg.SMTPHost,
		SMTPPort:      smtpPort,
		DefaultSender: cfg.EmailDefaultSender,
	}
	emailService := mail.NewService(broker, mailConfig, db, inboxRepo)
	emailService.SubscribeTo(ctx, "order.created")
	emailService.SubscribeTo(ctx, "payment.callback.processed")
	emailService.SubscribeTo(ctx, "payment.expired")

	// payment
	paymentService := payment.NewService(db, broker)

	// order
	// order service uses message-broker to broadcast message
	orderService := order.NewService(db, productService, paymentService, cartService, broker)
	orderService.SubscribeTo(ctx, "payment.expired")
	orderHandler := order.NewHandler(orderService)

	paymentService.SetOrderStatusUpdater(orderService)
	paymentHandler := payment.NewHandler(paymentService)

	// worker
	expirationWorker := payment.NewPaymentExpirationWorker(
		paymentService,
		broker,
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
		Broker:                       broker,
		EventPublisherWorker:         eventPublisherWorker,
	}, nil
}
