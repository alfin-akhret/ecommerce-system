package helper

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func NewLogger() *zap.Logger {
	logger, _ := zap.NewProduction()
	return logger
}

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

type loggerKey string

const LoggerIDKey = loggerKey("logger")

func LoggerMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			reqID := r.Context().Value(RequestIDKey).(string)
			l := logger.With(zap.String("request_id", reqID))
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
