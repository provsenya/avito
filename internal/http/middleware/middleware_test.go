package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/provsenya/avito/internal/http/middleware"
)

func TestMiddleware(t *testing.T) {
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Error(err)
		}
	})
	var logs bytes.Buffer
	m, err := middleware.New(slog.New(slog.NewJSONHandler(&logs, nil)), provider.Meter("test"))
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	router.Get("/trips/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	router.Get("/panic", func(http.ResponseWriter, *http.Request) { panic("secret") })
	handler := m.Wrap(router)
	for _, tc := range []struct {
		path   string
		status int
	}{{"/trips/a", 200}, {"/panic", 500}} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		if w.Code != tc.status || strings.Contains(w.Body.String(), "secret") {
			t.Fatalf("response=%d %s", w.Code, w.Body.String())
		}
	}
	decoder := json.NewDecoder(&logs)
	for _, want := range []struct {
		status int
		level  string
	}{{200, "INFO"}, {500, "ERROR"}} {
		var record map[string]any
		if err := decoder.Decode(&record); err != nil {
			t.Fatal(err)
		}
		if record["status"] != float64(want.status) || record["level"] != want.level {
			t.Fatalf("log=%v", record)
		}
		if want.status == 500 && record["stack"] == nil {
			t.Fatal("panic stack is missing")
		}
	}
	if decoder.More() {
		t.Fatal("duplicate request log")
	}
	var data metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &data); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != "http.server.request.duration" {
				continue
			}
			hist, ok := m.Data.(metricdata.Histogram[float64])
			if !ok {
				t.Fatal("wrong histogram type")
			}
			for _, point := range hist.DataPoints {
				route, _ := point.Attributes.Value("http.route")
				if route.AsString() == "/trips/{id}" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("route template metric missing")
	}
}
