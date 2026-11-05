package trip

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TransactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Repository interface {
	Create(ctx context.Context, value Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (Trip, error)
	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (Trip, error)
}

type StatusHistoryRepository interface {
	Add(
		ctx context.Context,
		tripID uuid.UUID,
		from *Status,
		to Status,
		reason *string,
	) error
}

type CreateInput struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
}

type Service struct {
	txManager TransactionManager
	trips     Repository
	history   StatusHistoryRepository
}

func NewService(
	txManager TransactionManager,
	trips Repository,
	history StatusHistoryRepository,
) *Service {
	return &Service{
		txManager: txManager,
		trips:     trips,
		history:   history,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Trip, error) {
	value := Trip{
		ID:         uuid.New(),
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     StatusActive,
		StartedAt:  time.Now().UTC(),
	}

	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		if err := s.trips.Create(ctx, value); err != nil {
			return fmt.Errorf("create trip: %w", err)
		}

		if err := s.history.Add(ctx, value.ID, nil, StatusActive, nil); err != nil {
			return fmt.Errorf("add initial trip status: %w", err)
		}

		return nil
	})
	if err != nil {
		return Trip{}, fmt.Errorf("create trip transaction: %w", err)
	}

	return value, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Trip, error) {
	value, err := s.trips.GetByID(ctx, id)
	if err != nil {
		return Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return value, nil
}

func (s *Service) Finish(ctx context.Context, id uuid.UUID) (Trip, error) {
	finishedAt := time.Now().UTC()
	var value Trip

	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		var err error
		value, err = s.trips.Finish(ctx, id, finishedAt)
		if err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}

		from := StatusActive
		if err := s.history.Add(ctx, id, &from, StatusCompleted, nil); err != nil {
			return fmt.Errorf("add completed trip status: %w", err)
		}

		return nil
	})
	if err != nil {
		return Trip{}, fmt.Errorf("finish trip transaction: %w", err)
	}

	return value, nil
}
