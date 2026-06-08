package service

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type LocalClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email,omitempty"`
}

func GenerateAccessToken(secret, subject, email string, ttl time.Duration) (string, error) {
	claims := LocalClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Email: email,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func GenerateRefreshToken(secret, subject string, ttl time.Duration) (string, error) {
	claims := LocalClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func ValidateAccessToken(secret, tokenString string) (*LocalClaims, error) {
	if tokenString == "" {
		return nil, errors.New("missing access token")
	}

	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer"))
	claims := &LocalClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid access token")
	}
	return claims, nil
}
