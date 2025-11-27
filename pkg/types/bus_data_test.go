package types

import (
	"encoding/json"
	"testing"
)

func TestVehicleActivityJSON_Marshal(t *testing.T) {
	vehicle := VehicleActivity{
		VehicleRef:                  "FBUS-12345",
		LineRef:                     "1",
		DirectionRef:                "inbound",
		OperatorRef:                 "FBUS",
		OriginRef:                   "490000001A",
		OriginName:                  "City Centre - Bus Station",
		DestinationRef:              "490000099Z",
		DestinationName:             "Lyde Green - Science Park",
		OriginAimedDepartureTime:    "2024-01-15T10:00:00Z",
		DestinationAimedArrivalTime: "2024-01-15T11:00:00Z",
		Longitude:                   -2.587910,
		Latitude:                    51.454513,
		RecordedAtTime:              "2024-01-15T10:29:45Z",
		ValidUntilTime:              "2024-01-15T10:34:45Z",
		BusImage:                    "data:image/svg+xml;base64,PHN2Zz4=",
		// New fields
		Bearing:           45.5,
		Velocity:          12.5,
		Occupancy:         "seatsAvailable",
		ProgressStatus:    "normalProgress",
		PublishedLineName: "Route 1",
		BlockRef:          "BLK001",
		MonitoredCall: &StopCall{
			StopPointRef:        "490000050X",
			StopPointName:       "High Street - Library",
			VisitNumber:         5,
			AimedArrivalTime:    "2024-01-15T10:28:00Z",
			ExpectedArrivalTime: "2024-01-15T10:29:30Z",
		},
		OnwardCalls: []StopCall{
			{
				StopPointRef:        "490000051Y",
				StopPointName:       "Market Square",
				VisitNumber:         6,
				AimedArrivalTime:    "2024-01-15T10:35:00Z",
				ExpectedArrivalTime: "2024-01-15T10:36:30Z",
			},
		},
	}

	data, err := json.Marshal(vehicle)
	if err != nil {
		t.Fatalf("Failed to marshal VehicleActivity: %v", err)
	}

	// Verify JSON contains expected fields
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check standard fields
	if result["vehicle_ref"] != "FBUS-12345" {
		t.Errorf("vehicle_ref = %v, want %v", result["vehicle_ref"], "FBUS-12345")
	}
	if result["line_ref"] != "1" {
		t.Errorf("line_ref = %v, want %v", result["line_ref"], "1")
	}

	// Check new fields
	if result["bearing"] != 45.5 {
		t.Errorf("bearing = %v, want %v", result["bearing"], 45.5)
	}
	if result["velocity"] != 12.5 {
		t.Errorf("velocity = %v, want %v", result["velocity"], 12.5)
	}
	if result["occupancy"] != "seatsAvailable" {
		t.Errorf("occupancy = %v, want %v", result["occupancy"], "seatsAvailable")
	}
	if result["progress_status"] != "normalProgress" {
		t.Errorf("progress_status = %v, want %v", result["progress_status"], "normalProgress")
	}
	if result["published_line_name"] != "Route 1" {
		t.Errorf("published_line_name = %v, want %v", result["published_line_name"], "Route 1")
	}
	if result["block_ref"] != "BLK001" {
		t.Errorf("block_ref = %v, want %v", result["block_ref"], "BLK001")
	}

	// Check nested MonitoredCall
	mc, ok := result["monitored_call"].(map[string]interface{})
	if !ok {
		t.Error("monitored_call is not a map")
	} else {
		if mc["stop_point_ref"] != "490000050X" {
			t.Errorf("monitored_call.stop_point_ref = %v, want %v", mc["stop_point_ref"], "490000050X")
		}
	}

	// Check OnwardCalls array
	oc, ok := result["onward_calls"].([]interface{})
	if !ok {
		t.Error("onward_calls is not an array")
	} else if len(oc) != 1 {
		t.Errorf("onward_calls length = %d, want 1", len(oc))
	}
}

