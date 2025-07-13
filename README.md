# AQI MQTT Daemon

A Go daemon that listens to AirGradient sensor readings via MQTT and calculates the Air Quality Index (AQI) based on PM2.5 and PM10 concentrations.

## Features

- Connects to MQTT broker and subscribes to AirGradient sensor topic
- Parses incoming JSON sensor data
- Calculates AQI using EPA methodology
- Publishes enriched data with AQI value to output topic
- Graceful shutdown on interrupt signals

## Prerequisites

- Go 1.19 or higher
- Access to MQTT broker (default: 192.168.2.71:1883)
- AirGradient sensor publishing to MQTT

## Installation

```bash
# Clone the repository
cd /path/to/aqi-mqtt

# Download dependencies
go mod tidy

# Build the daemon
go build -o aqi-mqtt-daemon
```

## Usage

Run the daemon:
```bash
./aqi-mqtt-daemon
```

The daemon will:
1. Connect to MQTT broker at `192.168.2.71:1883`
2. Subscribe to topic `airgradient/readings/d83bda1d7660`
3. Calculate AQI for each incoming message
4. Publish results to topic `aqi`

## Configuration

Currently, the MQTT broker and topics are hardcoded in `main.go`. To modify:
- Broker address: Line 109
- Input topic: Line 110  
- Output topic: Line 111

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
go test -v

# Run only unit tests (no Docker required)
go test -v -run TestAQICalculation
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

This project is provided as-is for educational and monitoring purposes.