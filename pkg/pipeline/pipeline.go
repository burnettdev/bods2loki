package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"bods2loki/pkg/bods"
	"bods2loki/pkg/loki"
	"bods2loki/pkg/metrics"
	pkgotel "bods2loki/pkg/otel"
	"bods2loki/pkg/parser"
	"bods2loki/pkg/types"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Pipeline struct {
	config     Config
	bodsClient *bods.Client
	lokiClient *loki.Client
	parser     *parser.XMLParser
	tracer     trace.Tracer
}

type Config struct {
	DryRun       bool
	APIKey       string
	DatasetID    string
	LineRefs     []string
	LokiURL      string
	LokiUser     string
	LokiPassword string
	Interval     time.Duration
}

func New(config Config) (*Pipeline, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if len(config.LineRefs) == 0 {
		return nil, fmt.Errorf("at least one line reference is required")
	}

	pipeline := &Pipeline{
		config:     config,
		bodsClient: bods.NewClient(config.APIKey, config.DatasetID),
		parser:     parser.NewXMLParser(),
		tracer:     otel.Tracer("pipeline"),
	}

	// Only create Loki client if not in dry run mode
	if !config.DryRun {
		pipeline.lokiClient = loki.NewClient(config.LokiURL, config.LokiUser, config.LokiPassword)
	}

	return pipeline, nil
}

func (p *Pipeline) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	slog.Info("Pipeline started", "interval", p.config.Interval)

	// Process immediately on start
	if err := p.processOnce(ctx); err != nil {
		slog.Error("Error in initial processing", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			slog.Info("Pipeline stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := p.processOnce(ctx); err != nil {
				slog.Error("Error processing", "error", err)
			}
		}
	}
}

func (p *Pipeline) processOnce(ctx context.Context) error {
	ctx, span := p.tracer.Start(ctx, "pipeline.process_once",
		trace.WithAttributes(
			attribute.StringSlice("line_refs", p.config.LineRefs),
			attribute.Bool("dry_run", p.config.DryRun),
			attribute.Int("lines_count", len(p.config.LineRefs)),
		),
	)
	defer span.End()

	// Add baggage for downstream spans
	member, _ := baggage.NewMember("pipeline.dry_run", strconv.FormatBool(p.config.DryRun))
	bag, _ := baggage.New(member)
	ctx = baggage.ContextWithBaggage(ctx, bag)

	// Add event for cycle start
	span.AddEvent("cycle.started", trace.WithAttributes(
		attribute.Int("lines.count", len(p.config.LineRefs)),
	))

	start := time.Now()

	// Record lines attempted this cycle
	p.recordCycleLinesTotal(ctx, len(p.config.LineRefs))

	// Process all lines concurrently
	type lineResult struct {
		lineRef       string
		data          *types.ParsedBusData
		err           error
		fetchDuration time.Duration
		parseDuration time.Duration
	}

	results := make(chan lineResult, len(p.config.LineRefs))

	// Start concurrent fetching for each line
	for _, lineRef := range p.config.LineRefs {
		// Track in-flight lines
		p.recordLineInFlight(ctx, 1)

		go func(line string) {
			defer p.recordLineInFlight(ctx, -1)

			lineCtx, lineSpan := p.tracer.Start(ctx, "pipeline.process_line",
				trace.WithAttributes(attribute.String("line_ref", line)),
			)
			defer lineSpan.End()

			// Fetch data from BODS API
			fetchStart := time.Now()
			busData, err := p.bodsClient.FetchBusData(lineCtx, line)
			fetchDuration := time.Since(fetchStart)
			if err != nil {
				lineSpan.RecordError(err)
				p.recordStageDuration(lineCtx, "fetch", line, fetchDuration)
				results <- lineResult{lineRef: line, err: fmt.Errorf("failed to fetch bus data for line %s: %w", line, err), fetchDuration: fetchDuration}
				return
			}
			p.recordStageDuration(lineCtx, "fetch", line, fetchDuration)

			// Parse XML to JSON
			parseStart := time.Now()
			parsedData, err := p.parser.ParseBusData(lineCtx, busData)
			parseDuration := time.Since(parseStart)
			if err != nil {
				lineSpan.RecordError(err)
				p.recordStageDuration(lineCtx, "parse", line, parseDuration)
				results <- lineResult{lineRef: line, err: fmt.Errorf("failed to parse bus data for line %s: %w", line, err), fetchDuration: fetchDuration, parseDuration: parseDuration}
				return
			}
			p.recordStageDuration(lineCtx, "parse", line, parseDuration)

			lineSpan.SetAttributes(
				attribute.Int("vehicles_processed", len(parsedData.VehicleData)),
			)

			// Record parsed vehicles
			p.recordVehiclesParsed(ctx, line, len(parsedData.VehicleData))

			results <- lineResult{lineRef: line, data: parsedData, err: nil, fetchDuration: fetchDuration, parseDuration: parseDuration}
		}(lineRef)
	}

	// Collect results
	var allData []*types.ParsedBusData
	var errors []error
	totalVehicles := 0

	for i := 0; i < len(p.config.LineRefs); i++ {
		result := <-results
		if result.err != nil {
			errors = append(errors, result.err)
			slog.Error("Error processing line", "line", result.lineRef, "error", result.err)
			// Record line failure
			errorType := p.categorizeError(result.err)
			p.recordLineProcessed(ctx, result.lineRef, "error", errorType)
			// Record error metric
			p.recordError(ctx, errorType, result.lineRef)
			// Record cycle line failed
			p.recordCycleLinesFailed(ctx, 1)
		} else {
			allData = append(allData, result.data)
			totalVehicles += len(result.data.VehicleData)
			// Record line success
			p.recordLineProcessed(ctx, result.lineRef, "success", "")
			// Add event for line completion
			span.AddEvent("line.completed", trace.WithAttributes(
				attribute.String("line_ref", result.lineRef),
				attribute.Int("vehicles.count", len(result.data.VehicleData)),
			))
		}
	}

	span.SetAttributes(
		attribute.Int("total_vehicles_processed", totalVehicles),
		attribute.Int("successful_lines", len(allData)),
		attribute.Int("failed_lines", len(errors)),
		attribute.String("processing_duration", time.Since(start).String()),
	)

	// Process successful results
	for _, data := range allData {
		if p.config.DryRun {
			if err := p.handleDryRun(ctx, data); err != nil {
				slog.Error("Error in dry run", "line", data.LineRef, "error", err)
			}
		} else {
			if err := p.sendToLoki(ctx, data); err != nil {
				slog.Error("Error sending to Loki", "line", data.LineRef, "error", err)
			}
		}
	}

	// Determine cycle status and record metrics
	cycleStatus := p.determineCycleStatus(len(allData), len(errors), len(p.config.LineRefs))
	p.recordCycleMetrics(ctx, start, cycleStatus, len(p.config.LineRefs))

	// Return error only if all lines failed
	if len(errors) == len(p.config.LineRefs) {
		err := fmt.Errorf("all lines failed: %v", errors)
		pkgotel.RecordError(span, err, pkgotel.ErrorTypeNetwork, true)
		return err
	}

	// Set span status to Ok on success (even partial success)
	pkgotel.SetSpanOk(span)
	return nil
}

