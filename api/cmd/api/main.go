package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/internal/app"
	"github.com/alfin-akhret/ecommerce-system/internal/auth"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/go-chi/chi"
)

func main() {
	application, err := app.New()
	if err != nil {
		panic(err)
	}

	r := chi.NewRouter()

	// create new logger
	logger := helper.NewLogger()

	// register middlewares
	r.Use(helper.RecoveryMiddleware(logger))
	r.Use(helper.RequestIDMiddleware)
	r.Use(helper.LoggerMiddleware(logger))
	r.Use(helper.MetricsMiddleware)

	// health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("OK cool")); err != nil {
			log.Printf("health write failed: %v", err)
		}
	})

	// metrics endpoint for prometheus
	r.Handle("/metrics", promhttp.Handler())

	r.Get("/users/{id}", helper.Handle(application.UserHandler.GetUser))
	r.Post("/register", helper.Handle(application.UserHandler.Register))
	r.Post("/login", helper.Handle(application.AuthHandler.Login))

	// protected: user
	r.With(auth.AuthMiddleware).Get("/me", helper.Handle(application.AuthHandler.Me))

	// protected: order
	r.With(auth.AuthMiddleware).Get("/orders", helper.Handle(application.OrderHandler.ListOrders))
	r.With(auth.AuthMiddleware).Get("/orders/{id}", helper.Handle(application.OrderHandler.GetOrder))
	r.With(auth.AuthMiddleware).Post("/create", helper.Handle(application.OrderHandler.CreateOrder))
	r.With(auth.AuthMiddleware).Get("/checkout", helper.Handle(application.OrderHandler.Checkout))

	// protected: cart
	r.With(auth.AuthMiddleware).Post("/cart", helper.Handle(application.CartHandler.AddItem))
	r.With(auth.AuthMiddleware).Delete("/cart", helper.Handle(application.CartHandler.DeleteCart))
	r.With(auth.AuthMiddleware).Delete("/cart/{product_id}", helper.Handle(application.CartHandler.RemoveItem))
	r.With(auth.AuthMiddleware).Patch("/cart", helper.Handle(application.CartHandler.UpdateQuantity))
	r.With(auth.AuthMiddleware).Get("/cart", helper.Handle(application.CartHandler.GetCart))

	// Payments route
	r.Route("/payments", func(r chi.Router) {
		r.With(auth.AuthMiddleware).Post("/", helper.Handle(application.PaymentHandler.CreatePayment))
		r.With(auth.AuthMiddleware).Get("/{payment_id}", helper.Handle(application.PaymentHandler.GetPayment))
		r.With(auth.AuthMiddleware).Patch("/{payment_id}", helper.Handle(application.PaymentHandler.UpdatePaymentStatus))
		r.Post("/callback", helper.Handle(application.PaymentHandler.HandleCallback))
		r.Post("/{payment_id}/success", helper.Handle(application.PaymentHandler.ProcessPaymentSuccess))
		r.Post("/{payment_id}/fail", helper.Handle(application.PaymentHandler.ProcessPaymentFailed))
	})

	r.Route("/products", func(r chi.Router) {
		r.Get("/", helper.Handle(application.ProductHandler.ListProducts))
		r.Get("/{id}", helper.Handle(application.ProductHandler.GetProduct))

		// protected: khusus admin/internal
		r.With(auth.AuthMiddleware, auth.AdminMiddleware).Post("/users", helper.Handle(application.UserHandler.CreateUser))
		r.With(auth.AuthMiddleware, auth.AdminMiddleware).Post("/", helper.Handle(application.ProductHandler.CreateProduct))
		r.With(auth.AuthMiddleware, auth.AdminMiddleware).Patch("/{id}/stock", helper.Handle(application.ProductHandler.UpdateStock))
		r.With(auth.AuthMiddleware, auth.AdminMiddleware).Post("/{id}/reserve", helper.Handle(application.ProductHandler.ReserveStock))
		r.With(auth.AuthMiddleware, auth.AdminMiddleware).Post("/{id}/release", helper.Handle(application.ProductHandler.ReleaseStock))
		r.With(auth.AuthMiddleware, auth.AdminMiddleware).Post("/{id}/confirm", helper.Handle(application.ProductHandler.ConfirmStock))

	})

	// panic test
	r.Get("/panic", PanicHandler)

	// run worker
	ctx := context.Background()

	application.PaymentExpirationWorker.Start(ctx)
	application.IdempotencyKeyDeletionWorker.Start(ctx)

	// handle shutdown
	defer application.PaymentExpirationWorker.Stop()
	defer application.IdempotencyKeyDeletionWorker.Stop()

	// select {} // block forever (sementara)

	fmt.Println("Server is running on :" + application.Config.Port)

	if err := http.ListenAndServe(":"+application.Config.Port, r); err != nil {
		log.Fatal(err)
	}

}

func PanicHandler(w http.ResponseWriter, r *http.Request) {
	panic("something went terribly wrong")
}
