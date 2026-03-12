package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/internal/app"
	"github.com/alfin-akhret/ecommerce-system/internal/auth"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"

	"github.com/go-chi/chi"
)

func main() {
	application, err := app.New()
	if err != nil {
		panic(err)
	}

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("OK cool")); err != nil {
			log.Printf("health write failed: %v", err)
		}
	})

	r.Post("/users", helper.Handle(application.UserHandler.CreateUser))
	r.Get("/users/{id}", helper.Handle(application.UserHandler.GetUser))
	r.Post("/register", helper.Handle(application.UserHandler.Register))
	r.Post("/login", helper.Handle(application.AuthHandler.Login))
	r.With(auth.AuthMiddleware).Get("/me", helper.Handle(application.AuthHandler.Me))

	r.Route("/products", func(r chi.Router) {
		r.Post("/", helper.Handle(application.ProductHandler.CreateProduct))
	})

	fmt.Println("Server is running on :" + application.Config.Port)

	if err := http.ListenAndServe(":"+application.Config.Port, r); err != nil {
		log.Fatal(err)
	}
}
