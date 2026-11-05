package finish_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/provsenya/avito/internal/domain/trip"
	"github.com/provsenya/avito/internal/model"
	"github.com/provsenya/avito/internal/usecase/trip/finish"
)

func TestExecute(t *testing.T) {
	failure := errors.New("repository failure")
	for _, tc := range []struct {
		name                 string
		status               trip.Status
		fetch, save, history error
	}{
		{name: "success", status: trip.StatusActive}, {name: "completed", status: trip.StatusCompleted}, {name: "missing", fetch: trip.ErrNotFound}, {name: "save error", status: trip.StatusActive, save: failure}, {name: "history error", status: trip.StatusActive, history: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			history := NewMockstatusHistory(ctrl)
			tx := NewMocktxManager(ctrl)
			events := NewMockevents(ctrl)
			id := uuid.New()
			tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
			repo.EXPECT().GetForUpdate(gomock.Any(), id).Return(model.Trip{ID: id, Status: tc.status}, tc.fetch)
			want := tc.fetch
			if want == nil && tc.status != trip.StatusActive {
				want = trip.ErrCompleted
			}
			if want == nil {
				repo.EXPECT().Finish(gomock.Any(), id, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, at time.Time) (model.Trip, error) {
					return model.Trip{ID: id, Status: trip.StatusCompleted, FinishedAt: &at}, tc.save
				})
				want = tc.save
				if want == nil {
					history.EXPECT().Add(gomock.Any(), id, gomock.Any(), trip.StatusCompleted, nil).Return(tc.history)
					want = tc.history
				}
			}
			if want == nil {
				events.EXPECT().Record(gomock.Any(), "completed")
			}
			u := finish.New(repo, history, tx, slog.New(slog.NewTextHandler(io.Discard, nil)), events)
			value, err := u.Execute(context.Background(), id)
			if !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
			if want == nil && (value.Status != trip.StatusCompleted || value.FinishedAt == nil) {
				t.Fatalf("trip=%+v", value)
			}
		})
	}
}
