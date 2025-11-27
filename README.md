# BODS2Loki

🚌 A Go service that fetches live bus tracking data from the UK Department for Transport's Bus Open Data Service (BODS) and streams it to Grafana Loki for real-time monitoring and analysis.

A live demo showing some Bristol bus routes can be found [here.](https://elfordo.grafana.net/public-dashboards/2c542d2643314f34b82fcf7942787443)

## Prerequisites

- BODS API key (get from [data.bus-data.dft.gov.uk](https://data.bus-data.dft.gov.uk))
- Grafana Loki instance (Cloud or OSS)

## Configuration

Create a `.env` file in the project root with the following variables:

```bash
BODS_API_KEY=your_bods_api_key_here
BODS_DATASET_ID=your_bods_dataset_id
BODS_LINE_REFS=bus_line_references
BODS_LOKI_URL=http://your-loki-instance

# Optional: For Grafana Cloud Logs authentication
BODS_LOKI_USER=your-grafana-tenant-id
BODS_LOKI_PASSWORD=your-grafana-api-key

# Logging Configuration
LOG_LEVEL=info

# OpenTelemetry Tracing Configuration (Optional)
OTEL_TRACING_ENABLED=true
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=https://otlp-gateway.example.com/otlp
OTEL_TRACES_SAMPLER=parentbased_always_on
# OTEL_TRACES_SAMPLER_ARG=0.1  # For ratio samplers: 0.1 = 10% sampling

# Pyroscope Profiling Configuration (Optional)
PYROSCOPE_PROFILING_ENABLED=true
PYROSCOPE_SERVER_ADDRESS=https://pyroscope.example.com
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
- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`: Full OTLP HTTP endpoint URL for traces (e.g., `https://otlp-gateway.grafana.net/otlp`)
- `OTEL_EXPORTER_OTLP_ENDPOINT`: Alternative way to set the endpoint (will append `/v1/traces` automatically)
- `OTEL_EXPORTER_OTLP_TRACES_HEADERS`: Headers for trace export (format: `key1=value1,key2=value2`)
- `OTEL_EXPORTER_OTLP_TRACES_INSECURE`: Override secure/insecure mode (`true` for HTTP, `false` for HTTPS). If not set, determined automatically from URL scheme.
- `OTEL_TRACES_SAMPLER`: Sampling strategy (default: `parentbased_always_on`)
  - `always_on`: Sample all traces (100%)
  - `always_off`: Disable tracing
  - `traceidratio`: Sample a percentage of traces
  - `parentbased_always_on`: Parent-based, samples all root spans
  - `parentbased_always_off`: Parent-based, never samples root spans
  - `parentbased_traceidratio`: Parent-based with ratio sampling for root spans
- `OTEL_TRACES_SAMPLER_ARG`: Sampler argument (for ratio samplers: `0.0`-`1.0`, e.g., `0.1` = 10%)

#### URL Format

The endpoint accepts full URLs including scheme and path:
- `https://otlp-gateway-prod-gb-south-1.grafana.net/otlp` - HTTPS with custom path
- `http://localhost:4318` - Local development with HTTP
- `otlp-gateway.example.com/otlp` - Without scheme (defaults to HTTPS)

The URL scheme (`http://` vs `https://`) automatically determines whether to use secure connections unless overridden by `OTEL_EXPORTER_OTLP_TRACES_INSECURE`.

#### Trace Information

When tracing is enabled, the application will create spans for:

- **Main processing cycle**: Overall operation span (`bods2loki.process_cycle`)
- **HTTP data fetch**: Fetching bus data from BODS API (`bods.fetch_bus_data`)
- **XML parsing**: Parsing the bus data (`xml-parser.parse_bus_data`)
- **Loki operations**: Pushing data to Loki (`loki.send_bus_data`)
- **HTTP requests**: Individual HTTP requests to Loki (`loki.http_request`)

Each span includes relevant attributes like HTTP status codes, durations, vehicle counts, and error information.

### Pyroscope Profiling Configuration

The application supports continuous profiling using Pyroscope. This is optional and disabled by default.

#### Environment Variables

- `PYROSCOPE_PROFILING_ENABLED`: Set to `true` or `1` to enable profiling
- `PYROSCOPE_SERVER_ADDRESS`: Pyroscope server address (default: `http://localhost:4040`)
- `PYROSCOPE_APPLICATION_NAME`: Application name for profiling (default: `bods2loki`)
- `PYROSCOPE_BASIC_AUTH_USER`: Basic auth username
- `PYROSCOPE_BASIC_AUTH_PASSWORD`: Basic auth password

#### Profile Types

When profiling is enabled, the application will collect:

- **CPU profiles**: CPU usage and hotspots
- **Memory profiles**: Memory allocation and usage patterns
- **Goroutine profiles**: Goroutine counts and stack traces
- **Mutex profiles**: Lock contention analysis
- **Block profiles**: Blocking operation analysis

#### Authentication

For servers requiring authentication, use basic auth:

```bash
PYROSCOPE_PROFILING_ENABLED=true
PYROSCOPE_SERVER_ADDRESS=https://your-pyroscope-server.com
PYROSCOPE_BASIC_AUTH_USER=your_username
PYROSCOPE_BASIC_AUTH_PASSWORD=your_password
```

### OpenTelemetry Metrics Configuration

The application supports metrics collection and export using OpenTelemetry. This is optional and disabled by default.

#### Environment Variables

**Feature Gate:**
- `OTEL_METRICS_ENABLED`: Set to `true` or `1` to enable metrics

**Base OTLP Configuration (applies to all signals unless overridden):**
- `OTEL_EXPORTER_OTLP_ENDPOINT`: Base URL for all signals (auto-appends `/v1/traces` and `/v1/metrics`)
- `OTEL_EXPORTER_OTLP_PROTOCOL`: Transport protocol (`grpc`, `http/protobuf`, `http/json`). Default: `http/protobuf`
- `OTEL_EXPORTER_OTLP_HEADERS`: Headers in `key=value` format (multiple: `key1=value1,key2=value2`)
- `OTEL_EXPORTER_OTLP_TIMEOUT`: Export timeout in milliseconds (default: `10000` = 10 seconds)
- `OTEL_EXPORTER_OTLP_INSECURE`: Disable TLS (`true` for HTTP, `false` for HTTPS)
- `OTEL_EXPORTER_OTLP_COMPRESSION`: Compression (`none`, `gzip`)

**Metrics-Specific Overrides (take precedence over base configuration):**
- `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT`: Full endpoint URL for metrics (use as-is, no path appending)
- `OTEL_EXPORTER_OTLP_METRICS_PROTOCOL`: Protocol override for metrics
- `OTEL_EXPORTER_OTLP_METRICS_HEADERS`: Headers override for metrics
- `OTEL_EXPORTER_OTLP_METRICS_TIMEOUT`: Timeout override for metrics
- `OTEL_EXPORTER_OTLP_METRICS_INSECURE`: Insecure mode override for metrics
- `OTEL_EXPORTER_OTLP_METRICS_COMPRESSION`: Compression override for metrics

#### Available Metrics

When metrics is enabled, the application exports the following metrics:

**HTTP Client Metrics (OTEL Semantic Conventions):**
- `http.client.request.duration`: Duration of HTTP client requests (histogram)
- `http.client.request.body.size`: Size of HTTP request bodies (histogram)
- `http.client.response.body.size`: Size of HTTP response bodies (histogram)

**Pipeline Metrics:**
- `pipeline.cycles.total`: Total pipeline processing cycles (counter)
- `pipeline.cycle.duration`: Duration of pipeline cycles (histogram)
- `pipeline.lines.processed`: Number of bus lines processed (counter)
- `pipeline.vehicles.processed`: Number of vehicles processed (counter)
- `pipeline.lines.in_flight`: Lines currently being processed (updown counter)

**Parser Metrics:**
- `xml.parse.duration`: Duration of XML parsing operations (histogram)

**Runtime Metrics:**
- `runtime.go.goroutines`: Current goroutine count (gauge)
- `pipeline.last_success.timestamp`: Unix timestamp of last successful cycle (gauge)

#### Example Configurations

**Single Endpoint for Both Traces and Metrics (Grafana Cloud):**
```bash
OTEL_TRACING_ENABLED=true
OTEL_METRICS_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=https://otlp-gateway-prod-gb-south-1.grafana.net/otlp
# Generate base64 credentials: echo -n "instanceId:apiToken" | base64
OTEL_EXPORTER_OTLP_HEADERS=Authorization=Basic <your-base64-encoded-credentials>
```

**Separate Endpoints:**
```bash
OTEL_TRACING_ENABLED=true
OTEL_METRICS_ENABLED=true
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=https://tempo.example.com/otlp/v1/traces
OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=https://mimir.example.com/otlp/v1/metrics
```

**gRPC Protocol:**
```bash
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
OTEL_EXPORTER_OTLP_INSECURE=true
```

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
  -e BODS_LINE_REFS=your_bus_lines_reference_numbers \
  -e BODS_LOKI_URL=http://your-loki-instance \
  -e OTEL_TRACING_ENABLED=true \
  -e OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://localhost:4318 \
  --restart unless-stopped \
  bods2loki
```

Pull from Github Container Registry:

```bash
# Run the container
docker run -d \
  --name bods2loki \
  -e BODS_API_KEY=your_bods_api_key_here \
  -e BODS_LINE_REFS=your_bus_lines_reference_numbers \
  -e BODS_LOKI_URL=http://your-loki-instance \
  -e OTEL_TRACING_ENABLED=true \
  -e OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://localhost:4318 \
  --restart unless-stopped \
  ghcr.io/burnettdev/bods2loki:main
```

### Docker Compose

For a complete setup using Docker Compose with environment variables from a `.env` file:

```bash
# Copy the example environment file
cp env.example .env

# Edit .env with your configuration
nano .env

# Start the services
docker-compose up -d

# View logs
docker-compose logs -f bods2loki

# For dry-run mode (safe for testing)
docker-compose run --rm bods2loki --dry-run
```

#### Supported Environment Variables

The `docker-compose.yml` passes the following environment variables from your `.env` file:

**Required:**
- `BODS_API_KEY` - Your BODS API key

**BODS Configuration:**
- `BODS_DATASET_ID` - BODS dataset ID (default: `699`)
- `BODS_LINE_REFS` - Bus line references (default: `49x`)
- `BODS_INTERVAL` - Polling interval (default: `30s`)

**Loki Configuration:**
- `BODS_LOKI_URL` - Loki endpoint (default: `http://loki:3100`)
- `BODS_LOKI_USER` - Loki username (for Grafana Cloud)
- `BODS_LOKI_PASSWORD` - Loki password/token (for Grafana Cloud)

**Logging:**
- `LOG_LEVEL` - Log level (default: `info`)

**OpenTelemetry Tracing:**
- `OTEL_TRACING_ENABLED` - Enable tracing (default: `false`)
- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` - OTLP endpoint URL (default: `http://localhost:4318`)
- `OTEL_TRACES_SAMPLER` - Sampling strategy (default: `parentbased_always_on`)
- `OTEL_TRACES_SAMPLER_ARG` - Sampler argument (for ratio samplers: `0.0`-`1.0`)
- `OTEL_EXPORTER_OTLP_TRACES_INSECURE` - Force insecure mode
- `OTEL_EXPORTER_OTLP_TRACES_HEADERS` - Custom headers (format: `key1=value1,key2=value2`)

**OpenTelemetry Metrics:**
- `OTEL_METRICS_ENABLED` - Enable metrics (default: `false`)
- `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` - OTLP endpoint URL for metrics
- `OTEL_EXPORTER_OTLP_METRICS_HEADERS` - Custom headers for metrics
- `OTEL_EXPORTER_OTLP_METRICS_INSECURE` - Force insecure mode for metrics

**Shared OTEL Configuration:**
- `OTEL_EXPORTER_OTLP_ENDPOINT` - Base endpoint (auto-appends signal paths)
- `OTEL_EXPORTER_OTLP_PROTOCOL` - Protocol (`grpc`, `http/protobuf`, `http/json`)
- `OTEL_EXPORTER_OTLP_HEADERS` - Shared headers (`key=value` format)
- `OTEL_EXPORTER_OTLP_TIMEOUT` - Export timeout in milliseconds (default: `10000`)
- `OTEL_EXPORTER_OTLP_COMPRESSION` - Compression (`none`, `gzip`)
- `OTEL_EXPORTER_OTLP_INSECURE` - Disable TLS verification

**Pyroscope Profiling:**
- `PYROSCOPE_PROFILING_ENABLED` - Enable profiling (default: `false`)
- `PYROSCOPE_SERVER_ADDRESS` - Pyroscope server URL (default: `http://localhost:4040`)
- `PYROSCOPE_APPLICATION_NAME` - Application name (default: `bods2loki`)
- `PYROSCOPE_BASIC_AUTH_USER` - Basic auth username
- `PYROSCOPE_BASIC_AUTH_PASSWORD` - Basic auth password

#### Example `.env` for Grafana Cloud

```bash
# BODS API
BODS_API_KEY=your_bods_api_key
BODS_LINE_REFS=49x,7

# Grafana Cloud Loki
BODS_LOKI_URL=https://logs-prod-gb-south-1.grafana.net
BODS_LOKI_USER=123456
BODS_LOKI_PASSWORD=glc_your_token

# Grafana Cloud OTEL (Traces and Metrics via single endpoint)
OTEL_TRACING_ENABLED=true
OTEL_METRICS_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=https://otlp-gateway-prod-gb-south-1.grafana.net/otlp
OTEL_EXPORTER_OTLP_HEADERS=Authorization=Basic base64encodedcreds

# Grafana Cloud Pyroscope (Profiling)
PYROSCOPE_PROFILING_ENABLED=true
PYROSCOPE_SERVER_ADDRESS=https://profiles-prod-gb-south-1.grafana.net
PYROSCOPE_BASIC_AUTH_USER=123456
PYROSCOPE_BASIC_AUTH_PASSWORD=glc_your_token
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

Traces are exported to an OTLP endpoint (default: `http://localhost:4318`). The endpoint accepts full URLs with scheme and path, e.g., `https://otlp-gateway.grafana.net/otlp`.

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

<img width="auth" height="150" alt="image" src="https://github.com/user-attachments/assets/cbb403e2-7042-42af-8124-422128e146e8" />

BODS2Loki is proudly part of Snyk's Open Source Programme, ensuring vulnerabilities are found sooner rather than later. You can find more information <a href="https://snyk.io/?utm_source=open-source&utm_medium=pg-ptr&utm_campaign=ref-2501-osp&utm_content=pg-cta">here.</a>

