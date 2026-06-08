package middleware

type contextKey string

const (
	loggerContextKey    contextKey = "zapLogger"
	jwtClaimsContextKey contextKey = "jwtClaims"
)
