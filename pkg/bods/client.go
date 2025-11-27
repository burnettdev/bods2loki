package bods

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"bods2loki/pkg/metrics"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	BaseURLTemplate = "https://data.bus-data.dft.gov.uk/api/v1/datafeed/%s/"
)

type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	serverHost string
	tracer     trace.Tracer
}

type BusData struct {
	XMLData   string
	Timestamp time.Time
	LineRef   string
}

func NewClient(apiKey, datasetID string) *Client {
	// Create HTTP client with OpenTelemetry instrumentation
	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   30 * time.Second,
	}

	baseURL := fmt.Sprintf(BaseURLTemplate, datasetID)

	// Extract server host for metrics
	parsedURL, _ := url.Parse(baseURL)
	serverHost := ""
	if parsedURL != nil {
		serverHost = parsedURL.Host
	}

	return &Client{
		httpClient: client,
		apiKey:     apiKey,
		baseURL:    baseURL,
		serverHost: serverHost,
		tracer:     otel.Tracer("bods-client"),
	}
}

func (c *Client) FetchBusData(ctx context.Context, lineRef string) (*BusData, error) {
	ctx, span := c.tracer.Start(ctx, "bods.fetch_bus_data",
		trace.WithAttributes(
			attribute.String("line_ref", lineRef),
			attribute.String("api.endpoint", c.baseURL),
		),
	)
	defer span.End()

	start := time.Now()

	// Build URL with parameters
	reqURL := fmt.Sprintf("%s?api_key=%s&lineRef=%s", c.baseURL, c.apiKey, lineRef)

	span.SetAttributes(
		attribute.String("http.url", reqURL),
		attribute.String("http.method", "GET"),
	)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		span.RecordError(err)
		c.recordHTTPMetrics(ctx, start, 0, 0, "request_creation_error")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "bods2loki/1.0.0")
	req.Header.Set("Accept", "*/*")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		c.recordHTTPMetrics(ctx, start, 0, 0, "network_error")
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.String("http.response.content_type", resp.Header.Get("Content-Type")),
	)

	if resp.StatusCode != http.StatusOK {
		// Read the error response body for debugging
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		span.RecordError(err)
		c.recordHTTPMetrics(ctx, start, resp.StatusCode, int64(len(body)), "")
		return nil, err
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		c.recordHTTPMetrics(ctx, start, resp.StatusCode, 0, "read_error")
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	span.SetAttributes(
		attribute.Int("response.size_bytes", len(body)),
	)

	// Record successful metrics
	c.recordHTTPMetrics(ctx, start, resp.StatusCode, int64(len(body)), "")

	return &BusData{
		XMLData:   string(body),
		Timestamp: time.Now(),
		LineRef:   lineRef,
	}, nil
}

// recordHTTPMetrics records HTTP client metrics for BODS API calls
func (c *Client) recordHTTPMetrics(ctx context.Context, start time.Time, statusCode int, responseSize int64, errorType string) {
	if !metrics.IsEnabled() {
		return
	}

	duration := time.Since(start).Seconds()

	// Common attributes
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", "GET"),
		attribute.String("server.address", c.serverHost),
		attribute.String("service.target", "bods_api"),
	}

	if statusCode > 0 {
		attrs = append(attrs, attribute.Int("http.response.status_code", statusCode))
	}
	if errorType != "" {
		attrs = append(attrs, attribute.String("error.type", errorType))
	}

	// Record duration
	metrics.HTTPClientRequestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))

	// Record response body size if available
	if responseSize > 0 {
		metrics.HTTPClientResponseBodySize.Record(ctx, responseSize, metric.WithAttributes(
			attribute.String("server.address", c.serverHost),
			attribute.String("service.target", "bods_api"),
		))
	}
}
