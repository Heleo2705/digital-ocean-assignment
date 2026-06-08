package middleware

import (
	"context"
	"net/http"

	"github.com/Heleo2705/assignment/service"
)

type contextKey string

const claimsContextKey contextKey = "keycloakClaims"

func NewKeycloakAuth(issuer, clientID string) func(http.Handler) http.Handler {
	kc := service.NewKeycloakService(issuer, clientID)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing Authorization header", http.StatusUnauthorized)
				return
			}

			claims, err := kc.ValidateAccessToken(r.Context(), authHeader)
			if err != nil {
				http.Error(w, "invalid access token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetKeycloakClaims(r *http.Request) *service.KeycloakClaims {
	claims, ok := r.Context().Value(claimsContextKey).(*service.KeycloakClaims)
	if !ok {
		return nil
	}
	return claims
}
