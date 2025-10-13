package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"bods2loki/pkg/bods"
	"bods2loki/pkg/loki"
	"bods2loki/pkg/parser"
	"bods2loki/pkg/types"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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

	log.Printf("Pipeline started - polling every %v", p.config.Interval)

	// Process immediately on start
	if err := p.processOnce(ctx); err != nil {
		log.Printf("Error in initial processing: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Pipeline stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := p.processOnce(ctx); err != nil {
				log.Printf("Error processing: %v", err)
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

	start := time.Now()

	// Process all lines concurrently
	type lineResult struct {
		lineRef string
		data    *types.ParsedBusData
		err     error
	}

	results := make(chan lineResult, len(p.config.LineRefs))

	// Start concurrent fetching for each line
	for _, lineRef := range p.config.LineRefs {
		go func(line string) {
			lineCtx, lineSpan := p.tracer.Start(ctx, "pipeline.process_line",
				trace.WithAttributes(attribute.String("line_ref", line)),
			)
			defer lineSpan.End()

			// Fetch data from BODS API
			busData, err := p.bodsClient.FetchBusData(lineCtx, line)
			if err != nil {
				lineSpan.RecordError(err)
				results <- lineResult{lineRef: line, err: fmt.Errorf("failed to fetch bus data for line %s: %w", line, err)}
				return
			}

			// Parse XML to JSON
			parsedData, err := p.parser.ParseBusData(lineCtx, busData)
			if err != nil {
				lineSpan.RecordError(err)
				results <- lineResult{lineRef: line, err: fmt.Errorf("failed to parse bus data for line %s: %w", line, err)}
				return
			}

			lineSpan.SetAttributes(
				attribute.Int("vehicles_processed", len(parsedData.VehicleData)),
			)

			results <- lineResult{lineRef: line, data: parsedData, err: nil}
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
			log.Printf("Error processing line %s: %v", result.lineRef, result.err)
		} else {
			allData = append(allData, result.data)
			totalVehicles += len(result.data.VehicleData)
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
				log.Printf("Error in dry run for line %s: %v", data.LineRef, err)
			}
		} else {
			if err := p.sendToLoki(ctx, data); err != nil {
				log.Printf("Error sending to Loki for line %s: %v", data.LineRef, err)
			}
		}
	}

	// Return error only if all lines failed
	if len(errors) == len(p.config.LineRefs) {
		return fmt.Errorf("all lines failed: %v", errors)
	}

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
		// Create individual vehicle log entry (same format as Loki client)
		vehicleLog := map[string]interface{}{
			"timestamp":                      data.Timestamp,
			"line_ref":                       data.LineRef,
			"vehicle_ref":                    vehicle.VehicleRef,
			"direction_ref":                  vehicle.DirectionRef,
			"operator_ref":                   vehicle.OperatorRef,
			"origin_ref":                     vehicle.OriginRef,
			"origin_name":                    vehicle.OriginName,
			"destination_ref":                vehicle.DestinationRef,
			"destination_name":               vehicle.DestinationName,
			"origin_aimed_departure_time":    vehicle.OriginAimedDepartureTime,
			"destination_aimed_arrival_time": vehicle.DestinationAimedArrivalTime,
			"longitude":                      vehicle.Longitude,
			"latitude":                       vehicle.Latitude,
			"recorded_at_time":               vehicle.RecordedAtTime,
			"valid_until_time":               vehicle.ValidUntilTime,
			"bus_image":                      vehicle.BusImage,
		}

		// Convert vehicle to JSON
		vehicleJSON, err := json.Marshal(vehicleLog)
		if err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to marshal vehicle JSON for dry run: %w", err)
		}

		fmt.Printf("Log Line %d: %s\n", i+1, string(vehicleJSON))
	}

	fmt.Println("=== END DRY RUN ===\n")

	span.SetAttributes(
		attribute.Int("vehicles_printed", len(data.VehicleData)),
	)

	return nil
}

func (p *Pipeline) sendToLoki(ctx context.Context, data *types.ParsedBusData) error {
	ctx, span := p.tracer.Start(ctx, "pipeline.send_to_loki")
	defer span.End()

	if p.lokiClient == nil {
		err := fmt.Errorf("loki client not initialized")
		span.RecordError(err)
		return err
	}

	if err := p.lokiClient.SendBusData(ctx, data); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to send data to Loki: %w", err)
	}

	log.Printf("Successfully sent %d individual vehicle log lines to Loki for line %s",
		len(data.VehicleData), data.LineRef)

	span.SetAttributes(
		attribute.Int("vehicles_sent", len(data.VehicleData)),
	)

	return nil
}
