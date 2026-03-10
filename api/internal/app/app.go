package app

import (
	"github.com/alfin-akhret/ecommerce-system/internal/config"
	"github.com/alfin-akhret/ecommerce-system/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type App struct {
	Config *config.Config
	DB     *pgxpool.Pool
	Redis  *redis.Client
}

func New() (*App, error) {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg.DBUrl)
	if err != nil {
		return nil, err
	}

	rdb := database.NewRedis(cfg.RedisAddr)

	return &App{
		Config: cfg,
		DB:     db,
		Redis:  rdb,
	}, nil
}
