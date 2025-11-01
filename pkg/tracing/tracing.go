package tracing

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// InitTracer initializes the OpenTelemetry tracer and returns a TracerProvider
func InitTracer(serviceName string) (*sdktrace.TracerProvider, error) {
	logger := slog.Default()

	// Get Tempo endpoint from environment or use default
	tempoEndpoint := os.Getenv("TEMPO_ENDPOINT")
	if tempoEndpoint == "" {
		tempoEndpoint = "tempo:4317"
	}

	logger.Info("initializing tracer",
		"service", serviceName,
		"tempo_endpoint", tempoEndpoint,
	)

	// Create OTLP trace exporter
	ctx := context.Background()
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(tempoEndpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		logger.Error("failed to create trace exporter", "error", err)
		return nil, err
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("environment", "development"),
		),
	)
	if err != nil {
		logger.Error("failed to create resource", "error", err)
		return nil, err
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Sample all traces in dev
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator to propagate trace context across services
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("tracer initialized successfully", "service", serviceName)

	return tp, nil
}

// GetTracer returns a tracer for the given instrumentation name
func GetTracer(instrumentationName string) trace.Tracer {
	return otel.Tracer(instrumentationName)
}

// SpanContext returns the span context from the given context
func SpanContext(ctx context.Context) trace.SpanContext {
	return trace.SpanContextFromContext(ctx)
}

// TraceIDFromContext returns the trace ID as a string from the context
func TraceIDFromContext(ctx context.Context) string {
	spanCtx := SpanContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// SpanIDFromContext returns the span ID as a string from the context
func SpanIDFromContext(ctx context.Context) string {
	spanCtx := SpanContext(ctx)
	if spanCtx.HasSpanID() {
		return spanCtx.SpanID().String()
	}
	return ""
}

// StartSpan starts a new span with the given name and options
// Returns the new context and span. Caller must call span.End() when done.
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer("docutag")
	return tracer.Start(ctx, spanName, opts...)
}

// RecordError records an error on the span in the context
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanAttributes sets attributes on the span in the context
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// AddEvent adds an event to the span in the context
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}
