package helper

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
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
