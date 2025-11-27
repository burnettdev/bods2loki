package loki

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bods2loki/pkg/types"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:3100", "user", "pass")

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.baseURL != "http://localhost:3100" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "http://localhost:3100")
	}
	if client.username != "user" {
		t.Errorf("username = %q, want %q", client.username, "user")
	}
	if client.password != "pass" {
		t.Errorf("password = %q, want %q", client.password, "pass")
	}
}

func TestSendBusData_MockServer(t *testing.T) {
	var receivedBody []byte
	var receivedHeaders http.Header
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedHeaders = r.Header
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "")

	testData := &types.ParsedBusData{
		LineRef:   "1",
		Timestamp: "2024-01-15T10:30:00.000Z",
		VehicleData: []types.VehicleActivity{
			{
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
			},
		},
	}

	err := client.SendBusData(context.Background(), testData)
	if err != nil {
		t.Fatalf("SendBusData failed: %v", err)
	}

	// Verify correct endpoint called
	if receivedPath != "/loki/api/v1/push" {
		t.Errorf("Expected path /loki/api/v1/push, got %s", receivedPath)
	}

	// Verify Content-Type header
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", receivedHeaders.Get("Content-Type"))
	}

	// Verify User-Agent header
	if receivedHeaders.Get("User-Agent") != "bods2loki/1.0.0" {
		t.Errorf("Expected User-Agent bods2loki/1.0.0, got %s", receivedHeaders.Get("User-Agent"))
	}

	// Parse and validate request body structure
	var pushReq PushRequest
	if err := json.Unmarshal(receivedBody, &pushReq); err != nil {
		t.Fatalf("Failed to parse request body: %v", err)
	}

	// Verify stream structure
	if len(pushReq.Streams) != 1 {
		t.Fatalf("Expected 1 stream, got %d", len(pushReq.Streams))
	}

	stream := pushReq.Streams[0]

	// Verify stream labels
	expectedLabels := map[string]string{
		"job":      "bods2loki",
		"service":  "bus-tracking",
		"line_ref": "1",
	}

	for key, expected := range expectedLabels {
		if stream.Stream[key] != expected {
			t.Errorf("Stream label %q = %q, want %q", key, stream.Stream[key], expected)
		}
	}

	// Verify log entries
	if len(stream.Values) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(stream.Values))
	}

	entry := stream.Values[0]
	if len(entry) != 2 {
		t.Fatalf("Expected entry with [timestamp, content], got %d elements", len(entry))
	}

	// First element should be timestamp (nanoseconds)
	timestamp := entry[0]
	if len(timestamp) < 10 {
		t.Error("Timestamp seems too short for nanoseconds")
	}

	// Second element should be JSON log content
	logContent := entry[1]
	var vehicleLog map[string]interface{}
	if err := json.Unmarshal([]byte(logContent), &vehicleLog); err != nil {
		t.Fatalf("Failed to parse log content JSON: %v", err)
	}

	// Verify vehicle data fields are present
	expectedFields := []string{
		"timestamp", "line_ref", "vehicle_ref", "direction_ref",
		"operator_ref", "origin_ref", "origin_name", "destination_ref",
		"destination_name", "longitude", "latitude", "recorded_at_time",
		"valid_until_time", "bus_image",
	}

	for _, field := range expectedFields {
		if _, exists := vehicleLog[field]; !exists {
			t.Errorf("Expected field %q in log content, not found", field)
		}
	}

	// Verify specific values
	if vehicleLog["vehicle_ref"] != "FBUS-12345" {
		t.Errorf("vehicle_ref = %v, want FBUS-12345", vehicleLog["vehicle_ref"])
	}
	if vehicleLog["line_ref"] != "1" {
		t.Errorf("line_ref = %v, want 1", vehicleLog["line_ref"])
	}
}

func TestSendBusData_WithAuthentication(t *testing.T) {
	var authHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "testuser", "testpass")

	testData := &types.ParsedBusData{
		LineRef:   "1",
		Timestamp: "2024-01-15T10:30:00.000Z",
		VehicleData: []types.VehicleActivity{
			{VehicleRef: "TEST-001", Longitude: -2.5, Latitude: 51.4},
		},
	}

	err := client.SendBusData(context.Background(), testData)
	if err != nil {
		t.Fatalf("SendBusData failed: %v", err)
	}

	// Verify Basic Auth header is present
	if authHeader == "" {
		t.Error("Expected Authorization header to be set")
	}
	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Errorf("Expected Basic auth, got %q", authHeader)
	}
}

