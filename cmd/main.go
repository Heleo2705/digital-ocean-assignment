package main

import (
	"net/http"

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

	r := chi.NewRouter()
	r.Use(appmiddleware.RequestLogger(logger))
	handler.RegisterRoutes(r)

	logger.Info("starting app", zap.String("addr", ":8000"))
	if err := http.ListenAndServe(":8000", r); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}
