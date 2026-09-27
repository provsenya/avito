package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/provsenya/avito/api"
)

type problemDefinition struct {
	status int
	code   string
	typeID string
	title  string
	detail string
}

var (
	invalidRequestProblem = problemDefinition{
		status: http.StatusBadRequest,
		code:   "invalid_request",
		typeID: "https://tripgo.example/problems/invalid-request",
		title:  "Invalid request",
		detail: "Request validation failed",
	}
	driverBusyProblem = problemDefinition{
		status: http.StatusConflict,
		code:   "driver_busy",
		typeID: "https://tripgo.example/problems/driver-busy",
		title:  "Driver busy",
		detail: "Driver already has an active trip",
	}
	tripNotFoundProblem = problemDefinition{
		status: http.StatusNotFound,
		code:   "trip_not_found",
		typeID: "https://tripgo.example/problems/trip-not-found",
		title:  "Trip not found",
		detail: "Trip was not found",
	}
	tripCompletedProblem = problemDefinition{
		status: http.StatusConflict,
		code:   "trip_completed",
		typeID: "https://tripgo.example/problems/trip-completed",
		title:  "Trip completed",
		detail: "Operation is not allowed for a completed trip",
	}
	internalErrorProblem = problemDefinition{
		status: http.StatusInternalServerError,
		code:   "internal_error",
		typeID: "https://tripgo.example/problems/internal-error",
		title:  "Internal Server Error",
		detail: "Internal server error",
	}
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeProblem(w http.ResponseWriter, r *http.Request, definition problemDefinition) {
	instance := r.URL.Path
	detail := definition.detail

	value := api.Problem{
		Type:     definition.typeID,
		Title:    definition.title,
		Status:   int32(definition.status),
		Detail:   &detail,
		Instance: &instance,
		Code:     definition.code,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(definition.status)
	_ = json.NewEncoder(w).Encode(value)
}
