package bods

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-api-key", "7721")

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	// Client is created, verify it's not nil (internal fields are not exported)
	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestFetchBusData_MockServer(t *testing.T) {
	sampleXML := `<?xml version="1.0" encoding="UTF-8"?>
<Siri xmlns="http://www.siri.org.uk/siri" version="2.0">
  <ServiceDelivery>
    <VehicleMonitoringDelivery version="2.0">
      <VehicleActivity>
        <RecordedAtTime>2024-01-15T10:29:45Z</RecordedAtTime>
        <MonitoredVehicleJourney>
          <LineRef>1</LineRef>
          <VehicleRef>TEST-001</VehicleRef>
          <VehicleLocation>
            <Longitude>-2.5</Longitude>
            <Latitude>51.4</Latitude>
          </VehicleLocation>
        </MonitoredVehicleJourney>
      </VehicleActivity>
    </VehicleMonitoringDelivery>
  </ServiceDelivery>
</Siri>`

	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sampleXML))
	}))
	defer server.Close()

	// Create a real client then override the baseURL for testing
	client := NewClient("test-key", "7721")
	client.baseURL = server.URL

	result, err := client.FetchBusData(context.Background(), "1")
	if err != nil {
		t.Fatalf("FetchBusData failed: %v", err)
	}

	// Verify query parameters
	if !strings.Contains(receivedQuery, "api_key=test-key") {
		t.Errorf("Expected api_key in query, got %q", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "lineRef=1") {
		t.Errorf("Expected lineRef in query, got %q", receivedQuery)
	}

	// Verify result
	if result.LineRef != "1" {
		t.Errorf("LineRef = %q, want %q", result.LineRef, "1")
	}
	if !strings.Contains(result.XMLData, "<Siri") {
		t.Error("XMLData should contain SIRI XML")
	}
	if result.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestFetchBusData_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient("test-key", "7721")
	client.baseURL = server.URL

	_, err := client.FetchBusData(context.Background(), "1")
	if err == nil {
		t.Error("Expected error for HTTP 500, got nil")
	}
}

func TestFetchBusData_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient("bad-key", "7721")
	client.baseURL = server.URL

	_, err := client.FetchBusData(context.Background(), "1")
	if err == nil {
		t.Error("Expected error for HTTP 401, got nil")
	}
}

// Integration test - only runs when BODS_API_KEY is set
func TestFetchBusData_Integration(t *testing.T) {
	apiKey := os.Getenv("BODS_API_KEY")
	if apiKey == "" {
		t.Skip("BODS_API_KEY not set, skipping integration test")
	}

	datasetID := os.Getenv("BODS_DATASET_ID")
	if datasetID == "" {
		datasetID = "7721"
	}

	lineRefs := os.Getenv("BODS_LINE_REFS")
	if lineRefs == "" {
		lineRefs = "1,2,3"
	}

	client := NewClient(apiKey, datasetID)

	// Test fetching data for each line
	lines := strings.Split(lineRefs, ",")
	for _, lineRef := range lines {
		lineRef = strings.TrimSpace(lineRef)
		t.Run("Line_"+lineRef, func(t *testing.T) {
			result, err := client.FetchBusData(context.Background(), lineRef)
			if err != nil {
				t.Fatalf("FetchBusData(%q) failed: %v", lineRef, err)
			}

			// Verify we got XML back
			if result.XMLData == "" {
				t.Error("XMLData is empty")
			}
			if !strings.Contains(result.XMLData, "<Siri") {
				t.Error("Response doesn't contain SIRI XML")
			}

			// Verify line ref is set
			if result.LineRef != lineRef {
				t.Errorf("LineRef = %q, want %q", result.LineRef, lineRef)
			}

			// Verify timestamp is set
			if result.Timestamp.IsZero() {
				t.Error("Timestamp is zero")
			}

			t.Logf("Received %d bytes of XML for line %s", len(result.XMLData), lineRef)
		})
	}
}

// Full integration test - fetch and parse
func TestFetchAndParse_Integration(t *testing.T) {
	apiKey := os.Getenv("BODS_API_KEY")
	if apiKey == "" {
		t.Skip("BODS_API_KEY not set, skipping integration test")
	}

	datasetID := os.Getenv("BODS_DATASET_ID")
	if datasetID == "" {
		datasetID = "7721"
	}

	client := NewClient(apiKey, datasetID)

	// Fetch data for line 1
	result, err := client.FetchBusData(context.Background(), "1")
	if err != nil {
		t.Fatalf("FetchBusData failed: %v", err)
	}

	// Log a snippet of the XML for debugging
	if len(result.XMLData) > 500 {
		t.Logf("XML snippet (first 500 chars): %s...", result.XMLData[:500])
	} else {
		t.Logf("XML data: %s", result.XMLData)
	}

	// Check for common SIRI elements
	expectedElements := []string{
		"ServiceDelivery",
		"VehicleMonitoringDelivery",
	}

	for _, elem := range expectedElements {
		if !strings.Contains(result.XMLData, elem) {
			t.Errorf("XML should contain element %q", elem)
		}
	}

	// Check if there are any vehicles
	if strings.Contains(result.XMLData, "VehicleActivity") {
		t.Log("VehicleActivity elements found in response")

		// Check for key vehicle data fields
		vehicleFields := []string{
			"VehicleRef",
			"VehicleLocation",
			"Longitude",
			"Latitude",
		}
		for _, field := range vehicleFields {
			if strings.Contains(result.XMLData, field) {
				t.Logf("Found field: %s", field)
			}
		}

		// Check for new fields we added support for
		newFields := []string{
			"Bearing",
			"Velocity",
			"Occupancy",
			"ProgressStatus",
			"MonitoredCall",
		}
		for _, field := range newFields {
			if strings.Contains(result.XMLData, field) {
				t.Logf("Found new field: %s", field)
			} else {
				t.Logf("New field not present in response: %s (may not be provided by this operator)", field)
			}
		}
	} else {
		t.Log("No VehicleActivity elements found - buses may not be running at this time")
	}
}
