package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/provsenya/avito/internal/config"
	"github.com/provsenya/avito/internal/container"
)

func main() {
	if err := run(); err != nil {
		slog.New(slog.NewJSONHandler(os.Stdout, nil)).Error("service stopped", "service", "trip-service", "trace_id", "", "span_id", "", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	app, err := container.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	return app.Run(ctx)
}
