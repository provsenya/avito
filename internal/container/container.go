package container

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/provsenya/avito/internal/config"
	httpapi "github.com/provsenya/avito/internal/http"
	"github.com/provsenya/avito/internal/http/middleware"
	createhttp "github.com/provsenya/avito/internal/http/trip/create"
	finishhttp "github.com/provsenya/avito/internal/http/trip/finish"
	gethttp "github.com/provsenya/avito/internal/http/trip/get"
	"github.com/provsenya/avito/internal/infra/postgres"
	"github.com/provsenya/avito/internal/infra/postgres/transaction"
	"github.com/provsenya/avito/internal/infra/telemetry"
	triprepo "github.com/provsenya/avito/internal/repository/trip"
	"github.com/provsenya/avito/internal/usecase/trip/create"
	"github.com/provsenya/avito/internal/usecase/trip/finish"
	"github.com/provsenya/avito/internal/usecase/trip/get"
)

type Container struct {
	server    *http.Server
	pool      *pgxpool.Pool
	telemetry *telemetry.Telemetry
	timeout   time.Duration
	logger    *slog.Logger
}

func New(ctx context.Context, cfg config.Config) (*Container, error) {
	telemetry, err := telemetry.New(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
		defer cancel()
		return nil, errors.Join(err, telemetry.Shutdown(shutdownCtx))
	}
	tx := transaction.NewTxManager(pool, cfg.Database.QueryTimeout)
	trips := triprepo.NewTripRepository(pool, cfg.Database.QueryTimeout)
	history := triprepo.NewStatusHistoryRepository(pool, cfg.Database.QueryTimeout)
	c := create.New(trips, history, tx, telemetry.Logger, telemetry)
	g := get.New(trips)
	f := finish.New(trips, history, tx, telemetry.Logger, telemetry)
	router := httpapi.New(createhttp.New(c), gethttp.New(g), finishhttp.New(f), pool, cfg.Database.QueryTimeout)
	middleware, err := middleware.New(telemetry.Logger, telemetry.Metrics.Meter("trip-service"))
	if err != nil {
		pool.Close()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
		defer cancel()
		return nil, errors.Join(err, telemetry.Shutdown(shutdownCtx))
	}
	return &Container{server: newHTTPServer(cfg, middleware.Wrap(router)), pool: pool, telemetry: telemetry, timeout: cfg.ShutdownTimeout, logger: telemetry.Logger}, nil
}

func (c *Container) Run(ctx context.Context) error { return c.serve(ctx) }
