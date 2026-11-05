package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/provsenya/avito/internal/http/domainerror"
	"github.com/provsenya/avito/internal/http/requeststate"
)

type Middleware struct {
	logger   *slog.Logger
	duration metric.Float64Histogram
	active   metric.Int64UpDownCounter
}

func New(logger *slog.Logger, meter metric.Meter) (*Middleware, error) {
	duration, err := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	active, err := meter.Int64UpDownCounter("http.server.active_requests")
	if err != nil {
		return nil, err
	}
	return &Middleware{logger: logger, duration: duration, active: active}, nil
}

func (m *Middleware) Wrap(router *chi.Mux) http.Handler {
	handler := m.logging(m.metrics(router))
	traced := otelhttp.NewHandler(handler, "", otelhttp.WithMeterProvider(noop.NewMeterProvider()), otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string { return r.Method + " " + requeststate.From(r.Context()).Route }))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeContext := chi.NewRouteContext()
		route := "unmatched"
		if router.Match(routeContext, r.Method, r.URL.Path) {
			route = routeContext.RoutePattern()
		}
		state := &requeststate.State{Status: http.StatusOK, Route: route, Context: r.Context()}
		r = r.WithContext(requeststate.With(r.Context(), state))
		defer func() {
			if value := recover(); value != nil {
				err := fmt.Errorf("panic: %v", value)
				if state.Err == nil {
					m.logger.ErrorContext(state.Context, "panic recovered", "error", err)
				}
				domainerror.Write(w, r.WithContext(state.Context), err)
			}
		}()
		traced.ServeHTTP(w, r)
	})
}

func (m *Middleware) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := requeststate.From(r.Context())
		state.Context = r.Context()
		started := time.Now()
		defer func() {
			level := slog.LevelInfo
			if state.Status >= 500 {
				level = slog.LevelError
			}
			args := []any{"method", r.Method, "route", state.Route, "status", state.Status, "duration_ms", time.Since(started).Milliseconds()}
			if state.Err != nil && state.Status >= 500 {
				args = append(args, "error", state.Err)
				if state.Stack != "" {
					args = append(args, "stack", state.Stack)
				}
			}
			m.logger.Log(r.Context(), level, "request handled", args...)
		}()
		next.ServeHTTP(w, r)
	})
}
