package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"bods2loki/pkg/metrics"
	pkgotel "bods2loki/pkg/otel"
	"bods2loki/pkg/types"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	serverHost string
	username   string
	password   string
	tracer     trace.Tracer
}

type PushRequest struct {
	Streams []Stream `json:"streams"`
}

type Stream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

func NewClient(baseURL, username, password string) *Client {
	// Create HTTP client with OpenTelemetry instrumentation
	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   30 * time.Second,
	}

	// Extract server host for metrics
	parsedURL, _ := url.Parse(baseURL)
	serverHost := ""
	if parsedURL != nil {
		serverHost = parsedURL.Host
	}

	return &Client{
		httpClient: client,
		baseURL:    baseURL,
		serverHost: serverHost,
		username:   username,
		password:   password,
		tracer:     otel.Tracer("loki-client"),
	}
}

// vehicleLogEntry wraps vehicle data with metadata for Loki log entries
type vehicleLogEntry struct {
	Timestamp string `json:"timestamp"`
	LineRef   string `json:"line_ref"`
	types.VehicleActivity
}

func (c *Client) SendBusData(ctx context.Context, data *types.ParsedBusData) error {
	ctx, span := c.tracer.Start(ctx, "loki.send_bus_data",
		trace.WithAttributes(
			attribute.String("line_ref", data.LineRef),
			attribute.Int("vehicles_count", len(data.VehicleData)),
			attribute.String("server.address", c.serverHost),
		),
	)
	defer span.End()

	start := time.Now()

	// Create individual log entries for each vehicle
	var logValues [][]string

	for _, vehicle := range data.VehicleData {
		// Create log entry with embedded vehicle data
		// All VehicleActivity fields are automatically included
		// Missing fields are silently omitted via omitempty tags
		entry := vehicleLogEntry{
			Timestamp:       data.Timestamp,
			LineRef:         data.LineRef,
			VehicleActivity: vehicle,
		}

		// Convert to JSON
		vehicleJSON, err := json.Marshal(entry)
		if err != nil {
			pkgotel.RecordError(span, err, pkgotel.ErrorTypeParse, false)
			return fmt.Errorf("failed to marshal vehicle JSON: %w", err)
		}

		// Add to log values with current timestamp
		logValues = append(logValues, []string{
			strconv.FormatInt(time.Now().UnixNano(), 10),
			string(vehicleJSON),
		})
	}

	// Add event for batch preparation
	span.AddEvent("batch.prepared", trace.WithAttributes(
		attribute.Int("log_lines.count", len(logValues)),
	))

	// Record batch metrics
	c.recordBatchMetrics(ctx, len(logValues), 1) // 1 stream per request currently

	// Create Loki push request with individual log lines
	lokiReq := PushRequest{
		Streams: []Stream{
			{
				Stream: map[string]string{
					"job":      "bods2loki",
					"service":  "bus-tracking",
					"line_ref": data.LineRef,
				},
				Values: logValues,
			},
		},
	}

	// Marshal Loki request
	reqBody, err := json.Marshal(lokiReq)
	if err != nil {
		pkgotel.RecordError(span, err, pkgotel.ErrorTypeParse, false)
		return fmt.Errorf("failed to marshal Loki request: %w", err)
	}

	requestSize := int64(len(reqBody))

	// Send to Loki
	reqURL := fmt.Sprintf("%s/loki/api/v1/push", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(reqBody))
	if err != nil {
		pkgotel.RecordError(span, err, pkgotel.ErrorTypeNetwork, false)
		c.recordHTTPMetrics(ctx, start, 0, requestSize, "request_creation_error")
		c.recordSendMetrics(ctx, start, "error")
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "bods2loki/1.0.0")

	// Add basic authentication if credentials are provided
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
		span.AddEvent("auth.applied", trace.WithAttributes(
			attribute.String("auth.type", "basic"),
		))
		span.SetAttributes(
			attribute.Bool("auth.enabled", true),
			attribute.String("auth.username", c.username),
		)
	} else {
		span.SetAttributes(
			attribute.Bool("auth.enabled", false),
		)
	}

	// Use OTEL semantic conventions for HTTP attributes
	span.SetAttributes(
		attribute.String("url.full", reqURL),
		attribute.String("http.request.method", "POST"),
		attribute.Int64("http.request.body.size", requestSize),
		attribute.Int("log_lines_count", len(logValues)),
	)

	// Add span event for request start
	span.AddEvent("http.request.started", trace.WithAttributes(
		attribute.String("http.request.method", "POST"),
	))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		pkgotel.RecordError(span, err, pkgotel.ErrorTypeNetwork, true)
		c.recordHTTPMetrics(ctx, start, 0, requestSize, "network_error")
		c.recordSendMetrics(ctx, start, "error")
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Add span event for response received
	span.AddEvent("http.response.received", trace.WithAttributes(
		attribute.Int("http.response.status_code", resp.StatusCode),
	))

	span.SetAttributes(
		attribute.Int("http.response.status_code", resp.StatusCode),
	)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("Loki returned status %d", resp.StatusCode)
		pkgotel.RecordError(span, err, pkgotel.ErrorTypeHTTP, false)
		c.recordHTTPMetrics(ctx, start, resp.StatusCode, requestSize, "")
		c.recordSendMetrics(ctx, start, "error")
		return err
	}

	// Record successful metrics
	c.recordHTTPMetrics(ctx, start, resp.StatusCode, requestSize, "")
	c.recordSendMetrics(ctx, start, "success")

	// Record vehicles sent to Loki
	c.recordVehiclesSent(ctx, data.LineRef, len(data.VehicleData))

	// Set span status to Ok on success
	pkgotel.SetSpanOk(span)

	return nil
}

