package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Manager struct {
	pool           *pgxpool.Pool
	options        pgx.TxOptions
	cleanupTimeout time.Duration
}

func NewTxManager(pool *pgxpool.Pool, cleanupTimeout time.Duration) *Manager {
	return &Manager{
		pool: pool,
		options: pgx.TxOptions{
			IsoLevel: pgx.ReadCommitted,
		},
		cleanupTimeout: cleanupTimeout,
	}
}

func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := transactionFromContext(ctx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.BeginTx(ctx, m.options)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txCtx := contextWithTransaction(ctx, tx)

	defer func() {
		if recovered := recover(); recovered != nil {
			_ = m.rollback(ctx, tx)
			panic(recovered)
		}

		if err == nil {
			return
		}

		if rollbackErr := m.rollback(ctx, tx); rollbackErr != nil {
			err = errors.Join(err, rollbackErr)
		}
	}()

	if err = fn(txCtx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (m *Manager) rollback(ctx context.Context, tx pgx.Tx) error {
	rollbackCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		m.cleanupTimeout,
	)
	defer cancel()

	if err := tx.Rollback(rollbackCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("rollback transaction: %w", err)
	}
	return nil
}
