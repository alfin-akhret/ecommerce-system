package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/app"
	"github.com/alfin-akhret/ecommerce-system/internal/auth"
	"github.com/alfin-akhret/ecommerce-system/internal/jobs"
	"github.com/alfin-akhret/ecommerce-system/internal/queue"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/go-chi/chi"
)

func main() {
	// === 1. init main Application
	application, err := app.New()
	if err != nil {
		panic(err)
	}

	// === 2. Init router
	r := chi.NewRouter()

	// === 3. Init Logger middleware
	logger := helper.NewLogger()

	// === 4. Init Tracer
	// root context mesti gracefully shutdown
	// karena http.ListenAndServe(...) itu blocking, sementara root context di shutdown menggunakan defer
	// defer hanya jalan jika function return, masalahnya ini main(), artinya kalau main() return, aplikasi mati
	// defer bisa jadi ga pernah kepanggil, dan trace span terakhir ga akan terkirim (data loss)
	// solusi:
	// 1. dengerin signal (SIGINT, SIGTERM)
	// 2. stop server
	// 3. shutdown tracer (flush data)
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	shutdown, err := helper.InitTracer(rootCtx,
		application.Config.OTelServiceName,
		application.Config.OTelExporterEndpoint,
	)
	if err != nil {
		log.Fatalf("failed to init tracer: %v", err)
	}

	// === 5. List of routes
	// metric endpoint, for promotheus scrapper
	r.Handle("/metrics", promhttp.Handler())

	// main routers
	r.Group(func(r chi.Router) {

		// register middlewares
		r.Use(helper.TracingMiddleware(application.Config.OTelServiceName))
		r.Use(helper.RequestIDMiddleware)
		r.Use(helper.RecoveryMiddleware(logger))
		r.Use(helper.LoggerMiddleware(logger))
		r.Use(helper.AccessLogMiddleware(logger))
		r.Use(helper.MetricsMiddleware)

		// health check endpoint
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("OK cool")); err != nil {
				log.Printf("health write failed: %v", err)
			}
		})

		r.Get("/users/{id}", helper.Handle(application.UserHandler.GetUser))
		r.Post("/register", helper.Handle(application.UserHandler.Register))
		r.Post("/login", helper.Handle(application.AuthHandler.Login))

		// protected: user
		r.With(auth.AuthMiddleware).Get("/me", helper.Handle(application.AuthHandler.Me))

		// protected: order
		r.With(auth.AuthMiddleware).Get("/orders", helper.Handle(application.OrderHandler.ListOrders))
		r.With(auth.AuthMiddleware).Get("/orders/{id}", helper.Handle(application.OrderHandler.GetOrder))
		r.With(auth.AuthMiddleware).Post("/orders/create", helper.Handle(application.OrderHandler.CreateOrder))
		r.With(auth.AuthMiddleware).Get("/orders/checkout", helper.Handle(application.OrderHandler.Checkout))

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

		// alert test p95:
		r.Get("/slow", func(w http.ResponseWriter, r *http.Request) {
			delay := time.Duration(rand.Intn(3000)) * time.Millisecond
			time.Sleep(delay)
			w.Write([]byte("ok"))
		})

	})

	// === 6. Run Workers
	// create new worker context derived from root context
	ctx, workerCancel := context.WithCancel(rootCtx)

	application.PaymentExpirationWorker.Start(ctx)
	application.IdempotencyKeyDeletionWorker.Start(ctx)

	// handle shutdown

	// === 7. Run HTTP server
	// jalankan di go routine
	srv := &http.Server{
		Addr:    ":" + application.Config.Port,
		Handler: r,
	}

	go func() {
		fmt.Println("Server is running on :" + application.Config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()

	// wait shutdown signal
	<-rootCtx.Done()
	log.Println("shutting down HTTP Server...")

	// === 8. Shutdown
	// urutan shutdown PENTING
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// ==== 8.1 stop HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	// ==== 8.2 stop workers
	workerCancel()

	// ==== 8.3 shutdown tracer (flush span)
	if err := shutdown(shutdownCtx); err != nil {
		log.Printf("tracer shutdown error: %v", err)
	}

	// 9. queue and worker
	registry := &queue.Registry{
		Handlers: make(map[string]queue.Handler),
	}

	// register send email job to the queue
	registry.Register("send_email", jobs.SendEmailHandler)

	// queue
	queue := &queue.Queue{
		Jobs:     make(chan queue.Job, 100),
		Registry: registry,
	}

	go queue.StartWorker(context.Background())
}

func PanicHandler(w http.ResponseWriter, r *http.Request) {
	panic("something went terribly wrong")
}
