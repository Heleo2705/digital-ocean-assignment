package main

import (
	"os"

	"github.com/Heleo2705/assignment/db"
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

	logger.Info("starting worker")
	// Worker implementation will be added here.
}