func TestSendBusData_NoAuthenticationWhenEmpty(t *testing.T) {
	var authHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "")

	testData := &types.ParsedBusData{
		LineRef:   "1",
		Timestamp: "2024-01-15T10:30:00.000Z",
		VehicleData: []types.VehicleActivity{
			{VehicleRef: "TEST-001", Longitude: -2.5, Latitude: 51.4},
		},
	}

	err := client.SendBusData(context.Background(), testData)
	if err != nil {
		t.Fatalf("SendBusData failed: %v", err)
	}

	// Verify no Authorization header when credentials are empty
	if authHeader != "" {
		t.Errorf("Expected no Authorization header, got %q", authHeader)
	}
}

func TestSendBusData_MultipleVehicles(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "")

	testData := &types.ParsedBusData{
		LineRef:   "1",
		Timestamp: "2024-01-15T10:30:00.000Z",
		VehicleData: []types.VehicleActivity{
			{VehicleRef: "BUS-001", Longitude: -2.5, Latitude: 51.4},
			{VehicleRef: "BUS-002", Longitude: -2.6, Latitude: 51.5},
			{VehicleRef: "BUS-003", Longitude: -2.7, Latitude: 51.6},
		},
	}

	err := client.SendBusData(context.Background(), testData)
	if err != nil {
		t.Fatalf("SendBusData failed: %v", err)
	}

	var pushReq PushRequest
	if err := json.Unmarshal(receivedBody, &pushReq); err != nil {
		t.Fatalf("Failed to parse request body: %v", err)
	}

	// Should have 3 log entries (one per vehicle)
	if len(pushReq.Streams[0].Values) != 3 {
		t.Errorf("Expected 3 log entries, got %d", len(pushReq.Streams[0].Values))
	}
}

func TestSendBusData_ErrorOnNon2xx(t *testing.T) {
	tests := []struct {
		statusCode int
		expectErr  bool
	}{
		{http.StatusOK, false},
		{http.StatusNoContent, false},
		{http.StatusCreated, false},
		{http.StatusBadRequest, true},
		{http.StatusUnauthorized, true},
		{http.StatusInternalServerError, true},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.URL, "", "")

			testData := &types.ParsedBusData{
				LineRef:   "1",
				Timestamp: "2024-01-15T10:30:00.000Z",
				VehicleData: []types.VehicleActivity{
					{VehicleRef: "TEST-001", Longitude: -2.5, Latitude: 51.4},
				},
			}

			err := client.SendBusData(context.Background(), testData)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error for status %d, got nil", tt.statusCode)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error for status %d: %v", tt.statusCode, err)
			}
		})
	}
}

func TestSendBusData_EmptyVehicles(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "")

	testData := &types.ParsedBusData{
		LineRef:     "1",
		Timestamp:   "2024-01-15T10:30:00.000Z",
		VehicleData: []types.VehicleActivity{}, // Empty
	}

	err := client.SendBusData(context.Background(), testData)
	if err != nil {
		t.Fatalf("SendBusData failed: %v", err)
	}

	var pushReq PushRequest
	if err := json.Unmarshal(receivedBody, &pushReq); err != nil {
		t.Fatalf("Failed to parse request body: %v", err)
	}

	// Should have 0 log entries
	if len(pushReq.Streams[0].Values) != 0 {
		t.Errorf("Expected 0 log entries for empty vehicles, got %d", len(pushReq.Streams[0].Values))
	}
}

func TestSendBusData_ServerUnavailable(t *testing.T) {
	// Use a port that's definitely not listening
	client := NewClient("http://127.0.0.1:59999", "", "")

	testData := &types.ParsedBusData{
		LineRef:   "1",
		Timestamp: "2024-01-15T10:30:00.000Z",
		VehicleData: []types.VehicleActivity{
			{VehicleRef: "TEST-001", Longitude: -2.5, Latitude: 51.4},
		},
	}

	err := client.SendBusData(context.Background(), testData)
	if err == nil {
		t.Error("Expected error when server is unavailable, got nil")
	}
}

func TestSendBusData_ContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if context was cancelled
		select {
		case <-r.Context().Done():
			return
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	testData := &types.ParsedBusData{
		LineRef:   "1",
		Timestamp: "2024-01-15T10:30:00.000Z",
		VehicleData: []types.VehicleActivity{
			{VehicleRef: "TEST-001", Longitude: -2.5, Latitude: 51.4},
		},
	}

	err := client.SendBusData(ctx, testData)
	if err == nil {
		t.Error("Expected error when context is cancelled, got nil")
	}
}
