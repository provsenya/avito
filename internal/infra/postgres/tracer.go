package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

type (
	queryKey  struct{}
	queryInfo struct {
		at               time.Time
		operation, table string
	}
)

type queryTracer struct {
	*otelpgx.Tracer
	duration metric.Float64Histogram
}

func newQueryTracer() (*queryTracer, error) {
	duration, err := otel.Meter("trip-service").Float64Histogram("db.client.operation.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	return &queryTracer{Tracer: otelpgx.NewTracer(otelpgx.WithDisableAcquireTracer(), otelpgx.WithMeterProvider(noop.NewMeterProvider()), otelpgx.WithDisableSQLStatementInAttributes()), duration: duration}, nil
}

func (t *queryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)
	fields := strings.Fields(data.SQL)
	info := queryInfo{at: time.Now()}
	if len(fields) > 0 {
		info.operation = strings.ToUpper(fields[0])
	}
	for i, word := range fields {
		if (strings.EqualFold(word, "FROM") || strings.EqualFold(word, "INTO") || strings.EqualFold(word, "UPDATE")) && i+1 < len(fields) {
			info.table = fields[i+1]
			break
		}
	}
	trace.SpanFromContext(ctx).SetAttributes(attribute.String("db.operation.name", info.operation), attribute.String("db.collection.name", info.table))
	return context.WithValue(ctx, queryKey{}, info)
}

func (t *queryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	info, _ := ctx.Value(queryKey{}).(queryInfo)
	errorType := ""
	if data.Err != nil {
		errorType = "database_error"
		span := trace.SpanFromContext(ctx)
		span.SetStatus(codes.Error, "database operation failed")
		span.SetAttributes(attribute.String("error.type", errorType))
	}
	t.duration.Record(ctx, time.Since(info.at).Seconds(), metric.WithAttributes(attribute.String("db.operation.name", info.operation), attribute.String("db.collection.name", info.table), attribute.String("error.type", errorType)))
	t.Tracer.TraceQueryEnd(ctx, conn, data)
}
