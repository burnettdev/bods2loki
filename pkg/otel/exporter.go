package otel

import (
	"context"
	"fmt"
	"net/url"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewTraceExporter creates a trace exporter based on the protocol configuration
func NewTraceExporter(ctx context.Context, cfg ExporterConfig) (*otlptrace.Exporter, error) {
	switch cfg.Protocol {
	case ProtocolGRPC:
		return createGRPCTraceExporter(ctx, cfg)
	case ProtocolHTTPProtobuf, ProtocolHTTPJSON:
		return createHTTPTraceExporter(ctx, cfg)
	default:
		return createHTTPTraceExporter(ctx, cfg)
	}
}

// NewMetricExporter creates a metric exporter based on the protocol configuration
func NewMetricExporter(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
	switch cfg.Protocol {
	case ProtocolGRPC:
		return createGRPCMetricExporter(ctx, cfg)
	case ProtocolHTTPProtobuf, ProtocolHTTPJSON:
		return createHTTPMetricExporter(ctx, cfg)
	default:
		return createHTTPMetricExporter(ctx, cfg)
	}
}

// createGRPCTraceExporter creates a gRPC-based trace exporter
func createGRPCTraceExporter(ctx context.Context, cfg ExporterConfig) (*otlptrace.Exporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithTimeout(cfg.Timeout),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	}

	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
	}

	if cfg.Compression == "gzip" {
		opts = append(opts, otlptracegrpc.WithCompressor("gzip"))
	}

	return otlptracegrpc.New(ctx, opts...)
}

// createHTTPTraceExporter creates an HTTP-based trace exporter
func createHTTPTraceExporter(ctx context.Context, cfg ExporterConfig) (*otlptrace.Exporter, error) {
	// Parse endpoint to extract host and path
	host, path, err := parseHTTPEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(host),
		otlptracehttp.WithTimeout(cfg.Timeout),
	}

	if path != "" {
		opts = append(opts, otlptracehttp.WithURLPath(path))
	}

	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
	}

	if cfg.Compression == "gzip" {
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
	}

	return otlptracehttp.New(ctx, opts...)
}

// createGRPCMetricExporter creates a gRPC-based metric exporter
func createGRPCMetricExporter(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
		otlpmetricgrpc.WithTimeout(cfg.Timeout),
	}

	if cfg.Insecure {
		opts = append(opts, otlpmetricgrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	}

	if len(cfg.Headers) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(cfg.Headers))
	}

	if cfg.Compression == "gzip" {
		opts = append(opts, otlpmetricgrpc.WithCompressor("gzip"))
	}

	return otlpmetricgrpc.New(ctx, opts...)
}

// createHTTPMetricExporter creates an HTTP-based metric exporter
func createHTTPMetricExporter(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
	// Parse endpoint to extract host and path
	host, path, err := parseHTTPEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}

	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(host),
		otlpmetrichttp.WithTimeout(cfg.Timeout),
	}

	if path != "" {
		opts = append(opts, otlpmetrichttp.WithURLPath(path))
	}

	if cfg.Insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	if len(cfg.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(cfg.Headers))
	}

	if cfg.Compression == "gzip" {
		opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
	}

	return otlpmetrichttp.New(ctx, opts...)
}

// parseHTTPEndpoint extracts host and path from an HTTP endpoint URL
func parseHTTPEndpoint(endpoint string) (host string, path string, err error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", err
	}

	// Host includes port if specified
	host = u.Host
	path = u.Path

	return host, path, nil
}
