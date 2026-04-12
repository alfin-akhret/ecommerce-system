package helper

import (
	"context"
	"net/http"

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

func LoggerMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			reqID := r.Context().Value(RequestIDKey).(string)
			l := logger.With(zap.String("request_id", reqID))
			ctx := context.WithValue(r.Context(), "logger", l)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
