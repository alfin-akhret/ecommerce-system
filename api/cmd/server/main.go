package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/internal/app"

	"github.com/go-chi/chi"
)

func main() {
	application, err := app.New()
	if err != nil {
		panic(err)
	}

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK cool"))
	})

	r.Post("/users", application.UserHandler.CreateUser)

	fmt.Println("Server is running on :" + application.Config.Port)

	if err := http.ListenAndServe(":"+application.Config.Port, r); err != nil {
		log.Fatal(err)
	}
}
