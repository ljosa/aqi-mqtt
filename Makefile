.PHONY: all build test test-unit test-e2e run clean help

# Default target
all: build

# Build the daemon
build:
	go build -o aqi-mqtt-daemon

# Cross-compile for Linux AMD64
build-linux:
	GOOS=linux GOARCH=amd64 go build -o aqi-mqtt-daemon-linux-amd64

# Run all tests
test:
	go test -v ./...

# Run only unit tests (no Docker required)
test-unit:
	go test -v -run "TestAQI"

# Run only end-to-end tests (requires Docker)
test-e2e:
	go test -v -run "TestEndToEnd"

# Run the daemon
run: build
	./aqi-mqtt-daemon

# Clean build artifacts
clean:
	rm -f aqi-mqtt-daemon aqi-mqtt-daemon-linux-amd64
	go clean

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Check if Docker is running (helper for e2e tests)
check-docker:
	@docker info > /dev/null 2>&1 || (echo "Docker is not running. Please start Docker to run e2e tests." && exit 1)

# Run e2e tests with Docker check
test-e2e-safe: check-docker test-e2e

# Help target
help:
	@echo "Available targets:"
	@echo "  make build       - Build the daemon binary"
	@echo "  make build-linux - Cross-compile for Linux AMD64"
	@echo "  make test        - Run all tests"
	@echo "  make test-unit   - Run unit tests only (no Docker required)"
	@echo "  make test-e2e    - Run end-to-end tests (requires Docker)"
	@echo "  make run         - Build and run the daemon"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make deps        - Download and tidy dependencies"
	@echo "  make fmt         - Format Go code"
	@echo "  make lint        - Run linter (requires golangci-lint)"
	@echo "  make help        - Show this help message"