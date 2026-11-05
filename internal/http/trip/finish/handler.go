package finish

import (
	"net/http"

	"github.com/provsenya/avito/api"
	"github.com/provsenya/avito/internal/http/domainerror"
	"github.com/provsenya/avito/internal/http/response"
)

type Handler struct{ usecase usecase }

func New(u usecase) *Handler { return &Handler{usecase: u} }
func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	value, err := h.usecase.Execute(r.Context(), id)
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, response.Trip(value))
}
