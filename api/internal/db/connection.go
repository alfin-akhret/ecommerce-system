package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	PostgresPool *pgxpool.Pool
	RedisClient  *redis.Client
)

func InitDB() error {
	// Postgres connection
	pgURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"),
	)

	pool, err := pgxpool.New(context.Background(), pgURL)
	if err != nil {
		return fmt.Errorf("failed to connect postgres: %w", err)
	}

	PostgresPool = pool

	// Redis connection
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
		Password: "", // no password
		DB:       0,
	})

	_, err = RedisClient.Ping(context.Background()).Result()
	if err != nil {
		return fmt.Errorf("failed to connect redis: %w", err)
	}

	return nil
}
