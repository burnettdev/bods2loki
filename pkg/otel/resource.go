package otel

import (
	"context"
	"os"
	"runtime"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

const (
	// ServiceName is the name of this service
	ServiceName = "bods2loki"
	// ServiceVersion is the version of this service
	ServiceVersion = "1.0.0"
)

// NewResource creates a shared resource with service and runtime attributes.
// This resource is used by both tracing and metrics providers.
func NewResource() (*resource.Resource, error) {
	return resource.New(context.Background(),
		resource.WithAttributes(
			// Service identification
			semconv.ServiceName(ServiceName),
			semconv.ServiceVersion(ServiceVersion),

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
