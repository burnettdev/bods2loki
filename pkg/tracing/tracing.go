package tracing

import (
	"context"
	"log/slog"

	"bods2loki/pkg/otel"

	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
)

// InitTracing initializes OpenTelemetry tracing with the configured exporter.
// Returns a shutdown function that should be called on application exit.
func InitTracing() (func(), error) {
	// Check if tracing is enabled
	if !otel.IsTracingEnabled() {
		slog.Debug("OpenTelemetry tracing is disabled")
		return func() {}, nil
	}

	ctx := context.Background()

	// Get exporter configuration for traces
	cfg := otel.GetExporterConfig(otel.SignalTraces)

	// Create exporter based on protocol
	exporter, err := otel.NewTraceExporter(ctx, cfg)
	if err != nil {
		slog.Warn("Failed to create OTLP trace exporter, using noop", "error", err)
		return func() {}, nil
	}

	// Create shared resource
	res, err := otel.NewResource()
	if err != nil {
		slog.Warn("Failed to create resource, using noop", "error", err)
		return func() {}, nil
	}

	// Create trace provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	// Set global trace provider
	otelapi.SetTracerProvider(tp)
	otelapi.SetTextMapPropagator(propagation.TraceContext{})

	slog.Debug("OpenTelemetry tracing initialized",
		"endpoint", cfg.Endpoint,
		"protocol", cfg.Protocol,
	)

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			slog.Error("Error shutting down tracer provider", "error", err)
		}
	}, nil
}
