package httpapi

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/provsenya/avito/api"
	"github.com/provsenya/avito/internal/trip"
)

func (h *Handler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	_ api.CreateTripParams,
) {
	request, err := decodeCreateTripRequest(w, r)
	if err != nil {
		writeProblem(w, r, invalidRequestProblem)
		return
	}

	if err := validateCreateTripRequest(request); err != nil {
		writeProblem(w, r, invalidRequestProblem)
		return
	}

	value, err := h.trips.Create(r.Context(), trip.CreateInput{
		UserID:     request.UserId,
		DriverID:   request.DriverId,
		StartPoint: toDomainCoordinates(request.StartPoint),
		EndPoint:   toDomainCoordinates(request.EndPoint),
		Price:      request.Price,
	})
	if errors.Is(err, trip.ErrDriverBusy) {
		writeProblem(w, r, driverBusyProblem)
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/api/v1/trips/%s", value.ID))
	writeJSON(w, http.StatusCreated, toAPITrip(value))
}

func (h *Handler) GetTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	value, err := h.trips.Get(r.Context(), tripID)
	if errors.Is(err, trip.ErrNotFound) {
		writeProblem(w, r, tripNotFoundProblem)
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPITrip(value))
}

func (h *Handler) FinishTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	value, err := h.trips.Finish(r.Context(), tripID)
	switch {
	case errors.Is(err, trip.ErrNotFound):
		writeProblem(w, r, tripNotFoundProblem)
		return
	case errors.Is(err, trip.ErrCompleted):
		writeProblem(w, r, tripCompletedProblem)
		return
	case err != nil:
		writeInternalError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPITrip(value))
}

func writeInternalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(
		r.Context(),
		"request failed",
		"method", r.Method,
		"path", r.URL.Path,
		"error", err,
	)
	writeProblem(w, r, internalErrorProblem)
}

func toDomainCoordinates(value api.Coordinates) trip.Coordinates {
	return trip.Coordinates{
		Latitude:  value.Latitude,
		Longitude: value.Longitude,
	}
}

func toAPITrip(value trip.Trip) api.Trip {
	return api.Trip{
		Id:       value.ID,
		UserId:   value.UserID,
		DriverId: value.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  value.StartPoint.Latitude,
			Longitude: value.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  value.EndPoint.Latitude,
			Longitude: value.EndPoint.Longitude,
		},
		Price:          value.Price,
		Status:         api.TripStatus(value.Status),
		StartedAt:      value.StartedAt,
		FinishedAt:     value.FinishedAt,
		LastPositionAt: value.LastPositionAt,
	}
}
