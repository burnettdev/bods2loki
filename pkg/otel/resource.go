package otel

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

const (
	// ServiceName is the name of this service
	ServiceName = "bods2loki"
)

// Version is set at build time via -ldflags
// e.g., go build -ldflags="-X bods2loki/pkg/otel.Version=1.2.3"
var Version = "dev"

// getEnvOrDefault returns the environment variable value or a default
func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// getServiceNamespace returns the service namespace for logical grouping.
// Used by Grafana Cloud App Observability for service grouping.
func getServiceNamespace() string {
	return getEnvOrDefault("OTEL_SERVICE_NAMESPACE", "bods2loki")
}

// getDeploymentEnvironment returns the deployment environment (dev/staging/prod).
// Used by Grafana Cloud App Observability for environment filtering.
func getDeploymentEnvironment() string {
	return getEnvOrDefault("OTEL_DEPLOYMENT_ENVIRONMENT", "production")
}

// getServiceInstanceID returns a unique identifier for this service instance.
// Priority: OTEL_SERVICE_INSTANCE_ID env var > hostname > process ID fallback
func getServiceInstanceID() string {
	if id := os.Getenv("OTEL_SERVICE_INSTANCE_ID"); id != "" {
		return id
	}
	// Use hostname as default - unique per container/pod
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return hostname
	}
	// Fallback to process ID (less ideal but always available)
	return fmt.Sprintf("bods2loki-%d", os.Getpid())
}

// NewResource creates a shared resource with service and runtime attributes.
// This resource is used by both tracing and metrics providers.
// Includes attributes required for Grafana Cloud Application Observability.
func NewResource() (*resource.Resource, error) {
	return resource.New(context.Background(),
		// Enable automatic env var detection (OTEL_SERVICE_NAME, OTEL_RESOURCE_ATTRIBUTES)
		resource.WithFromEnv(),
		// Auto-detect host metadata
		resource.WithHost(),
		// Auto-detect process info
		resource.WithProcess(),
		resource.WithAttributes(
			// Service identification (Grafana Cloud App Observability requirements)
			semconv.ServiceName(ServiceName),
			semconv.ServiceVersion(Version),
			semconv.ServiceNamespace(getServiceNamespace()),
			semconv.ServiceInstanceID(getServiceInstanceID()),

			// Deployment context (required for App Observability environment filtering)
			semconv.DeploymentEnvironment(getDeploymentEnvironment()),

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
}
