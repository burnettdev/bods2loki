package tracing

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"bods2loki/pkg/otel"

	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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

	// Create trace provider with sampler
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(getSampler()),
	)

	// Set global trace provider
	otelapi.SetTracerProvider(tp)

	// Set propagator with TraceContext and Baggage support
	otelapi.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

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

// getSampler returns a trace sampler based on OTEL_TRACES_SAMPLER and OTEL_TRACES_SAMPLER_ARG
// environment variables per the OpenTelemetry specification.
func getSampler() sdktrace.Sampler {
	samplerName := os.Getenv("OTEL_TRACES_SAMPLER")
	if samplerName == "" {
		samplerName = "parentbased_always_on" // OTEL default
	}

	arg := os.Getenv("OTEL_TRACES_SAMPLER_ARG")
	ratio := 1.0
	if arg != "" {
		if r, err := strconv.ParseFloat(arg, 64); err == nil {
			ratio = r
		}
	}

	switch strings.ToLower(samplerName) {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(ratio)
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case "parentbased_traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	default:
		slog.Warn("Unknown sampler, using parentbased_always_on", "sampler", samplerName)
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}
