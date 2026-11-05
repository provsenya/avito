package domainerror

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/provsenya/avito/api"
	"github.com/provsenya/avito/internal/domain/trip"
	"github.com/provsenya/avito/internal/http/requeststate"
)

func Problem(err error) (int, api.Problem) {
	status, code, title, detail := http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error"
	var validation *trip.ValidationError
	switch {
	case errors.As(err, &validation):
		status, code, title, detail = http.StatusBadRequest, "invalid_request", "Invalid request", validation.Error()
	case errors.Is(err, trip.ErrNotFound):
		status, code, title, detail = http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found"
	case errors.Is(err, trip.ErrCompleted):
		status, code, title, detail = http.StatusConflict, "trip_completed", "Trip completed", "Operation is not allowed for a completed trip"
	case errors.Is(err, trip.ErrDriverBusy):
		status, code, title, detail = http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip"
	}
	typeID := strings.ReplaceAll(code, "_", "-")
	return status, api.Problem{Type: "https://tripgo.example/problems/" + typeID, Title: title, Status: int32(status), Code: code, Detail: &detail}
}

func Write(w http.ResponseWriter, r *http.Request, err error) {
	status, body := Problem(err)
	if state := requeststate.From(r.Context()); state != nil {
		state.Err = err
	}
	span := trace.SpanFromContext(r.Context())
	span.RecordError(err)
	span.SetStatus(codes.Error, body.Title)
	span.SetAttributes(attribute.String("error.type", body.Code))
	instance := r.URL.Path
	body.Instance = &instance
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
