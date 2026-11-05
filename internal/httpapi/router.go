package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/provsenya/avito/api"
)

func NewRouter(handler api.ServerInterface) http.Handler {
	router := chi.NewRouter()

	return api.HandlerWithOptions(handler, api.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: invalidParameter,
	})
}

func invalidParameter(w http.ResponseWriter, r *http.Request, _ error) {
	writeProblem(w, r, invalidRequestProblem)
}