func (p *Pipeline) handleDryRun(ctx context.Context, data *types.ParsedBusData) error {
	_, span := p.tracer.Start(ctx, "pipeline.dry_run")
	defer span.End()

	// Print summary information
	fmt.Printf("\n=== DRY RUN - Bus Data for Line %s ===\n", data.LineRef)
	fmt.Printf("Timestamp: %s\n", data.Timestamp)
	fmt.Printf("Vehicles Found: %d\n", len(data.VehicleData))

	if len(data.VehicleData) > 0 {
		fmt.Println("\nVehicle Summary:")
		for i, vehicle := range data.VehicleData {
			route := ""
			if vehicle.OriginName != "" && vehicle.DestinationName != "" {
				route = fmt.Sprintf(" (%s → %s)", vehicle.OriginName, vehicle.DestinationName)
			}
			fmt.Printf("  %d. Vehicle: %s, Direction: %s, Location: (%.6f, %.6f)%s\n",
				i+1, vehicle.VehicleRef, vehicle.DirectionRef, vehicle.Latitude, vehicle.Longitude, route)
		}
	}

	fmt.Println("\nIndividual Log Lines (as sent to Loki):")
	fmt.Println("----------------------------------------")

	// Show individual log lines as they would be sent to Loki
	for i, vehicle := range data.VehicleData {
		// Create log entry with embedded vehicle data (same format as Loki client)
		// All VehicleActivity fields are automatically included
		// Missing fields are silently omitted via omitempty tags
		entry := struct {
			Timestamp string `json:"timestamp"`
			LineRef   string `json:"line_ref"`
			types.VehicleActivity
		}{
			Timestamp:       data.Timestamp,
			LineRef:         data.LineRef,
			VehicleActivity: vehicle,
		}

		// Convert to JSON
		vehicleJSON, err := json.Marshal(entry)
		if err != nil {
			pkgotel.RecordError(span, err, pkgotel.ErrorTypeParse, false)
			return fmt.Errorf("failed to marshal vehicle JSON for dry run: %w", err)
		}

		fmt.Printf("Log Line %d: %s\n", i+1, string(vehicleJSON))
	}

	fmt.Println("=== END DRY RUN ===")

	span.SetAttributes(
		attribute.Int("vehicles_printed", len(data.VehicleData)),
	)

	pkgotel.SetSpanOk(span)
	return nil
}

