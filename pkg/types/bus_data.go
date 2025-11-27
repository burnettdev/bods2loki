package types

type ParsedBusData struct {
	LineRef     string                 `json:"line_ref"`
	Timestamp   string                 `json:"timestamp"`
	VehicleData []VehicleActivity      `json:"vehicle_activities"`
	RawData     map[string]interface{} `json:"raw_data,omitempty"`
}

type VehicleActivity struct {
	VehicleRef                  string  `json:"vehicle_ref"`
	LineRef                     string  `json:"line_ref"`
	DirectionRef                string  `json:"direction_ref"`
	OperatorRef                 string  `json:"operator_ref"`
	OriginRef                   string  `json:"origin_ref"`
	OriginName                  string  `json:"origin_name"`
	DestinationRef              string  `json:"destination_ref"`
	DestinationName             string  `json:"destination_name"`
	OriginAimedDepartureTime    string  `json:"origin_aimed_departure_time"`
	DestinationAimedArrivalTime string  `json:"destination_aimed_arrival_time"`
	Longitude                   float64 `json:"longitude"`
	Latitude                    float64 `json:"latitude"`
	RecordedAtTime              string  `json:"recorded_at_time"`
	ValidUntilTime              string  `json:"valid_until_time"`
	BusImage                    string  `json:"bus_image"`

	// Additional SIRI-VM fields for enhanced Grafana visualization
	Bearing           float64 `json:"bearing,omitempty"`            // Vehicle heading direction (0-360 degrees)
	Velocity          float64 `json:"velocity,omitempty"`           // Speed in meters per second
	Occupancy         string  `json:"occupancy,omitempty"`          // Passenger load: full|seatsAvailable|standingAvailable
	ProgressStatus    string  `json:"progress_status,omitempty"`    // Status: normalProgress|noProgress|layover|prevTrip
	PublishedLineName string  `json:"published_line_name,omitempty"` // Customer-facing route name
	BlockRef          string  `json:"block_ref,omitempty"`          // Operational block identifier

	// Stop prediction data
	MonitoredCall *StopCall   `json:"monitored_call,omitempty"` // Current/next stop with ETA
	OnwardCalls   []StopCall  `json:"onward_calls,omitempty"`   // Future stops with predictions
}

// StopCall represents arrival/departure information for a stop
type StopCall struct {
	StopPointRef          string `json:"stop_point_ref"`
	StopPointName         string `json:"stop_point_name,omitempty"`
	AimedArrivalTime      string `json:"aimed_arrival_time,omitempty"`
	ExpectedArrivalTime   string `json:"expected_arrival_time,omitempty"`
	AimedDepartureTime    string `json:"aimed_departure_time,omitempty"`
	ExpectedDepartureTime string `json:"expected_departure_time,omitempty"`
	VisitNumber           int    `json:"visit_number,omitempty"`
}
