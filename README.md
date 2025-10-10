# BODS2Loki

🚌 A Go service that fetches live bus tracking data from the UK Department for Transport's Bus Open Data Service (BODS) and streams it to Grafana Loki for real-time monitoring and analysis.

## Prerequisites

- BODS API key (get from [data.bus-data.dft.gov.uk](https://data.bus-data.dft.gov.uk))
- Grafana Loki instance (Cloud or OSS)

## Configuration

Create a `.env` file in the project root with the following variables:

```bash
BODS_API_KEY=your_bods_api_key_here
BODS_LINE_REFS=49x,7
BODS_LOKI_URL=http://your-loki-instance

# Optional: For Grafana Cloud Logs authentication
BODS_LOKI_USER=your-grafana-tenant-id
BODS_LOKI_PASSWORD=your-grafana-api-key

# Logging Configuration
LOG_LEVEL=info

# OpenTelemetry Tracing Configuration (Optional)
OTEL_TRACING_ENABLED=true
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=localhost:4318
OTEL_TRACES_SAMPLER=always_on
```

### Authentication Options

#### No Authentication (Default)

If no authentication environment variables are set, the service will connect without authentication (suitable for local Grafana instances).

#### Grafana Cloud Authentication

For Grafana Cloud, use your Tenant ID and API key:

```bash
BODS_LOKI_USER=123456
BODS_LOKI_PASSWORD=glc_123
```

**Note**:
- The `BODS_LOKI_USER` is your Grafana Cloud Logs Tenant ID
- The `BODS_LOKI_PASSWORD` is your Grafana Cloud Access Token (not your login password)
- You can find your Logs Tenant ID in your Grafana Cloud Admin Portal
- Create an Access Token in the Grafana Cloud Admin Portal with appropriate permissions of Logs Write

### Logging Configuration

The application uses structured logging with configurable log levels.

#### Log Levels

Set the `LOG_LEVEL` environment variable to control logging verbosity:

- `debug`: Shows all logs including debug calls, environment variables, and HTTP requests
- `info`: Shows info, warn, and error logs (default)
- `warn`: Shows only warning and error logs
- `error`: Shows only error logs

### OpenTelemetry Tracing Configuration

The application supports distributed tracing using OpenTelemetry. This is optional and disabled by default.

#### Environment Variables

- `OTEL_TRACING_ENABLED`: Set to `true` or `1` to enable tracing
- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`: OTLP HTTP endpoint for traces (default: `localhost:4318`)
- `OTEL_EXPORTER_OTLP_ENDPOINT`: Alternative way to set the endpoint (will append `/v1/traces`)
- `OTEL_EXPORTER_OTLP_TRACES_HEADERS`: Headers for trace export (format: `key1=value1,key2=value2`)
- `OTEL_EXPORTER_OTLP_HEADERS`: Alternative way to set headers
- `OTEL_EXPORTER_OTLP_TRACES_INSECURE`: Set to `true` for insecure connections
- `OTEL_TRACES_SAMPLER`: Sampling strategy (`always_on`, `always_off`, `traceidratio`)

#### Trace Information

When tracing is enabled, the application will create spans for:

- **Main processing cycle**: Overall operation span (`bods2loki.process_cycle`)
- **HTTP data fetch**: Fetching bus data from BODS API (`bods.fetch_bus_data`)
- **XML parsing**: Parsing the bus data (`xml-parser.parse_bus_data`)
- **Loki operations**: Pushing data to Loki (`loki.send_bus_data`)
- **HTTP requests**: Individual HTTP requests to Loki (`loki.http_request`)

Each span includes relevant attributes like HTTP status codes, durations, vehicle counts, and error information.

## Installation

### Building from Source

1. Clone the repository:
```bash
git clone https://github.com/burnettdev/bods2loki.git
cd bods2loki
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o bods2loki
```

4. Run the application:
```bash
./bods2loki
```

### Docker

Build and run with Docker:

```bash
# Build the image
docker build -t bods2loki .

# Run the container
docker run -d \
  --name bods2loki \
  -e BODS_API_KEY=your_bods_api_key_here \
  -e BODS_LINE_REFS=49x,7 \
  -e BODS_LOKI_URL=http://your-loki-instance \
  -e OTEL_TRACING_ENABLED=true \
  -e OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=localhost:4318 \
  --restart unless-stopped \
  bods2loki
```

Pull from Github Container Registry:

```bash
# Run the container
docker run -d \
  --name bods2loki \
  -e BODS_API_KEY=your_bods_api_key_here \
  -e BODS_LINE_REFS=49x,7 \
  -e BODS_LOKI_URL=http://your-loki-instance \
  -e OTEL_TRACING_ENABLED=true \
  -e OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=localhost:4318 \
  --restart unless-stopped \
  ghcr.io/your-username/bods2loki:latest
```

### Docker Compose

For a complete setup with Loki, Grafana, and optional tracing:

```bash
# Copy the example environment file
cp env.example .env

# Edit .env with your configuration
nano .env

# Start the services
docker-compose up -d

# For dry-run mode (safe for testing)
docker-compose run --rm bods2loki --dry-run
```

## Usage

The service will:
- Fetch bus data every 30 seconds (configurable)
- Push the data to Loki with appropriate labels
- Log any errors that occur during the process

### Command Line Options

- `--dry-run`: Print data to stdout instead of sending to Loki
- `--api-key`: BODS API key (required)
- `--line-refs`: Bus line references, comma-separated (default: "49x")
- `--loki-url`: Grafana Loki URL (default: "http://localhost:3100")
- `--loki-user`: Loki username (for Grafana Cloud authentication)
- `--loki-password`: Loki password/token (for Grafana Cloud authentication)
- `--interval`: Polling interval (default: "30s")

## Grafana Cloud Setup

To use with Grafana Cloud Loki:

1. **Get your Grafana Cloud details**:
   - Loki URL: Usually `https://logs-prod-{region}.grafana.net`
   - Username: Your Grafana Cloud instance ID (numeric)
   - Password: Your Grafana Cloud API token

2. **Create a Grafana Cloud API token**:
   - Go to your Grafana Cloud portal
   - Navigate to "API Keys" or "Access Policies"
   - Create a new token with "Logs:Write" permissions

3. **Run with authentication**:
   ```bash
   ./bods2loki --api-key=YOUR_BODS_KEY \
     --line-refs=49x,7 \
     --loki-url=https://logs-prod-us-central1.grafana.net \
     --loki-user=123456 \
     --loki-password=glc_your_token_here
   ```

## Data Structure

The application converts BODS XML data to the following JSON structure:

```json
{
  "line_ref": "49x",
  "timestamp": "2025-10-09T15:37:47.000Z",
  "vehicle_activities": [
    {
      "vehicle_ref": "FBRI-37330",
      "line_ref": "49x",
      "direction_ref": "inbound",
      "operator_ref": "FBRI",
      "origin_ref": "017000005",
      "destination_ref": "0100BRZ00692",
      "origin_aimed_departure_time": "2025-10-09T15:25:00+00:00",
      "destination_aimed_arrival_time": "2025-10-09T16:25:00+00:00",
      "longitude": -2.480741,
      "latitude": 51.495853,
      "recorded_at_time": "2025-10-09T15:37:34+00:00",
      "valid_until_time": "2025-10-09T15:42:47.688+00:00"
    }
  ]
}
```

## OpenTelemetry Tracing

The application includes comprehensive OpenTelemetry tracing:

- HTTP requests to BODS API
- XML parsing operations  
- Loki data transmission
- Pipeline processing metrics

Traces are exported to an OTLP endpoint at `http://localhost:4318` by default.

## Loki Labels

Data sent to Loki includes the following labels:
- `job`: "bods2loki"
- `service`: "bus-tracking"  
- `line_ref`: The bus line reference (e.g., "49x")

## Development

### Running Tests

```bash
go test ./...
```

### Building for Different Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o bods2loki-linux

# Windows  
GOOS=windows GOARCH=amd64 go build -o bods2loki.exe

# macOS ARM
GOOS=darwin GOARCH=arm64 go build -o bods2loki-darwin-arm64
```

## Monitoring

The application provides several observability features:

1. **Logs**: Structured logging with timestamps and context
2. **Traces**: OpenTelemetry spans for all major operations
3. **Metrics**: Vehicle counts and processing durations in traces

## Troubleshooting

### Common Issues

1. **API Key Issues**: Ensure your BODS API key is valid and has access to the datafeed
2. **Network Connectivity**: Check firewall settings for outbound HTTPS (BODS) and HTTP (Loki)
3. **XML Parsing Errors**: The BODS API occasionally returns malformed XML; the app will log and continue
4. **Loki Connection**: Verify Loki is running and accessible at the configured URL

### Debug Mode

For verbose logging, you can modify the log level in the source code or use the dry run mode to inspect data.

## License

MIT License - see LICENSE file for details.
