package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kosench/ecommerce-lab/internal/handler"
	"github.com/Kosench/ecommerce-lab/internal/repository"
	"github.com/Kosench/ecommerce-lab/internal/service"
	"github.com/Kosench/ecommerce-lab/platform/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func main() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	logr, err := logger.New(env)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logr.(*logger.ZapLogger).Sync()

	dbURL := "postgres://postgres:postgres@localhost:5432/orderdb?sslmode=disable"
	httpAddr := ":8080"

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	logr.Info("database connected",
		zap.String("url", dbURL),
	)

	healthHandler := handler.NewHealthHandler(pool, logr)

	readyCtx, readyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readyCancel()

	if err := pool.Ping(readyCtx); err != nil {
		logr.Fatal("database is not ready",
			zap.Error(err),
		)
	}

	logr.Info("database is ready")

	orderRepo := repository.NewOrderRepository(pool, logr)
	orderService := service.NewOrderService(orderRepo, logr)
	orderHandler := handler.NewOrderHandler(orderService, logr)

	mux := http.NewServeMux()

	// Health endpoints
	mux.HandleFunc("GET /health", healthHandler.Liveness)
	mux.HandleFunc("GET /ready", healthHandler.Readiness)

	// Business endpoints
	mux.HandleFunc("POST /orders", orderHandler.CreateOrder)

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logr.Info("server starting",
			zap.String("addr", httpAddr),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logr.Fatal("server failed",
				zap.Error(err),
			)
		}
	}()

	<-done
	logr.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logr.Fatal("server shutdown failed",
			zap.Error(err),
		)
	}

	logr.Info("server stopped")
}
