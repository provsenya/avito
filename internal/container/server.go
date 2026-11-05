package container

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/provsenya/avito/internal/config"
)

func newHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
}

func (c *Container) serve(ctx context.Context) error {
	serverErr := make(chan error, 1)
	go func() { serverErr <- c.server.ListenAndServe() }()
	c.logger.Info("HTTP server started", "addr", c.server.Addr)
	var serveErr error
	select {
	case serveErr = <-serverErr:
	case <-ctx.Done():
		c.logger.Info("shutdown requested")
	}
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.timeout)
	defer cancel()
	err := c.server.Shutdown(shutdownCtx)
	if err != nil {
		err = errors.Join(err, c.server.Close())
	}
	cleanup := make(chan error, 1)
	go func() { c.pool.Close(); cleanup <- c.telemetry.Shutdown(shutdownCtx) }()
	select {
	case cleanupErr := <-cleanup:
		err = errors.Join(err, cleanupErr)
	case <-shutdownCtx.Done():
		err = errors.Join(err, shutdownCtx.Err())
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		err = errors.Join(err, fmt.Errorf("serve HTTP: %w", serveErr))
	}
	if err != nil {
		c.logger.Error("shutdown failed", "error", err)
		return err
	}
	c.logger.Info("HTTP server stopped")
	return nil
}
