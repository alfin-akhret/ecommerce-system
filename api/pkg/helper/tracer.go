package helper

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// Initialize open telemetry tracer
func InitTracer(ctx context.Context, serviceName string, otlpEndpoint string) (func(context.Context) error, error) {
	// 1. OTLP exporter (gRPC)
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otlpEndpoint),
		otlptracegrpc.WithInsecure(), // untuk local jaeger biasanya tidak pake TLS
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// 2. Resource (metadata service)
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource metadata: %w", err)
	}

	// 3. Tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// set global tracer provider
	otel.SetTracerProvider(tp)

	// 4. Propagator
	otel.SetTextMapPropagator(
		propagation.TraceContext{}, // W3C standard
	)

	// 5. Shutdown function
	shutdown := func(ctx context.Context) error {
		// kasih timeout biar ga nge hang
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := tp.Shutdown(ctx); err != nil {
			return fmt.Errorf("Failed to shutdown tracer provider: %w", err)
		}
		return nil
	}

	return shutdown, nil
}

func TracingMiddleware(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// ambil context dari request (penting)
			ctx := r.Context()

			// buat span baru
			ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path)
			defer span.End()

			// inject attribute dasar
			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.route", r.URL.Path),
			)

			// lanjut ke handler berikutnya
			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}
