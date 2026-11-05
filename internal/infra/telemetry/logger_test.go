package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestLogCorrelation(t *testing.T) {
	provider := sdktrace.NewTracerProvider()
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Error(err)
		}
	})
	ctx, span := provider.Tracer("test").Start(context.Background(), "request")
	defer span.End()
	var output bytes.Buffer
	logger := slog.New(&traceHandler{Handler: slog.NewJSONHandler(&output, nil)}).With("service", "trip-service")
	logger.InfoContext(ctx, "trip created")
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["trace_id"] != span.SpanContext().TraceID().String() || entry["span_id"] != span.SpanContext().SpanID().String() || entry["service"] != "trip-service" {
		t.Fatalf("entry=%v", entry)
	}
}
