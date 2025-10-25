package tracing

import (
	"log/slog"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware creates a middleware that instruments HTTP requests with tracing
func HTTPMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Use otelhttp to automatically create spans for HTTP requests
		return otelhttp.NewHandler(next, serviceName,
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				return r.Method + " " + r.URL.Path
			}),
		)
	}
}

// WrapHandler wraps an http.Handler with tracing
func WrapHandler(handler http.Handler, operationName string) http.Handler {
	return otelhttp.NewHandler(handler, operationName)
}

// WrapHandlerFunc wraps an http.HandlerFunc with tracing
func WrapHandlerFunc(handler http.HandlerFunc, operationName string) http.Handler {
	return otelhttp.NewHandler(handler, operationName)
}

// AddSpanAttributes adds attributes to the current span in the context
func AddSpanAttributes(r *http.Request, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(r.Context())
	span.SetAttributes(attrs...)
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(r *http.Request, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(r.Context())
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// LogWithTrace logs with trace context from the HTTP request
func LogWithTrace(r *http.Request, logger *slog.Logger, level slog.Level, msg string, args ...any) {
	traceID := TraceIDFromContext(r.Context())
	spanID := SpanIDFromContext(r.Context())

	logArgs := []any{"trace_id", traceID, "span_id", spanID}
	logArgs = append(logArgs, args...)

	logger.Log(r.Context(), level, msg, logArgs...)
}
