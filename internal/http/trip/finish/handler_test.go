package finish_test

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/provsenya/avito/internal/domain/trip"
	"github.com/provsenya/avito/internal/http/trip/finish"
	"github.com/provsenya/avito/internal/model"
)

func TestHandler(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{{nil, 200}, {trip.ErrCompleted, 409}, {errors.New("secret"), 500}} {
		ctrl := gomock.NewController(t)
		u := NewMockusecase(ctrl)
		id := uuid.New()
		u.EXPECT().Execute(gomock.Any(), id).Return(model.Trip{ID: id}, tc.err)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/v1/trips/"+id.String(), nil)
		finish.New(u).FinishTrip(w, r, id)
		if tc.status >= 400 {
			expected := map[int]string{400: "invalid_request", 404: "trip_not_found", 409: "trip_completed", 500: "internal_error"}[tc.status]
			if w.Header().Get("Content-Type") != "application/problem+json" || !strings.Contains(w.Body.String(), `"code":"`+expected+`"`) {
				t.Fatalf("problem=%s", w.Body.String())
			}
		}
		if w.Code != tc.status || strings.Contains(w.Body.String(), "secret") {
			t.Fatalf("response=%d %s", w.Code, w.Body.String())
		}
	}
}
