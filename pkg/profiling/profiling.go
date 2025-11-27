package profiling

import (
	"log/slog"
	"os"
	"strings"

	"github.com/grafana/pyroscope-go"
)

func InitProfiling() (func(), error) {
	// Check if profiling is enabled
	if enabled := getEnv("PYROSCOPE_PROFILING_ENABLED", "false"); !isTrue(enabled) {
		slog.Debug("Pyroscope profiling is disabled")
		return func() {}, nil
	}

	// Get Pyroscope server address
	serverAddress := getEnv("PYROSCOPE_SERVER_ADDRESS", "http://localhost:4040")

	// Get application name
	applicationName := getEnv("PYROSCOPE_APPLICATION_NAME", "bods2loki")

	// Get basic auth credentials if provided
	basicAuthUser := getEnv("PYROSCOPE_BASIC_AUTH_USER", "")
	basicAuthPassword := getEnv("PYROSCOPE_BASIC_AUTH_PASSWORD", "")

	// Create Pyroscope config
	config := pyroscope.Config{
		ApplicationName: applicationName,
		ServerAddress:   serverAddress,
		Logger:          pyroscope.StandardLogger,
		Tags: map[string]string{
			"service": "bods2loki",
			"version": "1.0.0",
		},
	}

	// Add basic authentication if provided
	if basicAuthUser != "" && basicAuthPassword != "" {
		config.BasicAuthUser = basicAuthUser
		config.BasicAuthPassword = basicAuthPassword
	}

	// Start profiling
	profiler, err := pyroscope.Start(config)
	if err != nil {
		slog.Warn("Failed to start Pyroscope profiler", "error", err)
		// Return a noop shutdown function if profiler creation fails
		return func() {}, nil
	}

	slog.Debug("Pyroscope profiling started", "server", serverAddress, "application", applicationName)

	// Return shutdown function
	return func() {
		if err := profiler.Stop(); err != nil {
			slog.Error("Error stopping Pyroscope profiler", "error", err)
		} else {
			slog.Debug("Pyroscope profiler stopped")
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
