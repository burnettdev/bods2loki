package tracing

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"runtime"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func InitTracing() (func(), error) {
	// Check if tracing is enabled
	if enabled := getEnv("OTEL_TRACING_ENABLED", "false"); !isTrue(enabled) {
		slog.Debug("OpenTelemetry tracing is disabled")
		return func() {}, nil
	}

	// Get parsed OTLP endpoint configuration
	endpointConfig := parseOTLPEndpoint()

	// Parse headers if provided
	headers := parseHeaders(getEnv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", ""))

	// Create OTLP exporter options with properly parsed host
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpointConfig.Host),
	}

	// Add URL path if specified
	if endpointConfig.Path != "" {
		opts = append(opts, otlptracehttp.WithURLPath(endpointConfig.Path))
	}

	// Determine insecure mode: explicit env var takes precedence, else use parsed scheme
	insecureEnv := getEnv("OTEL_EXPORTER_OTLP_TRACES_INSECURE", "")
	if insecureEnv != "" {
		if isTrue(insecureEnv) {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
	} else if endpointConfig.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if len(headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(headers))
	}

	// Create OTLP exporter
	exporter, err := otlptracehttp.New(context.Background(), opts...)
	if err != nil {
		slog.Warn("Failed to create OTLP exporter, using noop", "error", err)
		// Return a noop shutdown function if exporter creation fails
		return func() {}, nil
	}

	// Create resource with Go-specific attributes
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			// Service identification
			semconv.ServiceName("bods2loki"),
			semconv.ServiceVersion("1.0.0"),

			// Process and runtime information
			semconv.ProcessRuntimeName("go"),
			semconv.ProcessRuntimeVersion(runtime.Version()),
			semconv.ProcessRuntimeDescription("Go runtime"),
			semconv.ProcessPID(os.Getpid()),

			// Telemetry SDK information
			semconv.TelemetrySDKName("opentelemetry"),
			semconv.TelemetrySDKLanguageGo,
			semconv.TelemetrySDKVersion("1.21.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create trace provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			slog.Error("Error shutting down tracer provider", "error", err)
		}
	}, nil
}

// getEnv returns the value of an environment variable or a default value if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// isTrue checks if a string represents a true value
func isTrue(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// otlpEndpointConfig holds parsed OTLP endpoint configuration
type otlpEndpointConfig struct {
	Host     string // host:port for WithEndpoint()
	Path     string // URL path for WithURLPath()
	Insecure bool   // true for http://, false for https://
}

// parseOTLPEndpoint parses the OTLP endpoint from environment variables
// and extracts host, path, and scheme information for proper configuration.
func parseOTLPEndpoint() otlpEndpointConfig {
	endpoint := getEnv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	appendTracesPath := false

	if endpoint == "" {
		endpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
		appendTracesPath = true
	}

	if endpoint == "" {
		return otlpEndpointConfig{
			Host:     "localhost:4318",
			Path:     "",
			Insecure: true,
		}
	}

	// Add default scheme if missing (default to https for security)
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		// Fallback to treating as host:port
		slog.Warn("Failed to parse OTLP endpoint URL, using as-is", "error", err)
		return otlpEndpointConfig{Host: endpoint, Insecure: true}
	}

	path := u.Path
	if appendTracesPath && !strings.HasSuffix(path, "/v1/traces") {
		if path == "" || path == "/" {
			path = "/v1/traces"
		} else {
			path = strings.TrimSuffix(path, "/") + "/v1/traces"
		}
	}

	return otlpEndpointConfig{
		Host:     u.Host,
		Path:     path,
		Insecure: u.Scheme == "http",
	}
}

// parseHeaders parses header string in format "key1=value1,key2=value2"
func parseHeaders(headerStr string) map[string]string {
	headers := make(map[string]string)
	if headerStr == "" {
		return headers
	}

	pairs := strings.Split(headerStr, ",")
	for _, pair := range pairs {
		if kv := strings.SplitN(strings.TrimSpace(pair), "=", 2); len(kv) == 2 {
			headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	return headers
}
