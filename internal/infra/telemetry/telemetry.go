package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/provsenya/avito/internal/config"
)

type Telemetry struct {
	Traces             *sdktrace.TracerProvider
	Metrics            *sdkmetric.MeterProvider
	Logger             *slog.Logger
	created, completed metric.Int64Counter
}

func New(ctx context.Context, cfg config.Config) (*Telemetry, error) {
	logger := slog.New(&traceHandler{Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})}).With("service", "trip-service")
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) { logger.Warn("telemetry export failed", "error", err) }))
	traceExporter, metricExporter, err := exporters(ctx, cfg.Telemetry.Protocol)
	if err != nil {
		return nil, err
	}
	res, err := resource.New(ctx, resource.WithFromEnv(), resource.WithAttributes(attribute.String("service.name", "trip-service"), attribute.String("service.version", "lab2")))
	if err != nil {
		return nil, errors.Join(err, traceExporter.Shutdown(ctx), metricExporter.Shutdown(ctx))
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithBatcher(traceExporter))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(5*time.Second))))
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	meter := mp.Meter("trip-service")
	created, err := meter.Int64Counter("tripgo.trips.created")
	if err != nil {
		return nil, err
	}
	completed, err := meter.Int64Counter("tripgo.trips.completed")
	if err != nil {
		return nil, err
	}
	return &Telemetry{Traces: tp, Metrics: mp, Logger: logger, created: created, completed: completed}, nil
}

func exporters(ctx context.Context, protocol string) (sdktrace.SpanExporter, sdkmetric.Exporter, error) {
	var te sdktrace.SpanExporter
	var me sdkmetric.Exporter
	var err error
	if protocol == "grpc" {
		te, err = otlptracegrpc.New(ctx)
	} else {
		te, err = otlptracehttp.New(ctx)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("create trace exporter: %w", err)
	}
	if protocol == "grpc" {
		me, err = otlpmetricgrpc.New(ctx)
	} else {
		me, err = otlpmetrichttp.New(ctx)
	}
	if err != nil {
		return nil, nil, errors.Join(fmt.Errorf("create metric exporter: %w", err), te.Shutdown(ctx))
	}
	return te, me, nil
}

func (t *Telemetry) Record(ctx context.Context, event string) {
	switch event {
	case "created":
		t.created.Add(ctx, 1)
	case "completed":
		t.completed.Add(ctx, 1)
	}
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(t.Traces.Shutdown(ctx), t.Metrics.Shutdown(ctx))
}

type traceHandler struct{ slog.Handler }

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	span := trace.SpanContextFromContext(ctx)
	traceID, spanID := "", ""
	if span.IsValid() {
		traceID = span.TraceID().String()
		spanID = span.SpanID().String()
	}
	r.AddAttrs(slog.String("trace_id", traceID), slog.String("span_id", spanID))
	return h.Handler.Handle(ctx, r)
}

func (h *traceHandler) WithAttrs(a []slog.Attr) slog.Handler {
	return &traceHandler{h.Handler.WithAttrs(a)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{h.Handler.WithGroup(name)}
}