func TestVehicleActivityJSON_Unmarshal(t *testing.T) {
	jsonData := `{
		"vehicle_ref": "FBUS-12345",
		"line_ref": "1",
		"direction_ref": "inbound",
		"operator_ref": "FBUS",
		"longitude": -2.58791,
		"latitude": 51.454513,
		"bearing": 45.5,
		"velocity": 12.5,
		"occupancy": "seatsAvailable",
		"progress_status": "normalProgress",
		"published_line_name": "Route 1",
		"block_ref": "BLK001",
		"monitored_call": {
			"stop_point_ref": "490000050X",
			"stop_point_name": "High Street - Library",
			"visit_number": 5
		},
		"onward_calls": [
			{
				"stop_point_ref": "490000051Y",
				"stop_point_name": "Market Square"
			}
		]
	}`

	var vehicle VehicleActivity
	if err := json.Unmarshal([]byte(jsonData), &vehicle); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check standard fields
	if vehicle.VehicleRef != "FBUS-12345" {
		t.Errorf("VehicleRef = %q, want %q", vehicle.VehicleRef, "FBUS-12345")
	}

	// Check new fields
	if vehicle.Bearing != 45.5 {
		t.Errorf("Bearing = %v, want %v", vehicle.Bearing, 45.5)
	}
	if vehicle.Velocity != 12.5 {
		t.Errorf("Velocity = %v, want %v", vehicle.Velocity, 12.5)
	}
	if vehicle.Occupancy != "seatsAvailable" {
		t.Errorf("Occupancy = %q, want %q", vehicle.Occupancy, "seatsAvailable")
	}
	if vehicle.ProgressStatus != "normalProgress" {
		t.Errorf("ProgressStatus = %q, want %q", vehicle.ProgressStatus, "normalProgress")
	}
	if vehicle.PublishedLineName != "Route 1" {
		t.Errorf("PublishedLineName = %q, want %q", vehicle.PublishedLineName, "Route 1")
	}
	if vehicle.BlockRef != "BLK001" {
		t.Errorf("BlockRef = %q, want %q", vehicle.BlockRef, "BLK001")
	}

	// Check MonitoredCall
	if vehicle.MonitoredCall == nil {
		t.Error("MonitoredCall is nil")
	} else {
		if vehicle.MonitoredCall.StopPointRef != "490000050X" {
			t.Errorf("MonitoredCall.StopPointRef = %q, want %q", vehicle.MonitoredCall.StopPointRef, "490000050X")
		}
		if vehicle.MonitoredCall.VisitNumber != 5 {
			t.Errorf("MonitoredCall.VisitNumber = %d, want %d", vehicle.MonitoredCall.VisitNumber, 5)
		}
	}

	// Check OnwardCalls
	if len(vehicle.OnwardCalls) != 1 {
		t.Errorf("OnwardCalls length = %d, want 1", len(vehicle.OnwardCalls))
	}
}

func TestVehicleActivityJSON_OmitEmpty(t *testing.T) {
	// Vehicle with minimal fields - optional fields should be omitted
	vehicle := VehicleActivity{
		VehicleRef: "FBUS-12345",
		LineRef:    "1",
		Longitude:  -2.5,
		Latitude:   51.4,
	}

	data, err := json.Marshal(vehicle)
	if err != nil {
		t.Fatalf("Failed to marshal VehicleActivity: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// These optional fields should NOT be present when zero/empty
	optionalFields := []string{
		"bearing", "velocity", "occupancy", "progress_status",
		"published_line_name", "block_ref", "monitored_call", "onward_calls",
	}

	for _, field := range optionalFields {
		if _, exists := result[field]; exists {
			// Note: bearing and velocity are float64, so they might appear as 0
			// Check if it's actually zero or just present
			if field == "bearing" || field == "velocity" {
				if result[field] != float64(0) {
					t.Errorf("Field %q should be omitted when empty, got %v", field, result[field])
				}
			}
		}
	}

	// Required fields should always be present
	requiredFields := []string{"vehicle_ref", "line_ref", "longitude", "latitude"}
	for _, field := range requiredFields {
		if _, exists := result[field]; !exists {
			t.Errorf("Required field %q is missing", field)
		}
	}
}

func TestStopCallJSON(t *testing.T) {
	stopCall := StopCall{
		StopPointRef:          "490000050X",
		StopPointName:         "High Street - Library",
		VisitNumber:           5,
		AimedArrivalTime:      "2024-01-15T10:28:00Z",
		ExpectedArrivalTime:   "2024-01-15T10:29:30Z",
		AimedDepartureTime:    "2024-01-15T10:29:00Z",
		ExpectedDepartureTime: "2024-01-15T10:30:30Z",
	}

	data, err := json.Marshal(stopCall)
	if err != nil {
		t.Fatalf("Failed to marshal StopCall: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	expectedFields := map[string]interface{}{
		"stop_point_ref":          "490000050X",
		"stop_point_name":         "High Street - Library",
		"visit_number":            float64(5), // JSON numbers are float64
		"aimed_arrival_time":      "2024-01-15T10:28:00Z",
		"expected_arrival_time":   "2024-01-15T10:29:30Z",
		"aimed_departure_time":    "2024-01-15T10:29:00Z",
		"expected_departure_time": "2024-01-15T10:30:30Z",
	}

	for field, expected := range expectedFields {
		if result[field] != expected {
			t.Errorf("StopCall.%s = %v, want %v", field, result[field], expected)
		}
	}
}

func TestParsedBusDataJSON(t *testing.T) {
	data := ParsedBusData{
		LineRef:   "1",
		Timestamp: "2024-01-15T10:30:00.000Z",
		VehicleData: []VehicleActivity{
			{
				VehicleRef: "FBUS-12345",
				LineRef:    "1",
				Longitude:  -2.5,
				Latitude:   51.4,
			},
		},
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal ParsedBusData: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result["line_ref"] != "1" {
		t.Errorf("line_ref = %v, want %v", result["line_ref"], "1")
	}
	if result["timestamp"] != "2024-01-15T10:30:00.000Z" {
		t.Errorf("timestamp = %v, want %v", result["timestamp"], "2024-01-15T10:30:00.000Z")
	}

	vehicles, ok := result["vehicle_activities"].([]interface{})
	if !ok {
		t.Error("vehicle_activities is not an array")
	} else if len(vehicles) != 1 {
		t.Errorf("vehicle_activities length = %d, want 1", len(vehicles))
	}
}
