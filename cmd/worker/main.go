package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Heleo2705/assignment/db"
	"github.com/Heleo2705/assignment/service"
	"github.com/hibiken/asynq"
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

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		logger.Fatal("REDIS_ADDR is required")
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		parsed, err := strconv.Atoi(dbStr)
		if err != nil {
			logger.Fatal("REDIS_DB must be an integer", zap.Error(err))
		}
		redisDB = parsed
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

	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	}

	client := asynq.NewClient(redisOpt)
	defer client.Close()

	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"default": 10,
		},
	})

	mux := asynq.NewServeMux()
	mux.HandleFunc(service.TypeJobProcess, func(ctx context.Context, task *asynq.Task) error {
		return service.ProcessJobTask(ctx, store, task, logger)
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go service.StartOutboxPollers(ctx, store, client, logger, 2, 10, 5*time.Second)

	logger.Info("starting worker", zap.String("redis_addr", redisAddr))
	if err := server.Run(mux); err != nil {
		logger.Fatal("asynq server failed", zap.Error(err))
	}
}
