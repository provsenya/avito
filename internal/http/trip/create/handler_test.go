package create_test

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/provsenya/avito/api"
	"github.com/provsenya/avito/internal/domain/trip"
	"github.com/provsenya/avito/internal/http/trip/create"
	"github.com/provsenya/avito/internal/model"
)

func TestHandler(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{name: "success", status: 201}, {name: "busy", err: trip.ErrDriverBusy, status: 409}, {name: "internal", err: errors.New("secret"), status: 500}, {name: "invalid", status: 400, body: `{"extra":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			u := NewMockusecase(ctrl)
			body := tc.body
			if body == "" {
				body = `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":0,"longitude":0},"end_point":{"latitude":0,"longitude":0},"price":0}`
				u.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(model.Trip{ID: uuid.New()}, tc.err)
			}
			r := httptest.NewRequest("POST", "/api/v1/trips", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			create.New(u).CreateTrip(w, r, api.CreateTripParams{})
			if tc.status >= 400 {
				expected := map[int]string{400: "invalid_request", 404: "trip_not_found", 409: "driver_busy", 500: "internal_error"}[tc.status]
				if w.Header().Get("Content-Type") != "application/problem+json" || !strings.Contains(w.Body.String(), `"code":"`+expected+`"`) {
					t.Fatalf("problem=%s", w.Body.String())
				}
			}
			if w.Code != tc.status || strings.Contains(w.Body.String(), "secret") {
				t.Fatalf("response=%d %s", w.Code, w.Body.String())
			}
			if tc.status == 201 && w.Header().Get("Location") == "" {
				t.Fatal("missing location")
			}
		})
	}
}
