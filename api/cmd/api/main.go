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

	// protected: khusus user
	r.With(auth.AuthMiddleware).Get("/me", helper.Handle(application.AuthHandler.Me))
	r.With(auth.AuthMiddleware).Post("/orders", helper.Handle(application.OrderHandler.CreateOrder))
	r.With(auth.AuthMiddleware).Get("/orders", helper.Handle(application.OrderHandler.ListOrders))
	r.With(auth.AuthMiddleware).Get("/orders/{id}", helper.Handle(application.OrderHandler.GetOrder))

	r.Route("/products", func(r chi.Router) {
		r.Get("/", helper.Handle(application.ProductHandler.ListProducts))
		r.Get("/{id}", helper.Handle(application.ProductHandler.GetProduct))

		// protected: khusus admin/internal
		r.With(auth.AuthMiddleware, auth.AdminMiddleware).Post("/", helper.Handle(application.ProductHandler.CreateProduct))
		r.With(auth.AuthMiddleware, auth.AdminMiddleware).Patch("/{id}/stock", helper.Handle(application.ProductHandler.UpdateStock))
		r.With(auth.AuthMiddleware, auth.AdminMiddleware).Post("/{id}/reserve", helper.Handle(application.ProductHandler.ReserveStock))
		r.With(auth.AuthMiddleware, auth.AdminMiddleware).Post("/{id}/release", helper.Handle(application.ProductHandler.ReleaseStock))
		r.With(auth.AuthMiddleware, auth.AdminMiddleware).Post("/{id}/confirm", helper.Handle(application.ProductHandler.ConfirmStock))

	})

	fmt.Println("Server is running on :" + application.Config.Port)

	if err := http.ListenAndServe(":"+application.Config.Port, r); err != nil {
		log.Fatal(err)
	}
}
