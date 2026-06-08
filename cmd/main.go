package main

import (
	"net/http"
	"os"

	"github.com/Heleo2705/assignment/db"
	"github.com/Heleo2705/assignment/handler"
	appmiddleware "github.com/Heleo2705/assignment/middleware"

	_ "github.com/Heleo2705/assignment/docs"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

// @title Assignment API
// @version 1.0
// @description Async job processing REST API with idempotent job creation, outbox pattern, and JWT authentication.
// @termsOfService https://github.com/Heleo2705/assignment

// @contact.name API Support
// @contact.url https://github.com/Heleo2705/assignment

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8000
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer <token>" — get a token from POST /register or /login

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

	auth := appmiddleware.NewJWTAuth(jwtSecret)

	r := chi.NewRouter()
	r.Use(appmiddleware.RequestLogger(logger))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusFound)
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	h.RegisterRoutes(r, auth)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	logger.Info("starting app", zap.String("addr", ":"+port))
	logger.Info("swagger UI at http://localhost:" + port + "/swagger/index.html")
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}
