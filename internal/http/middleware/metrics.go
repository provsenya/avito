package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/provsenya/avito/internal/http/requeststate"
)

func (m *Middleware) metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := requeststate.From(r.Context())
		started := time.Now()
		attrs := []attribute.KeyValue{attribute.String("http.request.method", r.Method), attribute.String("http.route", state.Route)}
		trace.SpanFromContext(r.Context()).SetAttributes(attribute.String("http.route", state.Route))
		m.active.Add(r.Context(), 1, metric.WithAttributes(attrs...))
		writer := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		defer func() {
			value := recover()
			state.Status = writer.Status()
			if state.Status == 0 {
				state.Status = http.StatusOK
			}
			if value != nil {
				state.Status = http.StatusInternalServerError
				state.Err = fmt.Errorf("panic: %v", value)
				state.Stack = string(debug.Stack())
				span := trace.SpanFromContext(r.Context())
				span.RecordError(state.Err)
				span.SetStatus(codes.Error, "panic")
				span.SetAttributes(attribute.String("error.type", "panic"))
			}
			trace.SpanFromContext(r.Context()).SetAttributes(attribute.Int("http.response.status_code", state.Status))
			m.active.Add(r.Context(), -1, metric.WithAttributes(attrs...))
			attrs = append(attrs, attribute.Int("http.response.status_code", state.Status))
			m.duration.Record(r.Context(), time.Since(started).Seconds(), metric.WithAttributes(attrs...))
			if value != nil {
				panic(value)
			}
		}()
		next.ServeHTTP(writer, r)
	})
}