// recordHTTPMetrics records HTTP client metrics for Loki API calls
func (c *Client) recordHTTPMetrics(ctx context.Context, start time.Time, statusCode int, requestSize int64, errorType string) {
	if !metrics.IsEnabled() {
		return
	}

	duration := time.Since(start).Seconds()

	// Common attributes
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", "POST"),
		attribute.String("server.address", c.serverHost),
		attribute.String("service.target", "loki"),
	}

	if statusCode > 0 {
		attrs = append(attrs, attribute.Int("http.response.status_code", statusCode))
	}
	if errorType != "" {
		attrs = append(attrs, attribute.String("error.type", errorType))
	}

	// Record duration
	metrics.HTTPClientRequestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))

	// Record request body size
	if requestSize > 0 {
		metrics.HTTPClientRequestBodySize.Record(ctx, requestSize, metric.WithAttributes(
			attribute.String("server.address", c.serverHost),
			attribute.String("service.target", "loki"),
		))
	}
}

// recordVehiclesSent records vehicles sent to Loki
func (c *Client) recordVehiclesSent(ctx context.Context, lineRef string, count int) {
	if !metrics.IsEnabled() {
		return
	}

	metrics.PipelineVehiclesProcessed.Add(ctx, int64(count), metric.WithAttributes(
		attribute.String("line_ref", lineRef),
		attribute.String("stage", "sent_to_loki"),
	))
}

// recordBatchMetrics records Loki batch metrics
func (c *Client) recordBatchMetrics(ctx context.Context, recordCount, streamCount int) {
	if !metrics.IsEnabled() {
		return
	}

	metrics.LokiBatchSize.Record(ctx, int64(recordCount))
	metrics.LokiBatchStreams.Record(ctx, int64(streamCount))
}

// recordSendMetrics records Loki send operation metrics
func (c *Client) recordSendMetrics(ctx context.Context, start time.Time, status string) {
	if !metrics.IsEnabled() {
		return
	}

	duration := time.Since(start).Seconds()

	// Record send duration
	metrics.LokiSendDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String("status", status),
	))

	// Record send total
	metrics.LokiSendTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))
}
