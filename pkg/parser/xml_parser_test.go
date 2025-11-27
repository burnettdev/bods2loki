package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"bods2loki/pkg/bods"
)

func TestFormatStopName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "double underscore becomes dash with spaces",
			input:    "Lyde_Green__Science_Park",
			expected: "Lyde Green - Science Park",
		},
		{
			name:     "single underscore becomes space",
			input:    "High_Street",
			expected: "High Street",
		},
		{
			name:     "multiple double underscores",
			input:    "A__B__C",
			expected: "A - B - C",
		},
		{
			name:     "mixed underscores",
			input:    "City_Centre__Bus_Station",
			expected: "City Centre - Bus Station",
		},
		{
			name:     "no underscores",
			input:    "Airport",
			expected: "Airport",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStopName(tt.input)
			if got != tt.expected {
				t.Errorf("formatStopName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  float64
		expectErr bool
	}{
		{"positive float", "51.454513", 51.454513, false},
		{"negative float", "-2.587910", -2.587910, false},
		{"integer as float", "45", 45.0, false},
		{"zero", "0", 0.0, false},
		{"with whitespace", "  12.5  ", 12.5, false},
		{"invalid", "not-a-number", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFloat(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("parseFloat(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("parseFloat(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.expected {
					t.Errorf("parseFloat(%q) = %v, want %v", tt.input, got, tt.expected)
				}
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int
		expectErr bool
	}{
		{"positive int", "5", 5, false},
		{"zero", "0", 0, false},
		{"with whitespace", "  10  ", 10, false},
		{"invalid", "not-a-number", 0, true},
		{"empty", "", 0, true},
		{"float string", "3.14", 3, false}, // Sscanf parses up to non-digit
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInt(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("parseInt(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("parseInt(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.expected {
					t.Errorf("parseInt(%q) = %v, want %v", tt.input, got, tt.expected)
				}
			}
		})
	}
}

func TestParseStopCall(t *testing.T) {
	parser := NewXMLParser()

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected *struct {
			stopPointRef        string
			stopPointName       string
			aimedArrivalTime    string
			expectedArrivalTime string
			visitNumber         int
		}
		expectNil bool
	}{
		{
			name: "full stop call",
			input: map[string]interface{}{
				"StopPointRef":        "490000050X",
				"StopPointName":       "High_Street__Library",
				"VisitNumber":         "5",
				"AimedArrivalTime":    "2024-01-15T10:28:00Z",
				"ExpectedArrivalTime": "2024-01-15T10:29:30Z",
			},
			expected: &struct {
				stopPointRef        string
				stopPointName       string
				aimedArrivalTime    string
				expectedArrivalTime string
				visitNumber         int
			}{
				stopPointRef:        "490000050X",
				stopPointName:       "High Street - Library",
				aimedArrivalTime:    "2024-01-15T10:28:00Z",
				expectedArrivalTime: "2024-01-15T10:29:30Z",
				visitNumber:         5,
			},
		},
		{
			name: "minimal stop call",
			input: map[string]interface{}{
				"StopPointRef": "490000050X",
			},
			expected: &struct {
				stopPointRef        string
				stopPointName       string
				aimedArrivalTime    string
				expectedArrivalTime string
				visitNumber         int
			}{
				stopPointRef: "490000050X",
			},
		},
		{
			name:      "empty map returns nil",
			input:     map[string]interface{}{},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.parseStopCall(tt.input)
			if tt.expectNil {
				if got != nil {
					t.Errorf("parseStopCall() expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("parseStopCall() returned nil, expected non-nil")
			}
			if got.StopPointRef != tt.expected.stopPointRef {
				t.Errorf("StopPointRef = %q, want %q", got.StopPointRef, tt.expected.stopPointRef)
			}
			if got.StopPointName != tt.expected.stopPointName {
				t.Errorf("StopPointName = %q, want %q", got.StopPointName, tt.expected.stopPointName)
			}
			if got.VisitNumber != tt.expected.visitNumber {
				t.Errorf("VisitNumber = %d, want %d", got.VisitNumber, tt.expected.visitNumber)
			}
		})
	}
}

func TestParseOnwardCalls(t *testing.T) {
	parser := NewXMLParser()

	tests := []struct {
		name          string
		input         map[string]interface{}
		expectedCount int
	}{
		{
			name: "array of onward calls",
			input: map[string]interface{}{
				"OnwardCall": []interface{}{
					map[string]interface{}{
						"StopPointRef":  "490000051Y",
						"StopPointName": "Market_Square",
						"VisitNumber":   "6",
					},
					map[string]interface{}{
						"StopPointRef":  "490000052Z",
						"StopPointName": "Railway_Station",
						"VisitNumber":   "7",
					},
				},
			},
			expectedCount: 2,
		},
		{
			name: "single onward call as map",
			input: map[string]interface{}{
				"OnwardCall": map[string]interface{}{
					"StopPointRef":  "490000051Y",
					"StopPointName": "Market_Square",
				},
			},
			expectedCount: 1,
		},
		{
			name:          "empty map",
			input:         map[string]interface{}{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.parseOnwardCalls(tt.input)
			if len(got) != tt.expectedCount {
				t.Errorf("parseOnwardCalls() returned %d calls, want %d", len(got), tt.expectedCount)
			}
		})
	}
}

func TestParseBusData_SampleXML(t *testing.T) {
	// Load sample XML from testdata
	xmlPath := filepath.Join("testdata", "sample_siri_vm.xml")
	xmlData, err := os.ReadFile(xmlPath)
	if err != nil {
		t.Fatalf("Failed to read sample XML: %v", err)
	}

	parser := NewXMLParser()
	busData := &bods.BusData{
		XMLData: string(xmlData),
		LineRef: "1",
	}

	result, err := parser.ParseBusData(context.Background(), busData)
	if err != nil {
		t.Fatalf("ParseBusData() error: %v", err)
	}

	// Should have 3 vehicles
	if len(result.VehicleData) != 3 {
		t.Errorf("Expected 3 vehicles, got %d", len(result.VehicleData))
	}

	// Verify first vehicle (has all fields including new ones)
	v1 := result.VehicleData[0]

	// Standard fields
	if v1.LineRef != "1" {
		t.Errorf("Vehicle 1 LineRef = %q, want %q", v1.LineRef, "1")
	}
	if v1.DirectionRef != "inbound" {
		t.Errorf("Vehicle 1 DirectionRef = %q, want %q", v1.DirectionRef, "inbound")
	}
	if v1.OperatorRef != "FBUS" {
		t.Errorf("Vehicle 1 OperatorRef = %q, want %q", v1.OperatorRef, "FBUS")
	}
	if v1.VehicleRef != "FBUS-12345" {
		t.Errorf("Vehicle 1 VehicleRef = %q, want %q", v1.VehicleRef, "FBUS-12345")
	}

	// Location
	if v1.Longitude != -2.587910 {
		t.Errorf("Vehicle 1 Longitude = %v, want %v", v1.Longitude, -2.587910)
	}
	if v1.Latitude != 51.454513 {
		t.Errorf("Vehicle 1 Latitude = %v, want %v", v1.Latitude, 51.454513)
	}

	// Stop name formatting
	if v1.OriginName != "City Centre - Bus Station" {
		t.Errorf("Vehicle 1 OriginName = %q, want %q", v1.OriginName, "City Centre - Bus Station")
	}
	if v1.DestinationName != "Lyde Green - Science Park" {
		t.Errorf("Vehicle 1 DestinationName = %q, want %q", v1.DestinationName, "Lyde Green - Science Park")
	}

	// NEW FIELDS - Bearing
	if v1.Bearing != 45.5 {
		t.Errorf("Vehicle 1 Bearing = %v, want %v", v1.Bearing, 45.5)
	}

	// NEW FIELDS - Velocity
	if v1.Velocity != 12.5 {
		t.Errorf("Vehicle 1 Velocity = %v, want %v", v1.Velocity, 12.5)
	}

	// NEW FIELDS - Occupancy
	if v1.Occupancy != "seatsAvailable" {
		t.Errorf("Vehicle 1 Occupancy = %q, want %q", v1.Occupancy, "seatsAvailable")
	}

	// NEW FIELDS - ProgressStatus
	if v1.ProgressStatus != "normalProgress" {
		t.Errorf("Vehicle 1 ProgressStatus = %q, want %q", v1.ProgressStatus, "normalProgress")
	}

	// NEW FIELDS - PublishedLineName
	if v1.PublishedLineName != "Route 1" {
		t.Errorf("Vehicle 1 PublishedLineName = %q, want %q", v1.PublishedLineName, "Route 1")
	}

	// NEW FIELDS - BlockRef
	if v1.BlockRef != "BLK001" {
		t.Errorf("Vehicle 1 BlockRef = %q, want %q", v1.BlockRef, "BLK001")
	}

	// NEW FIELDS - MonitoredCall
	if v1.MonitoredCall == nil {
		t.Error("Vehicle 1 MonitoredCall is nil, expected non-nil")
	} else {
		if v1.MonitoredCall.StopPointRef != "490000050X" {
			t.Errorf("Vehicle 1 MonitoredCall.StopPointRef = %q, want %q", v1.MonitoredCall.StopPointRef, "490000050X")
		}
		if v1.MonitoredCall.StopPointName != "High Street - Library" {
			t.Errorf("Vehicle 1 MonitoredCall.StopPointName = %q, want %q", v1.MonitoredCall.StopPointName, "High Street - Library")
		}
		if v1.MonitoredCall.VisitNumber != 5 {
			t.Errorf("Vehicle 1 MonitoredCall.VisitNumber = %d, want %d", v1.MonitoredCall.VisitNumber, 5)
		}
		if v1.MonitoredCall.ExpectedArrivalTime != "2024-01-15T10:29:30Z" {
			t.Errorf("Vehicle 1 MonitoredCall.ExpectedArrivalTime = %q, want %q", v1.MonitoredCall.ExpectedArrivalTime, "2024-01-15T10:29:30Z")
		}
	}

	// NEW FIELDS - OnwardCalls
	if len(v1.OnwardCalls) != 2 {
		t.Errorf("Vehicle 1 OnwardCalls count = %d, want %d", len(v1.OnwardCalls), 2)
	} else {
		if v1.OnwardCalls[0].StopPointRef != "490000051Y" {
			t.Errorf("Vehicle 1 OnwardCalls[0].StopPointRef = %q, want %q", v1.OnwardCalls[0].StopPointRef, "490000051Y")
		}
		if v1.OnwardCalls[0].StopPointName != "Market Square" {
			t.Errorf("Vehicle 1 OnwardCalls[0].StopPointName = %q, want %q", v1.OnwardCalls[0].StopPointName, "Market Square")
		}
	}

	// Verify second vehicle (outbound, different occupancy)
	v2 := result.VehicleData[1]
	if v2.DirectionRef != "outbound" {
		t.Errorf("Vehicle 2 DirectionRef = %q, want %q", v2.DirectionRef, "outbound")
	}
	if v2.Occupancy != "standingAvailable" {
		t.Errorf("Vehicle 2 Occupancy = %q, want %q", v2.Occupancy, "standingAvailable")
	}
	if v2.Bearing != 225.0 {
		t.Errorf("Vehicle 2 Bearing = %v, want %v", v2.Bearing, 225.0)
	}

	// Verify third vehicle (uses DatedVehicleJourneyRef as fallback, missing some fields)
	v3 := result.VehicleData[2]
	if v3.VehicleRef != "VJ003-FALLBACK" {
		t.Errorf("Vehicle 3 VehicleRef = %q, want %q (should fallback to DatedVehicleJourneyRef)", v3.VehicleRef, "VJ003-FALLBACK")
	}
	if v3.ProgressStatus != "layover" {
		t.Errorf("Vehicle 3 ProgressStatus = %q, want %q", v3.ProgressStatus, "layover")
	}
	// Should have zero values for missing fields
	if v3.Bearing != 0 {
		t.Errorf("Vehicle 3 Bearing = %v, want %v (should be 0 when missing)", v3.Bearing, 0)
	}
	if v3.Velocity != 0 {
		t.Errorf("Vehicle 3 Velocity = %v, want %v (should be 0 when missing)", v3.Velocity, 0)
	}
	if v3.Occupancy != "" {
		t.Errorf("Vehicle 3 Occupancy = %q, want empty string (should be empty when missing)", v3.Occupancy)
	}
}

func TestParseBusData_SingleVehicleActivity(t *testing.T) {
	// Test with single VehicleActivity (not array)
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Siri xmlns="http://www.siri.org.uk/siri" version="2.0">
  <ServiceDelivery>
    <VehicleMonitoringDelivery version="2.0">
      <VehicleActivity>
        <RecordedAtTime>2024-01-15T10:29:45Z</RecordedAtTime>
        <MonitoredVehicleJourney>
          <LineRef>1</LineRef>
          <DirectionRef>inbound</DirectionRef>
          <OperatorRef>FBUS</OperatorRef>
          <VehicleRef>FBUS-12345</VehicleRef>
          <VehicleLocation>
            <Longitude>-2.587910</Longitude>
            <Latitude>51.454513</Latitude>
          </VehicleLocation>
        </MonitoredVehicleJourney>
      </VehicleActivity>
    </VehicleMonitoringDelivery>
  </ServiceDelivery>
</Siri>`

	parser := NewXMLParser()
	busData := &bods.BusData{
		XMLData: xmlData,
		LineRef: "1",
	}

	result, err := parser.ParseBusData(context.Background(), busData)
	if err != nil {
		t.Fatalf("ParseBusData() error: %v", err)
	}

	if len(result.VehicleData) != 1 {
		t.Errorf("Expected 1 vehicle, got %d", len(result.VehicleData))
	}
}

func TestParseBusData_EmptyResponse(t *testing.T) {
	// Test with no VehicleActivity elements
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Siri xmlns="http://www.siri.org.uk/siri" version="2.0">
  <ServiceDelivery>
    <VehicleMonitoringDelivery version="2.0">
    </VehicleMonitoringDelivery>
  </ServiceDelivery>
</Siri>`

	parser := NewXMLParser()
	busData := &bods.BusData{
		XMLData: xmlData,
		LineRef: "1",
	}

	result, err := parser.ParseBusData(context.Background(), busData)
	if err != nil {
		t.Fatalf("ParseBusData() error: %v", err)
	}

	if len(result.VehicleData) != 0 {
		t.Errorf("Expected 0 vehicles for empty response, got %d", len(result.VehicleData))
	}
}

func TestParseBusData_MalformedXML(t *testing.T) {
	xmlData := `this is not valid XML`

	parser := NewXMLParser()
	busData := &bods.BusData{
		XMLData: xmlData,
		LineRef: "1",
	}

	_, err := parser.ParseBusData(context.Background(), busData)
	if err == nil {
		t.Error("Expected error for malformed XML, got nil")
	}
}

func TestParseVehicleActivity_MissingFields(t *testing.T) {
	parser := NewXMLParser()

	// Minimal activity with just RecordedAtTime
	activity := map[string]interface{}{
		"RecordedAtTime": "2024-01-15T10:29:45Z",
	}

	result := parser.parseVehicleActivity(activity)
	if result == nil {
		t.Fatal("parseVehicleActivity() returned nil for minimal activity")
	}
	if result.RecordedAtTime != "2024-01-15T10:29:45Z" {
		t.Errorf("RecordedAtTime = %q, want %q", result.RecordedAtTime, "2024-01-15T10:29:45Z")
	}
	// All other fields should be zero values
	if result.LineRef != "" {
		t.Errorf("LineRef = %q, want empty", result.LineRef)
	}
	if result.Longitude != 0 {
		t.Errorf("Longitude = %v, want 0", result.Longitude)
	}
}

func TestBusImageGeneration(t *testing.T) {
	parser := NewXMLParser()

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Siri xmlns="http://www.siri.org.uk/siri" version="2.0">
  <ServiceDelivery>
    <VehicleMonitoringDelivery version="2.0">
      <VehicleActivity>
        <MonitoredVehicleJourney>
          <LineRef>49x</LineRef>
          <DirectionRef>inbound</DirectionRef>
          <VehicleLocation>
            <Longitude>-2.5</Longitude>
            <Latitude>51.4</Latitude>
          </VehicleLocation>
        </MonitoredVehicleJourney>
      </VehicleActivity>
    </VehicleMonitoringDelivery>
  </ServiceDelivery>
</Siri>`

	busData := &bods.BusData{
		XMLData: xmlData,
		LineRef: "49x",
	}

	result, err := parser.ParseBusData(context.Background(), busData)
	if err != nil {
		t.Fatalf("ParseBusData() error: %v", err)
	}

	if len(result.VehicleData) != 1 {
		t.Fatalf("Expected 1 vehicle, got %d", len(result.VehicleData))
	}

	// Verify bus image is generated
	if result.VehicleData[0].BusImage == "" {
		t.Error("BusImage is empty, expected base64-encoded SVG")
	}
	if len(result.VehicleData[0].BusImage) < 100 {
		t.Error("BusImage seems too short for a valid base64 SVG")
	}
}
