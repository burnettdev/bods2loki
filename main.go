package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"bods2loki/pkg/logging"
	"bods2loki/pkg/metrics"
	"bods2loki/pkg/pipeline"
	"bods2loki/pkg/profiling"
	"bods2loki/pkg/tracing"
)

func main() {
	// Command line flags
	var (
		dryRun       = flag.Bool("dry-run", false, "Print data to stdout instead of sending to Loki")
		apiKey       = flag.String("api-key", getEnv("BODS_API_KEY", ""), "BODS API key (required)")
		datasetID    = flag.String("dataset-id", getEnv("BODS_DATASET_ID", "699"), "BODS dataset ID")
		lineRefs     = flag.String("line-refs", getEnv("BODS_LINE_REFS", "49x"), "Bus line references, comma-separated")
		lokiURL      = flag.String("loki-url", getEnv("BODS_LOKI_URL", "http://localhost:3100"), "Grafana Loki URL")
		lokiUser     = flag.String("loki-user", getEnv("BODS_LOKI_USER", ""), "Loki username (for Grafana Cloud authentication)")
		lokiPassword = flag.String("loki-password", getEnv("BODS_LOKI_PASSWORD", ""), "Loki password/token (for Grafana Cloud authentication)")
		interval     = flag.String("interval", getEnv("BODS_INTERVAL", "30s"), "Polling interval")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "BODS to Loki Bus Tracking Pipeline\n\n")
		fmt.Fprintf(os.Stderr, "Fetches live bus tracking data from the UK Department for Transport's\n")
		fmt.Fprintf(os.Stderr, "Bus Open Data Service (BODS), converts XML to JSON, and sends it to\n")
		fmt.Fprintf(os.Stderr, "Grafana Loki for log aggregation and analysis.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BODS_API_KEY      - Your BODS API key (required)\n")
		fmt.Fprintf(os.Stderr, "  BODS_DATASET_ID   - BODS dataset ID (default: 699)\n")
		fmt.Fprintf(os.Stderr, "  BODS_LINE_REFS    - Bus line references, comma-separated (default: 49x)\n")
		fmt.Fprintf(os.Stderr, "  BODS_LOKI_URL     - Loki URL (default: http://localhost:3100)\n")
		fmt.Fprintf(os.Stderr, "  BODS_LOKI_USER    - Loki username (for Grafana Cloud)\n")
		fmt.Fprintf(os.Stderr, "  BODS_LOKI_PASSWORD - Loki password/token (for Grafana Cloud)\n")
		fmt.Fprintf(os.Stderr, "  BODS_INTERVAL     - Polling interval (default: 30s)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Dry run mode (safe for testing)\n")
		fmt.Fprintf(os.Stderr, "  %s --dry-run --api-key=YOUR_API_KEY --line-refs=49x\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Production mode with OSS Loki\n")
		fmt.Fprintf(os.Stderr, "  %s --api-key=YOUR_API_KEY --line-refs=49x,7 --loki-url=http://localhost:3100\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Production mode with Grafana Cloud\n")
		fmt.Fprintf(os.Stderr, "  %s --api-key=YOUR_API_KEY --line-refs=49x,7 \\\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "    --loki-url=https://logs-prod-us-central1.grafana.net \\\n")
		fmt.Fprintf(os.Stderr, "    --loki-user=123456 --loki-password=your_token\n\n")
	}

	flag.Parse()

	// Initialize logging first
	logging.InitLogging()

	// Validate required parameters
	if *apiKey == "" {
		fmt.Fprintf(os.Stderr, "Error: API key is required. Use --api-key or set BODS_API_KEY environment variable.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Parse interval
	intervalDuration, err := time.ParseDuration(*interval)
	if err != nil {
		slog.Error("Invalid interval format", "error", err)
		os.Exit(1)
	}

	// Parse line references
	lineRefsList := strings.Split(*lineRefs, ",")
	for i, ref := range lineRefsList {
		lineRefsList[i] = strings.TrimSpace(ref)
	}

	// Initialize tracing
	shutdownTracing, err := tracing.InitTracing()
	if err != nil {
		slog.Error("Failed to initialize tracing", "error", err)
		os.Exit(1)
	}
	defer shutdownTracing()

	// Initialize metrics
	shutdownMetrics, err := metrics.InitMetrics()
	if err != nil {
		slog.Error("Failed to initialize metrics", "error", err)
		os.Exit(1)
	}
	defer shutdownMetrics()

	// Initialize profiling
	shutdownProfiling, err := profiling.InitProfiling()
	if err != nil {
		slog.Error("Failed to initialize profiling", "error", err)
		os.Exit(1)
	}
	defer shutdownProfiling()

	// Create pipeline configuration
	config := pipeline.Config{
		DryRun:       *dryRun,
		APIKey:       *apiKey,
		DatasetID:    *datasetID,
		LineRefs:     lineRefsList,
		LokiURL:      *lokiURL,
		LokiUser:     *lokiUser,
		LokiPassword: *lokiPassword,
		Interval:     intervalDuration,
	}

	// Create pipeline
	pipelineInstance, err := pipeline.New(config)
	if err != nil {
		slog.Error("Failed to create pipeline", "error", err)
		os.Exit(1)
	}

	// Print startup information
	if *dryRun {
		slog.Info("Starting BODS to Loki pipeline in DRY RUN mode")
		slog.Info("Data will be printed to stdout, not sent to Loki")
	} else {
		slog.Info("Starting BODS to Loki pipeline in PRODUCTION mode")
		slog.Info("Data will be sent to Loki", "url", *lokiURL)
	}
	slog.Info("Monitoring lines", "lines", lineRefsList)
	slog.Info("Polling interval", "interval", intervalDuration)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start pipeline in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- pipelineInstance.Run(ctx)
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		slog.Info("Received signal, shutting down gracefully", "signal", sig)
		cancel()
		// Wait a bit for graceful shutdown
		select {
		case <-time.After(5 * time.Second):
			slog.Warn("Shutdown timeout, forcing exit")
		case <-errChan:
			slog.Info("Pipeline stopped")
		}
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			slog.Error("Pipeline error", "error", err)
			os.Exit(1)
		}
		slog.Info("Pipeline stopped")
	}

	slog.Info("BODS to Loki pipeline shutdown complete")
}

// getEnv returns the value of an environment variable or a default value if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
