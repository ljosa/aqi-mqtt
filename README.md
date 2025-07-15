# AQI MQTT Daemon

A Go daemon that listens to AirGradient sensor readings via MQTT and calculates the Air Quality Index (AQI) based on PM2.5 and PM10 concentrations.

## Features

- Connects to MQTT broker and subscribes to AirGradient sensor topic
- Parses incoming JSON sensor data
- Calculates AQI using EPA methodology
- Publishes enriched data with AQI value to output topic
- Configurable MQTT broker, topics, and client ID via command-line flags
- Automatic unique client ID generation to prevent conflicts
- Version information with git commit and build time
- Graceful shutdown on interrupt signals

## Prerequisites

- Go 1.19 or higher
- Access to an MQTT broker
- AirGradient sensor publishing to MQTT

## Installation

```bash
# Clone the repository
cd /path/to/aqi-mqtt

# Download dependencies
go mod tidy

# Build the daemon
make build

# Or manually with version info
GIT_COMMIT=$(git describe --always --dirty)
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
go build -ldflags "-X main.GitCommit=$GIT_COMMIT -X main.BuildTime=$BUILD_TIME" -o aqi-mqtt-daemon
```

## Usage

### Basic Usage

Run the daemon with required parameters:
```bash
./aqi-mqtt-daemon -broker <host> -input-topic <topic> -output-topic <topic>
```

### Command-Line Options

**Required:**
- `-broker` - MQTT broker hostname or IP address
- `-input-topic` - MQTT topic to subscribe for sensor readings
- `-output-topic` - MQTT topic to publish AQI data

**Optional:**
- `-port` - MQTT broker port (default: 1883)
- `-client-id` - MQTT client ID (default: aqi-mqtt-<pid>)
- `--version` - Print version information and exit

### Examples

```bash
# Connect to local broker with default port
./aqi-mqtt-daemon -broker localhost -input-topic airgradient/readings/sensor1 -output-topic aqi/sensor1

# Connect to remote broker with custom port
./aqi-mqtt-daemon -broker 192.168.1.100 -port 1884 -input-topic sensors/air -output-topic processed/aqi

# Use custom client ID
./aqi-mqtt-daemon -broker mqtt.example.com -input-topic input -output-topic output -client-id my-aqi-processor

# Check version
./aqi-mqtt-daemon --version
```

## Input Format

The daemon expects JSON messages from AirGradient sensors containing at minimum:
- `pm02Standard`: PM2.5 concentration in µg/m³
- `pm10Standard`: PM10 concentration in µg/m³

## Output Format

The daemon publishes the original message with an added `aqi` field:
```json
{
  "pm02Standard": 35.4,
  "pm10Standard": 45.0,
  ...other fields...,
  "aqi": 101
}
```

## AQI Calculation

See [AQI_DOCUMENTATION.md](AQI_DOCUMENTATION.md) for detailed information about:
- EPA AQI formula and methodology
- Breakpoint tables for PM2.5 and PM10
- Health implications of different AQI levels
- References to official EPA documentation

## Testing

The project includes comprehensive tests with an end-to-end test using Docker:

```bash
# Run all tests (requires Docker)
make test

# Run only unit tests (no Docker required)
make test-unit

# Run only end-to-end tests (requires Docker)
make test-e2e
```

The end-to-end test:
- Spins up a Mosquitto MQTT broker in Docker
- Publishes a test message with sensor data
- Verifies the daemon calculates and publishes the correct AQI
- Automatically cleans up the Docker container

## Development

To modify the AQI calculation or add support for additional pollutants:
1. Update breakpoint tables in `main.go`
2. Modify `computeAQI()` function to include new pollutants
3. Update documentation accordingly
4. Add corresponding test cases in `main_test.go`

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.