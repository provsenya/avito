package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/provsenya/avito/internal/config"
	"github.com/provsenya/avito/internal/httpapi"
	"github.com/provsenya/avito/internal/postgres"
	"github.com/provsenya/avito/internal/trip"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	txManager := postgres.NewTxManager(pool, cfg.Database.QueryTimeout)
	tripRepository := postgres.NewTripRepository(pool, cfg.Database.QueryTimeout)
	historyRepository := postgres.NewStatusHistoryRepository(pool, cfg.Database.QueryTimeout)
	tripService := trip.NewService(txManager, tripRepository, historyRepository)

	handler := httpapi.NewHandler(pool, tripService, cfg.Database.QueryTimeout)
	router := httpapi.NewRouter(handler)
	server := newHTTPServer(cfg, router)

	return serve(ctx, server, cfg.ShutdownTimeout, logger)
}
