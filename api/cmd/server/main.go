package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
)

func main() {
	// Load env (atau hardcode dulu untuk local)
	//os.Setenv("POSTGRES_USER", "admin")
	//os.Setenv("POSTGRES_PASSWORD", "admin")
	//os.Setenv("POSTGRES_HOST", "localhost")
	//os.Setenv("POSTGRES_PORT", "5432")
	//os.Setenv("POSTGRES_DB", "ecommerce")
	//os.Setenv("REDIS_HOST", "localhost")
	//os.Setenv("REDIS_PORT", "6379")

	//if err := db.InitDB(); err != nil {
	//log.Fatalf("DB init failed: %v", err)
	//}

	//r := gin.Default()

	//// health check
	//r.GET("/health", handler.HealthCheck)

	//port := "8080"
	//fmt.Printf("Server running on port %s\n", port)
	//if err := r.Run(":" + port); err != nil {
	//log.Fatalf("Server failed: %v", err)
	//}

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK cool"))
	})

	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":8080", r)
}
