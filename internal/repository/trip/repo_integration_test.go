//go:build integration

package triprepo_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/provsenya/avito/internal/domain/trip"
	"github.com/provsenya/avito/internal/infra/postgres/transaction"
	triprepo "github.com/provsenya/avito/internal/repository/trip"
)

func database(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is required")
	}
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	for _, name := range []string{"00001_create_trips.sql", "00002_create_trip_status_history.sql"} {
		raw, err := os.ReadFile("../../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, strings.Split(string(raw), "-- +goose Down")[0]); err != nil {
			t.Fatal(err)
		}
	}
	return pool
}

func value() trip.Trip {
	return trip.Trip{ID: uuid.New(), UserID: uuid.New(), DriverID: uuid.New(), Status: trip.StatusActive, StartedAt: time.Now().UTC()}
}

func TestRepository(t *testing.T) {
	pool := database(t)
	repo := triprepo.NewTripRepository(pool, time.Second*5)
	ctx := context.Background()
	v := value()
	if err := repo.Create(ctx, v); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, v.ID)
	if err != nil || got.ID != v.ID || got.DriverID != v.DriverID {
		t.Fatalf("trip=%+v err=%v", got, err)
	}
	duplicate := v
	duplicate.ID = uuid.New()
	if err := repo.Create(ctx, duplicate); !errors.Is(err, trip.ErrDriverBusy) {
		t.Fatalf("error=%v", err)
	}
	if _, err := repo.GetByID(ctx, uuid.New()); !errors.Is(err, trip.ErrNotFound) {
		t.Fatalf("error=%v", err)
	}
	completed, err := repo.Finish(ctx, v.ID, time.Now().UTC())
	if err != nil || completed.Status != trip.StatusCompleted || completed.FinishedAt == nil {
		t.Fatalf("trip=%+v err=%v", completed, err)
	}
	if _, err := repo.Finish(ctx, v.ID, time.Now().UTC()); !errors.Is(err, trip.ErrCompleted) {
		t.Fatalf("error=%v", err)
	}
	if _, err := repo.Finish(ctx, uuid.New(), time.Now().UTC()); !errors.Is(err, trip.ErrNotFound) {
		t.Fatalf("error=%v", err)
	}
}

func TestAtomicRollback(t *testing.T) {
	pool := database(t)
	ctx := context.Background()
	repo := triprepo.NewTripRepository(pool, time.Second*5)
	history := triprepo.NewStatusHistoryRepository(pool, time.Second*5)
	tx := transaction.NewTxManager(pool, time.Second*5)
	v := value()
	if _, err := pool.Exec(ctx, "ALTER TABLE trip_status_history ADD CONSTRAINT forced_failure CHECK (to_status <> 'active')"); err != nil {
		t.Fatal(err)
	}
	err := tx.Do(ctx, func(ctx context.Context) error {
		if err := repo.Create(ctx, v); err != nil {
			return err
		}
		return history.Add(ctx, v.ID, nil, trip.StatusActive, nil)
	})
	if err == nil {
		t.Fatal("expected second insert failure")
	}
	if _, err := repo.GetByID(ctx, v.ID); !errors.Is(err, trip.ErrNotFound) {
		t.Fatalf("orphan trip: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM trip_status_history").Scan(&count); err != nil || count != 0 {
		t.Fatalf("history count=%d err=%v", count, err)
	}
}

func TestNestedAndPanic(t *testing.T) {
	pool := database(t)
	ctx := context.Background()
	repo := triprepo.NewTripRepository(pool, time.Second*5)
	tx := transaction.NewTxManager(pool, time.Second*5)
	v := value()
	func() {
		defer func() {
			if recover() == nil {
				t.Error("panic was swallowed")
			}
		}()
		_ = tx.Do(ctx, func(ctx context.Context) error {
			return tx.Do(ctx, func(ctx context.Context) error {
				if err := repo.Create(ctx, v); err != nil {
					t.Fatal(err)
				}
				panic("rollback")
			})
		})
	}()
	if _, err := repo.GetByID(ctx, v.ID); !errors.Is(err, trip.ErrNotFound) {
		t.Fatalf("orphan trip: %v", err)
	}
	if err := tx.Do(ctx, func(ctx context.Context) error {
		return tx.Do(ctx, func(ctx context.Context) error { return repo.Create(ctx, v) })
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByID(ctx, v.ID); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentFinish(t *testing.T) {
	pool := database(t)
	repo := triprepo.NewTripRepository(pool, time.Second*5)
	ctx := context.Background()
	v := value()
	if err := repo.Create(ctx, v); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() { _, err := repo.Finish(ctx, v.ID, time.Now().UTC()); results <- err })
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, trip.ErrCompleted):
			conflict++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
}

func TestBeginTimeout(t *testing.T) {
	pool := database(t)
	ctx := context.Background()
	for range pool.Config().MaxConns {
		conn, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Release()
	}
	manager := transaction.NewTxManager(pool, 20*time.Millisecond)
	err := manager.Do(ctx, func(context.Context) error { t.Error("transaction started without a free connection"); return nil })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v", err)
	}
}
