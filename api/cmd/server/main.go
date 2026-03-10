package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
)

func main() {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK cool"))
	})

	fmt.Println("Server is running on :8080")
	http.ListenAndServe(":8080", r)
}
