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

	// PipelineErrorsTotal counts errors by stage and type
	PipelineErrorsTotal metric.Int64Counter

	// PipelineStageDuration measures duration per processing stage
	PipelineStageDuration metric.Float64Histogram

	// PipelineCycleLinesTotal counts lines attempted per cycle
	PipelineCycleLinesTotal metric.Int64Counter

	// PipelineCycleLinesFailed counts lines that failed per cycle
	PipelineCycleLinesFailed metric.Int64Counter
)

// Parser Metrics
var (
	// XMLParseDuration measures XML parsing duration
	XMLParseDuration metric.Float64Histogram

	// ParserVehiclesExtracted counts successfully extracted vehicles
	ParserVehiclesExtracted metric.Int64Counter

	// ParserVehiclesFailed counts vehicles that failed to parse
	ParserVehiclesFailed metric.Int64Counter

	// ParserPayloadSize measures the size of XML payloads being parsed
	ParserPayloadSize metric.Int64Histogram
)

// Loki Metrics
var (
	// LokiBatchSize measures the number of vehicle records per batch
	LokiBatchSize metric.Int64Histogram

	// LokiBatchStreams measures the number of streams per push request
	LokiBatchStreams metric.Int64Histogram

	// LokiSendDuration measures the duration of Loki push operations
	LokiSendDuration metric.Float64Histogram

	// LokiSendTotal counts total Loki sends by status
	LokiSendTotal metric.Int64Counter

	// LokiSendRetries counts retry attempts
	LokiSendRetries metric.Int64Counter
)

// BODS API Metrics
var (
	// BODSAPIRequestsTotal counts total BODS API requests
	BODSAPIRequestsTotal metric.Int64Counter
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

	PipelineErrorsTotal, err = Meter.Int64Counter(
		"pipeline.errors.total",
		metric.WithDescription("Total errors by stage and type"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return err
	}

	PipelineStageDuration, err = Meter.Float64Histogram(
		"pipeline.stage.duration",
		metric.WithDescription("Duration per processing stage"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0),
	)
	if err != nil {
		return err
	}

	PipelineCycleLinesTotal, err = Meter.Int64Counter(
		"pipeline.cycle.lines.total",
		metric.WithDescription("Lines attempted per cycle"),
		metric.WithUnit("{line}"),
	)
	if err != nil {
		return err
	}

	PipelineCycleLinesFailed, err = Meter.Int64Counter(
		"pipeline.cycle.lines.failed",
		metric.WithDescription("Lines that failed per cycle"),
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

	ParserVehiclesExtracted, err = Meter.Int64Counter(
		"parser.vehicles.extracted",
		metric.WithDescription("Vehicles successfully extracted"),
		metric.WithUnit("{vehicle}"),
	)
	if err != nil {
		return err
	}

	ParserVehiclesFailed, err = Meter.Int64Counter(
		"parser.vehicles.failed",
		metric.WithDescription("Vehicle records that failed to parse"),
		metric.WithUnit("{vehicle}"),
	)
	if err != nil {
		return err
	}

	ParserPayloadSize, err = Meter.Int64Histogram(
		"parser.payload.size",
		metric.WithDescription("Size of XML payloads being parsed"),
		metric.WithUnit("By"),
		metric.WithExplicitBucketBoundaries(1024, 10240, 102400, 1048576, 10485760), // 1KB to 10MB
	)
	if err != nil {
		return err
	}

	// Loki Metrics
	LokiBatchSize, err = Meter.Int64Histogram(
		"loki.batch.size",
		metric.WithDescription("Number of vehicle records per batch"),
		metric.WithUnit("{record}"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000),
	)
	if err != nil {
		return err
	}

	LokiBatchStreams, err = Meter.Int64Histogram(
		"loki.batch.streams",
		metric.WithDescription("Number of streams per push request"),
		metric.WithUnit("{stream}"),
		metric.WithExplicitBucketBoundaries(1, 2, 5, 10, 25, 50, 100),
	)
	if err != nil {
		return err
	}

	LokiSendDuration, err = Meter.Float64Histogram(
		"loki.send.duration",
		metric.WithDescription("Duration of Loki push operations"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0),
	)
	if err != nil {
		return err
	}

	LokiSendTotal, err = Meter.Int64Counter(
		"loki.send.total",
		metric.WithDescription("Total Loki sends by status"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return err
	}

	LokiSendRetries, err = Meter.Int64Counter(
		"loki.send.retries",
		metric.WithDescription("Retry attempts for Loki sends"),
		metric.WithUnit("{retry}"),
	)
	if err != nil {
		return err
	}

	// BODS API Metrics
	BODSAPIRequestsTotal, err = Meter.Int64Counter(
		"bods.api.requests.total",
		metric.WithDescription("Total BODS API requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return err
	}

	return nil
}
