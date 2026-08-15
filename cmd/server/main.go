package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/krishnabhardwaj25/flowithgo/internal/api"
	"github.com/krishnabhardwaj25/flowithgo/internal/db"
	"github.com/krishnabhardwaj25/flowithgo/internal/logger"
	"github.com/krishnabhardwaj25/flowithgo/internal/store"
	"github.com/krishnabhardwaj25/flowithgo/internal/worker"
)

func main() {
	logger.Init()
    
    // load .env if present (local dev), ignore error in production
    godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.L.Error("DATABASE_URL is not set")
		os.Exit(1)
	}

	database, err := db.Connect(databaseURL)
	if err != nil {
		logger.L.Error("could not connect to database", "error", err.Error())
		os.Exit(1)
	}
	defer database.Close()

	jobStore := store.NewJobStore(database)

	server := api.NewServer(jobStore)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := worker.NewPool(5, jobStore, server.GetBroadcaster())
	pool.Start(ctx)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: server,
	}

	go func() {
		logger.L.Info("server starting", "port", 8080)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L.Error("server failed", "error", err.Error())
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.L.Info("shutdown signal received")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.L.Error("server shutdown failed", "error", err.Error())
	}

	pool.Wait()
	logger.L.Info("shutdown complete")
}