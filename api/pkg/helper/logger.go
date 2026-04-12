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
