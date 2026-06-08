package middleware

import (
	"context"
	"net/http"

	"github.com/Heleo2705/assignment/service"
)

func NewJWTAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			claims, err := service.ValidateAccessToken(secret, authHeader)
			if err != nil {
				http.Error(w, "invalid access token", http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, jwtClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetJWTClaims(r *http.Request) *service.LocalClaims {
	claims, ok := r.Context().Value(jwtClaimsContextKey).(*service.LocalClaims)
	if !ok {
		return nil
	}
	return claims
}
