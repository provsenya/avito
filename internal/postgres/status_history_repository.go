package postgres

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/provsenya/avito/internal/trip"
)

type StatusHistoryRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
	statement    sq.StatementBuilderType
}

func NewStatusHistoryRepository(
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
) *StatusHistoryRepository {
	return &StatusHistoryRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
		statement:    sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *StatusHistoryRepository) Add(
	ctx context.Context,
	tripID uuid.UUID,
	from *trip.Status,
	to trip.Status,
	reason *string,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	var fromValue any
	if from != nil {
		fromValue = *from
	}

	var reasonValue any
	if reason != nil {
		reasonValue = *reason
	}

	query, args, err := r.statement.
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(tripID, fromValue, to, reasonValue).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip status history query: %w", err)
	}

	_, err = executorFromContext(ctx, r.pool).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert trip status history: %w", err)
	}
	return nil
}
