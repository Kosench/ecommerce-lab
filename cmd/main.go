package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kosench/ecommerce-lab/internal/config"
	"github.com/Kosench/ecommerce-lab/internal/handler"
	"github.com/Kosench/ecommerce-lab/internal/middleware/httpmw"
	"github.com/Kosench/ecommerce-lab/internal/repository"
	"github.com/Kosench/ecommerce-lab/internal/service"
	"github.com/Kosench/ecommerce-lab/platform/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func main() {
	cfg := config.MustLoad()

	logr, err := logger.New(cfg.Environment)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logr.(*logger.ZapLogger).Sync()

	poolConfig, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		logr.Fatal("failed to parse database url",
			zap.Error(err),
		)
	}

	poolConfig.MaxConns = int32(cfg.Database.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.Database.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.Database.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		logr.Fatal("failed to connect to db",
			zap.Error(err),
		)
	}
	defer pool.Close()

	logr.Info("database connected",
		zap.String("url", cfg.Database.URL),
		zap.Int("max_open_conns", cfg.Database.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.Database.MaxIdleConns),
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

	handlerWithMiddleware := httpmw.Recovery(
		httpmw.Logging(mux, logr),
		logr,
	)

	server := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      handlerWithMiddleware,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logr.Info("server starting",
			zap.String("addr", cfg.Server.Addr),
			zap.String("env", cfg.Environment),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logr.Fatal("server failed",
				zap.Error(err),
			)
		}
	}()

	<-done
	logr.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logr.Fatal("server shutdown failed",
			zap.Error(err),
		)
	}

	logr.Info("server stopped")
}
