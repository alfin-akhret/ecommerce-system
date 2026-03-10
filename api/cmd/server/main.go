package main

import (
	"fmt"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/internal/config"
	"github.com/alfin-akhret/ecommerce-system/internal/database"

	"github.com/go-chi/chi"
)

func main() {
	cfg := config.Load()

	db, err := database.NewPostgres(cfg.DBUrl)
	if err != nil {
		panic(err)
	}

	redis := database.NewRedis(cfg.RedisAddr)

	fmt.Println("Postgres connected:", db != nil)
	fmt.Println("Redis connected:", redis != nil)

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK cool"))
	})

	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":"+cfg.Port, r)
}
