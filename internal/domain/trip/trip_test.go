package trip_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/provsenya/avito/internal/domain/trip"
)

func TestFinish(t *testing.T) {
	at := time.Now().UTC()
	for _, tc := range []struct {
		name   string
		status trip.Status
		want   error
	}{{"active", trip.StatusActive, nil}, {"completed", trip.StatusCompleted, trip.ErrCompleted}} {
		t.Run(tc.name, func(t *testing.T) {
			value := trip.Trip{Status: tc.status}
			err := value.Finish(at)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v", err)
			}
			if tc.want == nil && (value.Status != trip.StatusCompleted || value.FinishedAt == nil || !value.FinishedAt.Equal(at)) {
				t.Fatalf("invalid transition: %+v", value)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		change      func(*trip.CreateInput)
	}{
		{"valid", "", func(*trip.CreateInput) {}},
		{"user", "user_id", func(v *trip.CreateInput) { v.UserID = uuid.Nil }},
		{"driver", "driver_id", func(v *trip.CreateInput) { v.DriverID = uuid.Nil }},
		{"latitude", "start_point.latitude", func(v *trip.CreateInput) { v.StartPoint.Latitude = 91 }},
		{"longitude", "end_point.longitude", func(v *trip.CreateInput) { v.EndPoint.Longitude = -181 }},
		{"nan", "start_point.latitude", func(v *trip.CreateInput) { v.StartPoint.Latitude = math.NaN() }},
		{"price", "price", func(v *trip.CreateInput) { v.Price = -1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := trip.CreateInput{UserID: uuid.New(), DriverID: uuid.New()}
			tc.change(&value)
			err := value.Validate()
			if tc.field == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var validation *trip.ValidationError
			if !errors.As(err, &validation) || validation.Field != tc.field {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
