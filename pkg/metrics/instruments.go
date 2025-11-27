package metrics

import (
	"go.opentelemetry.io/otel/metric"
)

// HTTP Client Metrics (OTEL Semantic Conventions)
var (
	// HTTPClientRequestDuration measures the duration of HTTP client requests
	HTTPClientRequestDuration metric.Float64Histogram

	// HTTPClientRequestBodySize measures the size of HTTP request bodies
	HTTPClientRequestBodySize metric.Int64Histogram

	// HTTPClientResponseBodySize measures the size of HTTP response bodies
	HTTPClientResponseBodySize metric.Int64Histogram
)

// Pipeline Metrics
var (
	// PipelineCyclesTotal counts pipeline processing cycles
	PipelineCyclesTotal metric.Int64Counter

	// PipelineCycleDuration measures the duration of pipeline cycles
	PipelineCycleDuration metric.Float64Histogram

	// PipelineLinesProcessed counts processed bus lines
	PipelineLinesProcessed metric.Int64Counter

	// PipelineVehiclesProcessed counts processed vehicles
	PipelineVehiclesProcessed metric.Int64Counter

	// PipelineLinesInFlight tracks concurrent line processing
	PipelineLinesInFlight metric.Int64UpDownCounter
)

// Parser Metrics
var (
	// XMLParseDuration measures XML parsing duration
	XMLParseDuration metric.Float64Histogram
)

// initializeInstruments creates all metric instruments
func initializeInstruments() error {
	var err error

	// HTTP Client Metrics - following OTEL semantic conventions
	HTTPClientRequestDuration, err = Meter.Float64Histogram(
		"http.client.request.duration",
		metric.WithDescription("Duration of HTTP client requests"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1.0, 2.5, 5.0, 7.5, 10.0),
	)
	if err != nil {
		return err
	}

	HTTPClientRequestBodySize, err = Meter.Int64Histogram(
		"http.client.request.body.size",
		metric.WithDescription("Size of HTTP request bodies"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(1024, 10240, 102400, 1048576, 10485760), // 1KB to 10MB
	)
	if err != nil {
		return err
	}

	HTTPClientResponseBodySize, err = Meter.Int64Histogram(
		"http.client.response.body.size",
		metric.WithDescription("Size of HTTP response bodies"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(1024, 10240, 102400, 1048576, 10485760), // 1KB to 10MB
	)
	if err != nil {
		return err
	}

	// Pipeline Metrics
	PipelineCyclesTotal, err = Meter.Int64Counter(
		"pipeline.cycles.total",
		metric.WithDescription("Total number of pipeline processing cycles"),
		metric.WithUnit("{cycle}"),
	)
	if err != nil {
		return err
	}

	PipelineCycleDuration, err = Meter.Float64Histogram(
		"pipeline.cycle.duration",
		metric.WithDescription("Duration of pipeline processing cycles"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0),
	)
	if err != nil {
		return err
	}

	PipelineLinesProcessed, err = Meter.Int64Counter(
		"pipeline.lines.processed",
		metric.WithDescription("Number of bus lines processed"),
		metric.WithUnit("{line}"),
	)
	if err != nil {
		return err
	}

	PipelineVehiclesProcessed, err = Meter.Int64Counter(
		"pipeline.vehicles.processed",
		metric.WithDescription("Number of vehicles processed"),
		metric.WithUnit("{vehicle}"),
	)
	if err != nil {
		return err
	}

	PipelineLinesInFlight, err = Meter.Int64UpDownCounter(
		"pipeline.lines.in_flight",
		metric.WithDescription("Number of lines currently being processed"),
		metric.WithUnit("{line}"),
	)
	if err != nil {
		return err
	}

	// Parser Metrics
	XMLParseDuration, err = Meter.Float64Histogram(
		"xml.parse.duration",
		metric.WithDescription("Duration of XML parsing operations"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5),
	)
	if err != nil {
		return err
	}

	return nil
}
