package otel

import (
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Protocol represents OTLP transport protocol
type Protocol string

const (
	ProtocolGRPC         Protocol = "grpc"
	ProtocolHTTPProtobuf Protocol = "http/protobuf"
	ProtocolHTTPJSON     Protocol = "http/json"
)

// SignalType represents the OTEL signal type
type SignalType string

const (
	SignalTraces  SignalType = "traces"
	SignalMetrics SignalType = "metrics"
)

// ExporterConfig holds parsed OTLP exporter configuration for a signal
type ExporterConfig struct {
	Endpoint    string
	Protocol    Protocol
	Headers     map[string]string
	Timeout     time.Duration
	Insecure    bool
	Compression string
}

// IsTracingEnabled returns true if OTEL tracing is enabled
func IsTracingEnabled() bool {
	return isTrue(getEnv("OTEL_TRACING_ENABLED", "false"))
}

// IsMetricsEnabled returns true if OTEL metrics is enabled
func IsMetricsEnabled() bool {
	return isTrue(getEnv("OTEL_METRICS_ENABLED", "false"))
}

// IsGrafanaCloudHostMetricEnabled returns true if the Grafana Cloud host hours billing metric is enabled.
// When enabled, emits the traces_host_info metric required for Application Observability billing.
func IsGrafanaCloudHostMetricEnabled() bool {
	return isTrue(getEnv("GC_ENABLE_HOSTHOURS_METRIC", "false"))
}

// GetExporterConfig returns the exporter configuration for a specific signal type.
// It resolves signal-specific environment variables with fallback to base variables.
func GetExporterConfig(signal SignalType) ExporterConfig {
	signalUpper := strings.ToUpper(string(signal))

	// Resolve protocol first as it affects default endpoint
	protocol := resolveProtocol(signalUpper)

	// Resolve endpoint with signal-specific override
	endpoint := resolveEndpoint(signal, signalUpper, protocol)

	// Resolve other configuration with fallbacks
	headers := parseHeaders(getEnvWithFallback(
		"OTEL_EXPORTER_OTLP_"+signalUpper+"_HEADERS",
		"OTEL_EXPORTER_OTLP_HEADERS",
		"",
	))

	timeout := parseDuration(getEnvWithFallback(
		"OTEL_EXPORTER_OTLP_"+signalUpper+"_TIMEOUT",
		"OTEL_EXPORTER_OTLP_TIMEOUT",
		"10s",
	), 10*time.Second)

	insecure := resolveInsecure(signalUpper, endpoint)

	compression := getEnvWithFallback(
		"OTEL_EXPORTER_OTLP_"+signalUpper+"_COMPRESSION",
		"OTEL_EXPORTER_OTLP_COMPRESSION",
		"",
	)

	return ExporterConfig{
		Endpoint:    endpoint,
		Protocol:    protocol,
		Headers:     headers,
		Timeout:     timeout,
		Insecure:    insecure,
		Compression: compression,
	}
}

// resolveProtocol determines the protocol from environment variables
func resolveProtocol(signalUpper string) Protocol {
	protocolStr := getEnvWithFallback(
		"OTEL_EXPORTER_OTLP_"+signalUpper+"_PROTOCOL",
		"OTEL_EXPORTER_OTLP_PROTOCOL",
		"http/protobuf",
	)

	switch strings.ToLower(protocolStr) {
	case "grpc":
		return ProtocolGRPC
	case "http/json":
		return ProtocolHTTPJSON
	default:
		return ProtocolHTTPProtobuf
	}
}

// resolveEndpoint determines the endpoint, handling path appending for base endpoints
func resolveEndpoint(signal SignalType, signalUpper string, protocol Protocol) string {
	// Check signal-specific endpoint first (use as-is, no path appending)
	signalEndpoint := getEnv("OTEL_EXPORTER_OTLP_"+signalUpper+"_ENDPOINT", "")
	if signalEndpoint != "" {
		return normalizeEndpoint(signalEndpoint, protocol)
	}

	// Check base endpoint (append signal path)
	baseEndpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	if baseEndpoint != "" {
		return appendSignalPath(normalizeEndpoint(baseEndpoint, protocol), signal, protocol)
	}

	// Return default endpoint based on protocol
	return getDefaultEndpoint(signal, protocol)
}

// normalizeEndpoint ensures the endpoint has proper format
func normalizeEndpoint(endpoint string, protocol Protocol) string {
	// For gRPC, we just need host:port
	if protocol == ProtocolGRPC {
		// Strip any scheme if present
		endpoint = strings.TrimPrefix(endpoint, "http://")
		endpoint = strings.TrimPrefix(endpoint, "https://")
		// Remove any path
		if idx := strings.Index(endpoint, "/"); idx != -1 {
			endpoint = endpoint[:idx]
		}
		return endpoint
	}

	// For HTTP, ensure we have a scheme
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}
	return endpoint
}

