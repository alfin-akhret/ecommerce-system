package app

import (
	"github.com/alfin-akhret/ecommerce-system/internal/auth"
	"github.com/alfin-akhret/ecommerce-system/internal/config"
	"github.com/alfin-akhret/ecommerce-system/internal/order"
	"github.com/alfin-akhret/ecommerce-system/internal/platform/database"
	"github.com/alfin-akhret/ecommerce-system/internal/product"
	"github.com/alfin-akhret/ecommerce-system/internal/user"
)

type App struct {
	Config *config.Config

	UserHandler    *user.UserHandler
	AuthHandler    *auth.AuthHandler
	ProductHandler *product.Handler
	OrderHandler   *order.Handler
}

func New() (*App, error) {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg.DBUrl)
	if err != nil {
		return nil, err
	}

	// rdb := database.NewRedis(cfg.RedisAddr)

	// user
	userRepo := user.NewUserRepository(db)
	userService := user.NewUserService(userRepo)
	userHandler := user.NewUserHandler(userService)
	authHandler := auth.NewAuthHandler(userService)

	// product
	productService := product.NewService(db)
	productHandler := product.NewHandler(productService)

	// order
	orderService := order.NewService(db)
	orderHandler := order.NewHandler(orderService)

	return &App{
		Config:         cfg,
		UserHandler:    userHandler,
		AuthHandler:    authHandler,
		ProductHandler: productHandler,
		OrderHandler:   orderHandler,
	}, nil
}
