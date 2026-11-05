package finish

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

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	var value model.Trip
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		var err error
		value, err = u.trips.GetForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("lock trip: %w", err)
		}
		at := time.Now().UTC()
		if err := value.Finish(at); err != nil {
			return err
		}
		value, err = u.trips.Finish(ctx, id, at)
		if err != nil {
			return fmt.Errorf("save completed trip: %w", err)
		}
		from := trip.StatusActive
		if err := u.history.Add(ctx, id, &from, trip.StatusCompleted, nil); err != nil {
			return fmt.Errorf("save completed status: %w", err)
		}
		return nil
	})
	if err != nil {
		return model.Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	u.events.Record(ctx, "completed")
	u.logger.InfoContext(ctx, "trip completed", "trip_id", value.ID)
	return value, nil
}
