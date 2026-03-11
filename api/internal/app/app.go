package app

import (
	"github.com/alfin-akhret/ecommerce-system/internal/config"
	"github.com/alfin-akhret/ecommerce-system/internal/auth"
	"github.com/alfin-akhret/ecommerce-system/internal/user"
	"github.com/alfin-akhret/ecommerce-system/pkg/database"
)

type App struct {
	Config *config.Config

	UserHandler *user.UserHandler
	AuthHandler *auth.AuthHandler
}

func New() (*App, error) {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg.DBUrl)
	if err != nil {
		return nil, err
	}

	// rdb := database.NewRedis(cfg.RedisAddr)

	userRepo := user.NewUserRepository(db)
	userService := user.NewUserService(userRepo)
	userHandler := user.NewUserHandler(userService)
	authHandler := auth.NewAuthHandler(userService)

	return &App{
		Config:      cfg,
		UserHandler: userHandler,
		AuthHandler: authHandler,
	}, nil
}
