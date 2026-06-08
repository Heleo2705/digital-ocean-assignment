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

	r := chi.NewRouter()
	r.Use(appmiddleware.RequestLogger(logger))
	h.RegisterRoutes(r)

	logger.Info("starting app", zap.String("addr", ":8000"))
	if err := http.ListenAndServe(":8000", r); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}
