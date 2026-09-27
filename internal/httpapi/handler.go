package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/provsenya/avito/api"
	"github.com/provsenya/avito/internal/trip"
)

type Handler struct {
	pool         *pgxpool.Pool
	trips        *trip.Service
	readyTimeout time.Duration
}

func NewHandler(
	pool *pgxpool.Pool,
	trips *trip.Service,
	readyTimeout time.Duration,
) *Handler {
	return &Handler{
		pool:         pool,
		trips:        trips,
		readyTimeout: readyTimeout,
	}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeHealth(w, http.StatusOK, api.Ok)
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.readyTimeout)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		writeHealth(w, http.StatusServiceUnavailable, api.Unavailable)
		return
	}
	writeHealth(w, http.StatusOK, api.Ok)
}

func writeHealth(w http.ResponseWriter, status int, value api.HealthResponseStatus) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: value})
}
