package domainerror_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/provsenya/avito/internal/domain/trip"
	"github.com/provsenya/avito/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{trip.ErrDriverBusy, 409, "driver_busy"}, {trip.ErrCompleted, 409, "trip_completed"}, {trip.ErrNotFound, 404, "trip_not_found"}, {&trip.ValidationError{Field: "price", Message: "negative"}, 400, "invalid_request"}, {errors.New("secret database password"), 500, "internal_error"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			status, body := domainerror.Problem(fmt.Errorf("wrapped: %w", tc.err))
			if status != tc.status || body.Code != tc.code || int(body.Status) != tc.status {
				t.Fatalf("status=%d body=%+v", status, body)
			}
			if body.Detail == nil || strings.Contains(*body.Detail, "secret") {
				t.Fatalf("unsafe detail: %+v", body.Detail)
			}
		})
	}
}
