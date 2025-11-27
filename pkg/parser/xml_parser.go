package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"bods2loki/pkg/bods"
	"bods2loki/pkg/types"

	"github.com/clbanning/mxj/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type XMLParser struct {
	tracer         trace.Tracer
	imageGenerator *BusImageGenerator
}

func NewXMLParser() *XMLParser {
	return &XMLParser{
		tracer:         otel.Tracer("xml-parser"),
		imageGenerator: NewBusImageGenerator(),
	}
}

func (p *XMLParser) ParseBusData(ctx context.Context, busData *bods.BusData) (*types.ParsedBusData, error) {
	ctx, span := p.tracer.Start(ctx, "xml_parser.parse_bus_data",
		trace.WithAttributes(
			attribute.String("line_ref", busData.LineRef),
			attribute.Int("xml_size_bytes", len(busData.XMLData)),
		),
	)
	defer span.End()

	// Parse XML to map
	xmlMap, err := mxj.NewMapXml([]byte(busData.XMLData))
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	// Extract vehicle activities
	vehicles, err := p.extractVehicleActivities(ctx, xmlMap)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to extract vehicle activities: %w", err)
	}

	span.SetAttributes(
		attribute.Int("vehicles_count", len(vehicles)),
	)

	return &types.ParsedBusData{
		LineRef:     busData.LineRef,
		Timestamp:   busData.Timestamp.Format("2006-01-02T15:04:05.000Z"),
		VehicleData: vehicles,
		RawData:     xmlMap,
	}, nil
}

func (p *XMLParser) extractVehicleActivities(ctx context.Context, xmlMap map[string]interface{}) ([]types.VehicleActivity, error) {
	_, span := p.tracer.Start(ctx, "xml_parser.extract_vehicle_activities")
	defer span.End()

	var vehicles []types.VehicleActivity

	// Navigate through the XML structure to find VehicleActivity elements
	// The structure appears to be: Siri -> ServiceDelivery -> VehicleMonitoringDelivery -> VehicleActivity
	siri, ok := xmlMap["Siri"].(map[string]interface{})
	if !ok {
		return vehicles, nil
	}

	serviceDelivery, ok := siri["ServiceDelivery"].(map[string]interface{})
	if !ok {
		return vehicles, nil
	}

	vmDelivery, ok := serviceDelivery["VehicleMonitoringDelivery"].(map[string]interface{})
	if !ok {
		return vehicles, nil
	}

	// VehicleActivity can be a single item or an array
	var vehicleActivities []interface{}
	switch va := vmDelivery["VehicleActivity"].(type) {
	case []interface{}:
		vehicleActivities = va
	case map[string]interface{}:
		vehicleActivities = []interface{}{va}
	default:
		return vehicles, nil
	}

	for _, activity := range vehicleActivities {
		activityMap, ok := activity.(map[string]interface{})
		if !ok {
			continue
		}

		vehicle := p.parseVehicleActivity(activityMap)
		if vehicle != nil {
			vehicles = append(vehicles, *vehicle)
		}
	}

	span.SetAttributes(
		attribute.Int("extracted_vehicles", len(vehicles)),
	)

	return vehicles, nil
}

