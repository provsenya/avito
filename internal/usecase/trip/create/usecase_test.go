package create_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/provsenya/avito/internal/domain/trip"
	"github.com/provsenya/avito/internal/model"
	"github.com/provsenya/avito/internal/usecase/trip/create"
)

func TestExecute(t *testing.T) {
	failure := errors.New("repository failure")
	for _, tc := range []struct {
		name              string
		save, history, tx error
		invalid           bool
	}{
		{name: "success"}, {name: "busy", save: trip.ErrDriverBusy}, {name: "repository", save: failure}, {name: "history", history: failure}, {name: "commit", tx: failure}, {name: "invalid", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			history := NewMockstatusHistory(ctrl)
			tx := NewMocktxManager(ctrl)
			events := NewMockevents(ctrl)
			input := model.CreateTrip{UserID: uuid.New(), DriverID: uuid.New(), Price: 10}
			want := tc.save
			if want == nil {
				want = tc.history
			}
			if want == nil {
				want = tc.tx
			}
			if tc.invalid {
				input.Price = -1
			} else {
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					if err := fn(ctx); err != nil {
						return err
					}
					return tc.tx
				})
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(tc.save)
				if tc.save == nil {
					history.EXPECT().Add(gomock.Any(), gomock.Any(), nil, trip.StatusActive, nil).Return(tc.history)
				}
				if want == nil {
					events.EXPECT().Record(gomock.Any(), "created")
				}
			}
			u := create.New(repo, history, tx, slog.New(slog.NewTextHandler(io.Discard, nil)), events)
			value, err := u.Execute(context.Background(), input)
			if tc.invalid {
				var validation *trip.ValidationError
				if !errors.As(err, &validation) {
					t.Fatalf("error=%v", err)
				}
				return
			}
			if !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
			if want == nil && (value.ID == uuid.Nil || value.DriverID != input.DriverID || value.Status != trip.StatusActive || value.StartedAt.IsZero()) {
				t.Fatalf("trip=%+v", value)
			}
		})
	}
}
