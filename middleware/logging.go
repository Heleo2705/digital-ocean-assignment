package middleware

import (
	"context"
	"net/http"

	"go.uber.org/zap"
)

type contextKey string

const loggerContextKey contextKey = "zapLogger"

func RequestLogger(root *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := root.With(
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
			)

			ctx := context.WithValue(r.Context(), loggerContextKey, logger)
			logger.Info("handling request")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetLogger(r *http.Request) *zap.Logger {
	logger, ok := r.Context().Value(loggerContextKey).(*zap.Logger)
	if !ok {
		return zap.NewNop()
	}
	return logger
}