func (p *XMLParser) parseVehicleActivity(activity map[string]interface{}) *types.VehicleActivity {
	vehicle := &types.VehicleActivity{}

	// Extract RecordedAtTime and ValidUntilTime from top level
	if rat, ok := activity["RecordedAtTime"].(string); ok {
		vehicle.RecordedAtTime = rat
	}
	if vut, ok := activity["ValidUntilTime"].(string); ok {
		vehicle.ValidUntilTime = vut
	}

	// Extract MonitoredVehicleJourney data
	mvj, ok := activity["MonitoredVehicleJourney"].(map[string]interface{})
	if !ok {
		return vehicle
	}

	// Extract basic fields
	if lineRef, ok := mvj["LineRef"].(string); ok {
		vehicle.LineRef = lineRef
	}
	if dirRef, ok := mvj["DirectionRef"].(string); ok {
		vehicle.DirectionRef = dirRef
	}
	if opRef, ok := mvj["OperatorRef"].(string); ok {
		vehicle.OperatorRef = opRef
	}

	// Extract VehicleRef
	if vRef, ok := mvj["VehicleRef"].(string); ok {
		vehicle.VehicleRef = vRef
	}

	// Extract FramedVehicleJourneyRef data
	if fvjr, ok := mvj["FramedVehicleJourneyRef"].(map[string]interface{}); ok {
		if datedVJRef, ok := fvjr["DatedVehicleJourneyRef"].(string); ok {
			// Use this as additional vehicle identifier if VehicleRef is empty
			if vehicle.VehicleRef == "" {
				vehicle.VehicleRef = datedVJRef
			}
		}
	}

	// Extract origin and destination
	if originRef, ok := mvj["OriginRef"].(string); ok {
		vehicle.OriginRef = originRef
	}
	if originName, ok := mvj["OriginName"].(string); ok {
		vehicle.OriginName = formatStopName(originName)
	}
	if destRef, ok := mvj["DestinationRef"].(string); ok {
		vehicle.DestinationRef = destRef
	}
	if destName, ok := mvj["DestinationName"].(string); ok {
		vehicle.DestinationName = formatStopName(destName)
	}
	if originAimed, ok := mvj["OriginAimedDepartureTime"].(string); ok {
		vehicle.OriginAimedDepartureTime = originAimed
	}
	if destAimed, ok := mvj["DestinationAimedArrivalTime"].(string); ok {
		vehicle.DestinationAimedArrivalTime = destAimed
	}

	// Extract location data (including Bearing and Velocity)
	if location, ok := mvj["VehicleLocation"].(map[string]interface{}); ok {
		if lng, ok := location["Longitude"].(string); ok {
			if f, err := parseFloat(lng); err == nil {
				vehicle.Longitude = f
			}
		}
		if lat, ok := location["Latitude"].(string); ok {
			if f, err := parseFloat(lat); err == nil {
				vehicle.Latitude = f
			}
		}
		// Extract bearing (vehicle heading direction in degrees)
		if bearing, ok := location["Bearing"].(string); ok {
			if f, err := parseFloat(bearing); err == nil {
				vehicle.Bearing = f
			}
		}
	}

	// Extract velocity (speed in m/s) - can be at VehicleLocation level or MVJ level
	if velocity, ok := mvj["Velocity"].(string); ok {
		if f, err := parseFloat(velocity); err == nil {
			vehicle.Velocity = f
		}
	}

	// Extract occupancy status (full|seatsAvailable|standingAvailable)
	if occupancy, ok := mvj["Occupancy"].(string); ok {
		vehicle.Occupancy = occupancy
	}

	// Extract progress status
	if progressStatus, ok := mvj["ProgressStatus"].(string); ok {
		vehicle.ProgressStatus = progressStatus
	}

	// Extract published line name (customer-facing route name)
	if publishedLineName, ok := mvj["PublishedLineName"].(string); ok {
		vehicle.PublishedLineName = publishedLineName
	}

	// Extract block reference (operational block identifier)
	if blockRef, ok := mvj["BlockRef"].(string); ok {
		vehicle.BlockRef = blockRef
	}

	// Extract MonitoredCall (current/next stop with ETA)
	if mc, ok := mvj["MonitoredCall"].(map[string]interface{}); ok {
		vehicle.MonitoredCall = p.parseStopCall(mc)
	}

	// Extract OnwardCalls (future stops with predictions)
	if oc, ok := mvj["OnwardCalls"].(map[string]interface{}); ok {
		vehicle.OnwardCalls = p.parseOnwardCalls(oc)
	}

	// Generate bus image with line number and direction
	vehicle.BusImage = p.imageGenerator.GenerateCompactBusImage(vehicle.LineRef, vehicle.DirectionRef)

	return vehicle
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// formatStopName cleans up stop names from BODS format
// Rules:
// - Double underscores (__) become " - "
// - Single underscores (_) become spaces
// Example: "Lyde_Green__Science_Park" becomes "Lyde Green - Science Park"
func formatStopName(name string) string {
	if name == "" {
		return ""
	}

	// First replace double underscores with " - "
	formatted := strings.ReplaceAll(name, "__", " - ")

	// Then replace remaining single underscores with spaces
	formatted = strings.ReplaceAll(formatted, "_", " ")

	return formatted
}

// parseStopCall extracts stop call data from a MonitoredCall or OnwardCall element
func (p *XMLParser) parseStopCall(callData map[string]interface{}) *types.StopCall {
	call := &types.StopCall{}

	if stopRef, ok := callData["StopPointRef"].(string); ok {
		call.StopPointRef = stopRef
	}
	if stopName, ok := callData["StopPointName"].(string); ok {
		call.StopPointName = formatStopName(stopName)
	}
	if aimed, ok := callData["AimedArrivalTime"].(string); ok {
		call.AimedArrivalTime = aimed
	}
	if expected, ok := callData["ExpectedArrivalTime"].(string); ok {
		call.ExpectedArrivalTime = expected
	}
	if aimed, ok := callData["AimedDepartureTime"].(string); ok {
		call.AimedDepartureTime = aimed
	}
	if expected, ok := callData["ExpectedDepartureTime"].(string); ok {
		call.ExpectedDepartureTime = expected
	}
	if visitNum, ok := callData["VisitNumber"].(string); ok {
		if n, err := parseInt(visitNum); err == nil {
			call.VisitNumber = n
		}
	}

	// Return nil if no meaningful data was extracted
	if call.StopPointRef == "" && call.StopPointName == "" {
		return nil
	}

	return call
}

// parseOnwardCalls extracts an array of future stop calls
func (p *XMLParser) parseOnwardCalls(onwardCallsData map[string]interface{}) []types.StopCall {
	var calls []types.StopCall

	// OnwardCall can be a single item or an array
	switch oc := onwardCallsData["OnwardCall"].(type) {
	case []interface{}:
		for _, item := range oc {
			if callMap, ok := item.(map[string]interface{}); ok {
				if call := p.parseStopCall(callMap); call != nil {
					calls = append(calls, *call)
				}
			}
		}
	case map[string]interface{}:
		if call := p.parseStopCall(oc); call != nil {
			calls = append(calls, *call)
		}
	}

	return calls
}

// parseInt parses a string to int
func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// ToJSON converts ParsedBusData to formatted JSON
func ToJSON(data *types.ParsedBusData) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}
