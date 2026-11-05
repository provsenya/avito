package create

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/provsenya/avito/internal/domain/trip"
	"github.com/provsenya/avito/internal/model"
)

type Usecase struct {
	trips   tripRepository
	history statusHistory
	tx      txManager
	logger  *slog.Logger
	events  events
}

func New(trips tripRepository, history statusHistory, tx txManager, logger *slog.Logger, events events) *Usecase {
	return &Usecase{trips: trips, history: history, tx: tx, logger: logger, events: events}
}

func (u *Usecase) Execute(ctx context.Context, input model.CreateTrip) (model.Trip, error) {
	if err := input.Validate(); err != nil {
		return model.Trip{}, err
	}
	value := model.Trip{ID: uuid.New(), UserID: input.UserID, DriverID: input.DriverID, StartPoint: input.StartPoint, EndPoint: input.EndPoint, Price: input.Price, Status: trip.StatusActive, StartedAt: time.Now().UTC()}
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		if err := u.trips.Create(ctx, value); err != nil {
			return fmt.Errorf("save trip: %w", err)
		}
		if err := u.history.Add(ctx, value.ID, nil, trip.StatusActive, nil); err != nil {
			return fmt.Errorf("save initial status: %w", err)
		}
		return nil
	})
	if err != nil {
		return model.Trip{}, fmt.Errorf("create trip: %w", err)
	}
	u.events.Record(ctx, "created")
	u.logger.InfoContext(ctx, "trip created", "trip_id", value.ID)
	return value, nil
}
