package app

import (
	"github.com/alfin-akhret/ecommerce-system/internal/config"
	"github.com/alfin-akhret/ecommerce-system/internal/database"
	"github.com/alfin-akhret/ecommerce-system/internal/handler"
	"github.com/alfin-akhret/ecommerce-system/internal/repository"
	"github.com/alfin-akhret/ecommerce-system/internal/service"
)

type App struct {
	Config *config.Config

	UserHandler *handler.UserHandler
}

func New() (*App, error) {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg.DBUrl)
	if err != nil {
		return nil, err
	}

	// rdb := database.NewRedis(cfg.RedisAddr)

	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService)

	return &App{
		Config:      cfg,
		UserHandler: userHandler,
	}, nil
}
