package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/provsenya/avito/internal/domain/trip"
	"github.com/provsenya/avito/internal/model"
	"github.com/provsenya/avito/internal/usecase/trip/get"
)

func TestExecute(t *testing.T) {
	for _, want := range []error{nil, trip.ErrNotFound, errors.New("database failure")} {
		ctrl := gomock.NewController(t)
		repo := NewMocktripRepository(ctrl)
		id := uuid.New()
		value := model.Trip{ID: id}
		repo.EXPECT().GetByID(gomock.Any(), id).Return(value, want)
		got, err := get.New(repo).Execute(context.Background(), id)
		if !errors.Is(err, want) {
			t.Fatalf("error=%v", err)
		}
		if want == nil && got.ID != id {
			t.Fatalf("id=%v", got.ID)
		}
	}
}
