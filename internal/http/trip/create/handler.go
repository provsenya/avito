package create

import (
	"fmt"
	"net/http"

	"github.com/provsenya/avito/api"
	"github.com/provsenya/avito/internal/domain/trip"
	"github.com/provsenya/avito/internal/http/domainerror"
	"github.com/provsenya/avito/internal/http/response"
	"github.com/provsenya/avito/internal/model"
)

type Handler struct{ usecase usecase }

func New(u usecase) *Handler { return &Handler{usecase: u} }

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	request, err := decodeCreateTripRequest(w, r)
	if err != nil {
		domainerror.Write(w, r, &trip.ValidationError{Field: "body", Message: err.Error()})
		return
	}
	value, err := h.usecase.Execute(r.Context(), model.CreateTrip{UserID: request.UserId, DriverID: request.DriverId, StartPoint: response.Coordinates(request.StartPoint), EndPoint: response.Coordinates(request.EndPoint), Price: request.Price})
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/v1/trips/%s", value.ID))
	response.JSON(w, http.StatusCreated, response.Trip(value))
}
