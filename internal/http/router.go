package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/provsenya/avito/api"
	"github.com/provsenya/avito/internal/domain/trip"
	"github.com/provsenya/avito/internal/http/domainerror"
	"github.com/provsenya/avito/internal/http/response"
	"github.com/provsenya/avito/internal/http/trip/create"
	"github.com/provsenya/avito/internal/http/trip/finish"
	"github.com/provsenya/avito/internal/http/trip/get"
)

type Handler struct {
	*create.Handler
	Get     *get.Handler
	Finish  *finish.Handler
	pool    readiness
	timeout time.Duration
}

func New(c *create.Handler, g *get.Handler, f *finish.Handler, pool readiness, timeout time.Duration) *chi.Mux {
	h := &Handler{Handler: c, Get: g, Finish: f, pool: pool, timeout: timeout}
	router := chi.NewRouter()
	api.HandlerWithOptions(h, api.ChiServerOptions{BaseRouter: router, ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
		domainerror.Write(w, r, &trip.ValidationError{Field: "parameter", Message: "invalid UUID"})
	}})
	return router
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	h.Get.GetTrip(w, r, id)
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	h.Finish.FinishTrip(w, r, id)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()
	if err := h.pool.Ping(ctx); err != nil {
		response.JSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	response.JSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}
