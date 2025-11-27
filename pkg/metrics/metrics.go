package metrics

import (
	"context"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"

	"bods2loki/pkg/otel"

	otelapi "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	// meterProvider is the global meter provider
	meterProvider *sdkmetric.MeterProvider

	// Meter is the global meter for creating instruments
	Meter metric.Meter

	// lastSuccessTimestamp tracks the last successful pipeline cycle (Unix timestamp)
	lastSuccessTimestamp atomic.Int64
)

// InitMetrics initializes OpenTelemetry metrics with the configured exporter.
// Returns a shutdown function that should be called on application exit.
func InitMetrics() (func(), error) {
	// Check if metrics is enabled
	if !otel.IsMetricsEnabled() {
		slog.Debug("OpenTelemetry metrics is disabled")
		return func() {}, nil
	}

	ctx := context.Background()

	// Get exporter configuration for metrics
	cfg := otel.GetExporterConfig(otel.SignalMetrics)

	// Create exporter based on protocol
	exporter, err := otel.NewMetricExporter(ctx, cfg)
	if err != nil {
		slog.Warn("Failed to create OTLP metric exporter, using noop", "error", err)
		return func() {}, nil
	}

	// Create shared resource
	res, err := otel.NewResource()
	if err != nil {
		slog.Warn("Failed to create resource, using noop", "error", err)
		return func() {}, nil
	}

	// Create meter provider with periodic reader (60s export interval)
	meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(60*time.Second),
			),
		),
		sdkmetric.WithResource(res),
	)

	// Set global meter provider
	otelapi.SetMeterProvider(meterProvider)

	// Create meter for this application
	Meter = meterProvider.Meter("bods2loki")

	// Initialize all instruments
	if err := initializeInstruments(); err != nil {
		slog.Error("Failed to initialize metric instruments", "error", err)
		return func() {}, nil
	}

	// Register runtime metrics
	if err := registerRuntimeMetrics(); err != nil {
		slog.Warn("Failed to register runtime metrics", "error", err)
	}

	slog.Debug("OpenTelemetry metrics initialized",
		"endpoint", cfg.Endpoint,
		"protocol", cfg.Protocol,
	)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := meterProvider.Shutdown(ctx); err != nil {
			slog.Error("Error shutting down meter provider", "error", err)
		}
	}, nil
}

// registerRuntimeMetrics registers observable gauges for runtime metrics
func registerRuntimeMetrics() error {
	// Goroutine count
	_, err := Meter.Int64ObservableGauge(
		"runtime.go.goroutines",
		metric.WithDescription("Number of goroutines"),
		metric.WithUnit("{goroutine}"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(int64(runtime.NumGoroutine()))
			return nil
		}),
	)
	if err != nil {
		return err
	}

	// Last success timestamp
	_, err = Meter.Int64ObservableGauge(
		"pipeline.last_success.timestamp",
		metric.WithDescription("Unix timestamp of the last successful pipeline cycle"),
		metric.WithUnit("s"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			ts := lastSuccessTimestamp.Load()
			if ts > 0 {
				o.Observe(ts)
			}
			return nil
		}),
	)
	if err != nil {
		return err
	}

	return nil
}

// RecordLastSuccessTimestamp records the current time as the last successful cycle
func RecordLastSuccessTimestamp() {
	lastSuccessTimestamp.Store(time.Now().Unix())
}

// IsEnabled returns true if metrics collection is enabled
func IsEnabled() bool {
	return Meter != nil
}