// appendSignalPath appends the signal-specific path for base endpoints
func appendSignalPath(endpoint string, signal SignalType, protocol Protocol) string {
	// gRPC doesn't use URL paths
	if protocol == ProtocolGRPC {
		return endpoint
	}

	signalPath := "/v1/" + string(signal)

	// Parse URL to properly append path
	u, err := url.Parse(endpoint)
	if err != nil {
		// Fallback: just append
		return strings.TrimSuffix(endpoint, "/") + signalPath
	}

	// Don't append if path already contains the signal path
	if strings.HasSuffix(u.Path, signalPath) {
		return endpoint
	}

	// Append the signal path
	if u.Path == "" || u.Path == "/" {
		u.Path = signalPath
	} else {
		u.Path = strings.TrimSuffix(u.Path, "/") + signalPath
	}

	return u.String()
}

// getDefaultEndpoint returns the default endpoint for a signal and protocol
func getDefaultEndpoint(signal SignalType, protocol Protocol) string {
	if protocol == ProtocolGRPC {
		return "localhost:4317"
	}
	// HTTP default includes the signal path
	return "http://localhost:4318/v1/" + string(signal)
}

// resolveInsecure determines whether to use insecure mode
func resolveInsecure(signalUpper string, endpoint string) bool {
	// Check explicit insecure setting first
	insecureStr := getEnvWithFallback(
		"OTEL_EXPORTER_OTLP_"+signalUpper+"_INSECURE",
		"OTEL_EXPORTER_OTLP_INSECURE",
		"",
	)
	if insecureStr != "" {
		return isTrue(insecureStr)
	}

	// Infer from endpoint scheme
	return strings.HasPrefix(endpoint, "http://")
}

// getEnv returns the value of an environment variable or a default value if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvWithFallback checks signal-specific env var, then base, then default
func getEnvWithFallback(signalSpecific, base, defaultValue string) string {
	if value := os.Getenv(signalSpecific); value != "" {
		return value
	}
	if value := os.Getenv(base); value != "" {
		return value
	}
	return defaultValue
}

// isTrue checks if a string represents a true value
func isTrue(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// parseHeaders parses header string in format "key1=value1,key2=value2"
// For Grafana Cloud OTEL: "Authorization=Basic base64credentials"
func parseHeaders(headerStr string) map[string]string {
	headers := make(map[string]string)
	if headerStr == "" {
		return headers
	}

	pairs := strings.Split(headerStr, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		// Use Index instead of SplitN to preserve the full value after first =
		if idx := strings.Index(pair, "="); idx > 0 {
			key := strings.TrimSpace(pair[:idx])
			value := pair[idx+1:] // Don't trim value - preserve exact content
			headers[key] = value
			slog.Debug("Parsed OTEL header", "key", key, "value_length", len(value))
		}
	}

	return headers
}

// parseDuration parses a duration string, returning default on failure.
// Supports both Go duration format ("10s", "1m") and OTEL spec milliseconds ("10000").
func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	// Try parsing as Go duration first (e.g., "10s", "1m")
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	// Try parsing as milliseconds integer (OTEL spec format)
	if ms, err := strconv.Atoi(s); err == nil {
		return time.Duration(ms) * time.Millisecond
	}
	return defaultVal
}