func (p *Pipeline) sendToLoki(ctx context.Context, data *types.ParsedBusData) error {
	ctx, span := p.tracer.Start(ctx, "pipeline.send_to_loki")
	defer span.End()

	if p.lokiClient == nil {
		err := fmt.Errorf("loki client not initialized")
		pkgotel.RecordError(span, err, pkgotel.ErrorTypeValidation, false)
		return err
	}

	if err := p.lokiClient.SendBusData(ctx, data); err != nil {
		pkgotel.RecordError(span, err, pkgotel.ErrorTypeNetwork, true)
		return fmt.Errorf("failed to send data to Loki: %w", err)
	}

	slog.Debug("Successfully sent vehicle log lines to Loki", "count", len(data.VehicleData), "line", data.LineRef)

	span.SetAttributes(
		attribute.Int("vehicles_sent", len(data.VehicleData)),
	)

	pkgotel.SetSpanOk(span)
	return nil
}

// determineCycleStatus determines the status of a processing cycle
func (p *Pipeline) determineCycleStatus(successCount, errorCount, totalCount int) string {
	if errorCount == 0 {
		return "success"
	}
	if errorCount == totalCount {
		return "total_failure"
	}
	return "partial_failure"
}

// categorizeError categorizes an error for metrics
func (p *Pipeline) categorizeError(err error) string {
	if err == nil {
		return ""
	}
	errStr := err.Error()
	if contains(errStr, "fetch") {
		return "fetch_failed"
	}
	if contains(errStr, "parse") {
		return "parse_failed"
	}
	if contains(errStr, "send") || contains(errStr, "loki") {
		return "send_failed"
	}
	return "unknown"
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsLower(s, substr)))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if matchLower(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func matchLower(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// Metrics recording functions

func (p *Pipeline) recordCycleMetrics(ctx context.Context, start time.Time, status string, linesCount int) {
	if !metrics.IsEnabled() {
		return
	}

	duration := time.Since(start).Seconds()

	// Record cycle count
	metrics.PipelineCyclesTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
	))

	// Record cycle duration
	metrics.PipelineCycleDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String("status", status),
		attribute.Int("lines_count", linesCount),
	))

	// Record last success timestamp if successful
	if status == "success" {
		metrics.RecordLastSuccessTimestamp()
	}
}

func (p *Pipeline) recordLineProcessed(ctx context.Context, lineRef, status, errorType string) {
	if !metrics.IsEnabled() {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("line_ref", lineRef),
		attribute.String("status", status),
	}
	if errorType != "" {
		attrs = append(attrs, attribute.String("error.type", errorType))
	}

	metrics.PipelineLinesProcessed.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (p *Pipeline) recordVehiclesParsed(ctx context.Context, lineRef string, count int) {
	if !metrics.IsEnabled() {
		return
	}

	metrics.PipelineVehiclesProcessed.Add(ctx, int64(count), metric.WithAttributes(
		attribute.String("line_ref", lineRef),
		attribute.String("stage", "parsed"),
	))
}

func (p *Pipeline) recordLineInFlight(ctx context.Context, delta int64) {
	if !metrics.IsEnabled() {
		return
	}

	metrics.PipelineLinesInFlight.Add(ctx, delta)
}

func (p *Pipeline) recordError(ctx context.Context, errorType, lineRef string) {
	if !metrics.IsEnabled() {
		return
	}

	stage := "unknown"
	switch errorType {
	case "fetch_failed":
		stage = "fetch"
	case "parse_failed":
		stage = "parse"
	case "send_failed":
		stage = "send"
	}

	metrics.PipelineErrorsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("pipeline.stage", stage),
		attribute.String("error.type", errorType),
		attribute.String("line_ref", lineRef),
	))
}

func (p *Pipeline) recordStageDuration(ctx context.Context, stage, lineRef string, duration time.Duration) {
	if !metrics.IsEnabled() {
		return
	}

	metrics.PipelineStageDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("pipeline.stage", stage),
		attribute.String("line_ref", lineRef),
	))
}

func (p *Pipeline) recordCycleLinesTotal(ctx context.Context, count int) {
	if !metrics.IsEnabled() {
		return
	}

	metrics.PipelineCycleLinesTotal.Add(ctx, int64(count))
}

func (p *Pipeline) recordCycleLinesFailed(ctx context.Context, count int) {
	if !metrics.IsEnabled() {
		return
	}

	metrics.PipelineCycleLinesFailed.Add(ctx, int64(count))
}
