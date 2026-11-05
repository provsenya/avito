package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/provsenya/avito/internal/trip"
)

const (
	activeTripPerDriverConstraint = "trips_one_active_per_driver_idx"
	uniqueViolationCode           = "23505"
	tripColumnList                = `id, user_id, driver_id,
		start_latitude, start_longitude,
		end_latitude, end_longitude,
		price, status, started_at, finished_at`
)

type TripRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
	statement    sq.StatementBuilderType
}

func NewTripRepository(pool *pgxpool.Pool, queryTimeout time.Duration) *TripRepository {
	return &TripRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
		statement:    sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *TripRepository) Create(ctx context.Context, value trip.Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := r.statement.
		Insert("trips").
		Columns(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
			"finished_at",
		).
		Values(
			value.ID,
			value.UserID,
			value.DriverID,
			value.StartPoint.Latitude,
			value.StartPoint.Longitude,
			value.EndPoint.Latitude,
			value.EndPoint.Longitude,
			value.Price,
			value.Status,
			value.StartedAt,
			value.FinishedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip query: %w", err)
	}

	_, err = executorFromContext(ctx, r.pool).Exec(ctx, query, args...)
	if isDriverBusyViolation(err) {
		return fmt.Errorf("insert trip: %w", trip.ErrDriverBusy)
	}
	if err != nil {
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *TripRepository) GetByID(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	executor := executorFromContext(ctx, r.pool)
	return r.getByID(ctx, executor, id)
}

func (r *TripRepository) Finish(
	ctx context.Context,
	id uuid.UUID,
	finishedAt time.Time,
) (trip.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := r.statement.
		Update("trips").
		Set("status", trip.StatusCompleted).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(sq.Eq{
			"id":     id,
			"status": trip.StatusActive,
		}).
		Suffix("RETURNING " + tripColumnList).
		ToSql()
	if err != nil {
		return trip.Trip{}, fmt.Errorf("build finish trip query: %w", err)
	}

	executor := executorFromContext(ctx, r.pool)
	value, err := scanTrip(executor.QueryRow(ctx, query, args...))
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return trip.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	_, lookupErr := r.getByID(ctx, executor, id)
	if errors.Is(lookupErr, trip.ErrNotFound) {
		return trip.Trip{}, trip.ErrNotFound
	}
	if lookupErr != nil {
		return trip.Trip{}, fmt.Errorf("check trip after unsuccessful finish: %w", lookupErr)
	}
	return trip.Trip{}, trip.ErrCompleted
}

func (r *TripRepository) getByID(
	ctx context.Context,
	executor Executor,
	id uuid.UUID,
) (trip.Trip, error) {
	query, args, err := r.statement.
		Select(tripColumnList).
		From("trips").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return trip.Trip{}, fmt.Errorf("build get trip query: %w", err)
	}

	value, err := scanTrip(executor.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return trip.Trip{}, trip.ErrNotFound
	}
	if err != nil {
		return trip.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return value, nil
}

func scanTrip(row pgx.Row) (trip.Trip, error) {
	var value trip.Trip
	var status string

	err := row.Scan(
		&value.ID,
		&value.UserID,
		&value.DriverID,
		&value.StartPoint.Latitude,
		&value.StartPoint.Longitude,
		&value.EndPoint.Latitude,
		&value.EndPoint.Longitude,
		&value.Price,
		&status,
		&value.StartedAt,
		&value.FinishedAt,
	)
	if err != nil {
		return trip.Trip{}, err
	}

	value.Status = trip.Status(status)
	return value, nil
}

func isDriverBusyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == uniqueViolationCode &&
		pgErr.ConstraintName == activeTripPerDriverConstraint
}
