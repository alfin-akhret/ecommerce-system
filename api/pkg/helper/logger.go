package helper

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func NewLogger() *zap.Logger {
	logger, _ := zap.NewProduction()
	return logger
}

// Request ID Middleware
// Inject request ID to request context

type contextKey string

const RequestIDKey = contextKey("request_id")

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)

		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logger middleware
// Inject logger to the request context
// the logger also contain the request ID

type loggerKey string

const LoggerIDKey = loggerKey("logger")

func LoggerMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// add tracer span and tracer span context; see helper/tracer.go
			span := trace.SpanFromContext(r.Context())
			spanCtx := span.SpanContext()
			traceID := spanCtx.TraceID().String()
			spanID := spanCtx.SpanID().String()

			reqID := r.Context().Value(RequestIDKey).(string)
			l := logger.With(zap.String("request_id", reqID),
				zap.String("trace_id", traceID),
				zap.String("span_id", spanID),
			)
			ctx := context.WithValue(r.Context(), LoggerIDKey, l)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// helper to get logger from context
func LoggerFromCtx(ctx context.Context) *zap.Logger {
	l, ok := ctx.Value(LoggerIDKey).(*zap.Logger)
	if !ok {
		return NewLogger()
	}
	return l
}

func RecoveryMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			defer func() {
				if rec := recover(); rec != nil {
					// ambil request id kalau ada
					reqID, _ := r.Context().Value(RequestIDKey).(string)

					logger.Error("panic recovered",
						zap.Any("error", rec),
						zap.String("request_id", reqID),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.ByteString("stack trace", debug.Stack()))

					// return response ke client
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}

			}()

			next.ServeHTTP(w, r)
		})
	}
}

// Access Log middleware
// to log incoming request and response
func AccessLogMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rec := &statusRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rec, r)

			// add tracer span and tracer span context; see helper/tracer.go
			span := trace.SpanFromContext(r.Context())
			spanCtx := span.SpanContext()
			traceID := spanCtx.TraceID().String()
			spanID := spanCtx.SpanID().String()

			reqID, _ := r.Context().Value(RequestIDKey).(string)

			logger.Info("http request completed",
				zap.String("request_id", reqID),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status_code", rec.status),
				zap.Int64("duration_ms", time.Since(start).Milliseconds()),
				zap.String("remote_ip", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
				zap.String("trace_id", traceID),
				zap.String("span_id", spanID),
			)

		})
	}
}
