package main

import (
	"net/http"
	"os"

	"github.com/Heleo2705/assignment/db"
	"github.com/Heleo2705/assignment/handler"
	appmiddleware "github.com/Heleo2705/assignment/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL is required")
	}

	if err := db.Migrate(databaseURL); err != nil {
		logger.Fatal("database migration failed", zap.Error(err))
	}

	dbConn, err := db.OpenDatabase(databaseURL)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}
	defer dbConn.Close()

	store := db.NewStore(dbConn)
	h := handler.New(store)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Fatal("JWT_SECRET is required")
	}

	var auth func(http.Handler) http.Handler

	keycloakURL := os.Getenv("KEYCLOAK_URL")
	if keycloakURL != "" {
		clientID := os.Getenv("KEYCLOAK_CLIENT_ID")
		if clientID == "" {
			clientID = "assignment-api"
		}
		logger.Info("using keycloak authentication", zap.String("issuer", keycloakURL), zap.String("client_id", clientID))
		auth = appmiddleware.NewKeycloakAuth(keycloakURL, clientID)
	} else {
		auth = appmiddleware.NewJWTAuth(jwtSecret)
	}

	r := chi.NewRouter()
	r.Use(appmiddleware.RequestLogger(logger))
	h.RegisterRoutes(r, auth)

	logger.Info("starting app", zap.String("addr", ":8000"))
	if err := http.ListenAndServe(":8000", r); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}
