package middleware

type contextKey string

const (
	loggerContextKey    contextKey = "zapLogger"
	claimsContextKey    contextKey = "keycloakClaims"
	jwtClaimsContextKey contextKey = "jwtClaims"
)
