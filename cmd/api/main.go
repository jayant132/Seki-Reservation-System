package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/jayant132/seki/internal/auth"
	"github.com/jayant132/seki/internal/config"
	"github.com/jayant132/seki/internal/db"
	"github.com/jayant132/seki/internal/handler"
	"github.com/jayant132/seki/internal/tracing"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// .env is optional — in Docker/prod, real environment variables are
	// used instead. This just makes local `go run` convenient.
	_ = godotenv.Load()

	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Tracing failures are logged but never fatal — a demo/dev environment
	// without a Jaeger collector running should still serve traffic, just
	// without traces. Production deployments should treat this as required.
	shutdownTracing, err := tracing.Init(ctx, "seki-api", cfg.OTLPEndpoint)
	if err != nil {
		logger.Warn("tracing disabled: failed to initialize exporter", slog.String("error", err.Error()))
		shutdownTracing = func(context.Context) error { return nil }
	} else {
		logger.Info("tracing initialized", slog.String("otlp_endpoint", cfg.OTLPEndpoint))
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(shutdownCtx); err != nil {
			logger.Warn("tracing shutdown error", slog.String("error", err.Error()))
		}
	}()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to postgres")

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to redis", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer redisClient.Close()
	logger.Info("connected to redis")

	issuer := auth.NewIssuer(cfg.JWTSecret)

	router := handler.NewRouter(handler.Dependencies{
		Pool:   pool,
		Redis:  redisClient,
		Issuer: issuer,
		Logger: logger,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("server starting", slog.String("port", cfg.Port), slog.String("env", cfg.Env))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received, draining connections")

	// Graceful shutdown: stop accepting new requests but let in-flight
	// ones finish (up to 15s) before the process exits. This is what
	// prevents a rolling deploy from cutting off a request mid-transaction.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
	}
	logger.Info("server stopped cleanly")
}
