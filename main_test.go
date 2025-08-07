package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	testBrokerPort  = "21883"
	testBroker      = "tcp://localhost:" + testBrokerPort
	testInputTopic  = "test/airgradient/readings"
	testOutputTopic = "test/aqi"
	containerName   = "mqtt-test-broker"
)

// startMosquitto starts a Mosquitto MQTT broker in Docker
func startMosquitto(t *testing.T) {
	t.Helper()

	// Stop any existing container
	_ = exec.Command("docker", "stop", containerName).Run()
	_ = exec.Command("docker", "rm", containerName).Run()

	// Get current working directory for mounting config
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Start Mosquitto container with custom config
	cmd := exec.Command("docker", "run", "-d",
		"--name", containerName,
		"-p", testBrokerPort+":1883",
		"-v", pwd+"/test-mosquitto.conf:/mosquitto/config/mosquitto.conf",
		"eclipse-mosquitto")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if Docker is running
		if dockerErr := exec.Command("docker", "info").Run(); dockerErr != nil {
			t.Fatalf("Docker is not running. Please start Docker to run end-to-end tests.")
		}
		// Check if port is already in use
		if execErr := exec.Command("lsof", "-i", ":"+testBrokerPort).Run(); execErr == nil {
			t.Fatalf("Port %s is already in use. Please free the port or use a different port.", testBrokerPort)
		}
		t.Fatalf("Failed to start Mosquitto: %v\nOutput: %s", err, output)
	}

	// Wait for broker to be ready
	waitForBroker(t, testBroker)
}

// waitForBroker waits for the MQTT broker to be ready
func waitForBroker(t *testing.T, broker string) {
	t.Helper()

	// Try to connect for up to 10 seconds
	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		opts := mqtt.NewClientOptions()
		opts.AddBroker(broker)
		opts.SetClientID("test-wait-client")
		opts.SetConnectTimeout(1 * time.Second)

		client := mqtt.NewClient(opts)
		token := client.Connect()

		if token.WaitTimeout(1*time.Second) && token.Error() == nil {
			client.Disconnect(250)
			return // Broker is ready
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatal("Timeout waiting for broker to be ready")
}

// stopMosquitto stops and removes the Mosquitto container
func stopMosquitto(t *testing.T) {
	t.Helper()

	cmd := exec.Command("docker", "stop", containerName)
	if err := cmd.Run(); err != nil {
		t.Logf("Failed to stop container: %v", err)
	}

	cmd = exec.Command("docker", "rm", containerName)
	if err := cmd.Run(); err != nil {
		t.Logf("Failed to remove container: %v", err)
	}
}

// createTestClient creates an MQTT client for testing
func createTestClient(t *testing.T, clientID string) mqtt.Client {
	t.Helper()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(testBroker)
	opts.SetClientID(clientID)
	opts.SetConnectTimeout(5 * time.Second)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(5 * time.Second) {
		t.Fatal("Timeout connecting to broker")
	}
	if err := token.Error(); err != nil {
		t.Fatalf("Failed to connect to broker: %v", err)
	}

	return client
}

// waitForDaemonReady waits for the daemon to be ready by checking if it responds to messages
func waitForDaemonReady(t *testing.T, inputTopic string) bool {
	t.Helper()

	// Create a test client to verify daemon is ready
	verifyClient := createTestClient(t, "verify-daemon-client")
	defer verifyClient.Disconnect(250)

	// Subscribe to the output topic to see if daemon responds
	readyChan := make(chan bool, 1)
	token := verifyClient.Subscribe(testOutputTopic, 1, func(client mqtt.Client, msg mqtt.Message) {
		readyChan <- true
	})
	if !token.WaitTimeout(2*time.Second) || token.Error() != nil {
		t.Logf("Failed to subscribe for readiness check: %v", token.Error())
		return false
	}
	defer verifyClient.Unsubscribe(testOutputTopic)

	// Try for up to 5 seconds
	deadline := time.Now().Add(5 * time.Second)
	
	for time.Now().Before(deadline) {
		// Send a small test message to see if daemon processes it
		testMsg := `{"pm02Standard": 10.0, "pm10Standard": 10.0}`
		token := verifyClient.Publish(inputTopic, 0, false, []byte(testMsg))
		if token.WaitTimeout(1*time.Second) && token.Error() == nil {
			// Wait for response
			select {
			case <-readyChan:
				t.Log("Daemon is ready - received response to test message")
				return true
			case <-time.After(500 * time.Millisecond):
				// Try again
			}
		}
		
		time.Sleep(200 * time.Millisecond)
	}
	
	return false
}

func TestEndToEndHappyPath(t *testing.T) {
	// Start Mosquitto
	startMosquitto(t)
	defer stopMosquitto(t)

	// Create test client
	testClient := createTestClient(t, "test-client")
	defer testClient.Disconnect(250)

	// Channel to receive output message
	outputChan := make(chan *AQIReading, 1)

	// Subscribe to output topic
	token := testClient.Subscribe(testOutputTopic, 1, func(client mqtt.Client, msg mqtt.Message) {
		var reading AQIReading
		if err := json.Unmarshal(msg.Payload(), &reading); err != nil {
			t.Errorf("Failed to parse output message: %v", err)
			return
		}
		outputChan <- &reading
	})

	if !token.WaitTimeout(5*time.Second) || token.Error() != nil {
		t.Fatalf("Failed to subscribe to output topic: %v", token.Error())
	}

	// Build the daemon
	buildCmd := exec.Command("go", "build", "-o", "test-aqi-daemon", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build daemon: %v", err)
	}
	defer os.Remove("test-aqi-daemon")

	// Start the daemon with proper command-line flags
	daemonCmd := exec.Command("./test-aqi-daemon",
		"-broker", "localhost",
		"-port", testBrokerPort,
		"-input-topic", testInputTopic,
		"-output-topic", testOutputTopic,
		"-client-id", "aqi-daemon-test")
	
	// Capture daemon output for debugging in test logs
	// This helps when tests fail to see what the daemon was doing
	if testing.Verbose() {
		daemonCmd.Stdout = os.Stdout
		daemonCmd.Stderr = os.Stderr
	}
	
	if err := daemonCmd.Start(); err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}
	defer func() {
		if err := daemonCmd.Process.Kill(); err != nil {
			t.Logf("Failed to kill daemon process: %v", err)
		}
		daemonCmd.Wait()
	}()

	// Wait for daemon to be ready by checking if it can accept connections
	if !waitForDaemonReady(t, testInputTopic) {
		t.Fatal("Daemon failed to become ready within timeout")
	}

	// Wait a moment and clear any readiness check messages
	time.Sleep(500 * time.Millisecond)
	for {
		select {
		case <-outputChan:
			// Discard any pending message from readiness check
			t.Log("Cleared a readiness check message from output channel")
		default:
			// No more pending messages
			goto done
		}
	}
	done:

	// Prepare test input
	testInput := SensorReading{
		PM01:            2,
		PM02:            4,
		PM10:            4,
		PM01Standard:    2,
		PM02Standard:    35.7, // Should result in AQI ~102
		PM10Standard:    45,   // Should result in AQI ~41
		PM003Count:      303.5,
		PM005Count:      249.67,
		PM01Count:       39.5,
		PM02Count:       2,
		Atmp:            24.1,
		AtmpCompensated: 23.35,
		Rhum:            60.7,
		RhumCompensated: 83.76,
		PM02Compensated: 2.61,
		RCO2:            417,
		TVOCIndex:       48,
		TVOCRaw:         32520.83,
		NOXIndex:        2,
		NOXRaw:          17731.08,
		Boot:            2378,
		BootCount:       2378,
		Wifi:            -69,
		SerialNo:        "d83bda1d7660",
		Firmware:        "3.2.0",
		Model:           "O-1PST",
	}

	// Publish test message
	inputJSON, err := json.Marshal(testInput)
	if err != nil {
		t.Fatalf("Failed to marshal test input: %v", err)
	}

	token = testClient.Publish(testInputTopic, 1, false, inputJSON)
	if !token.WaitTimeout(5*time.Second) || token.Error() != nil {
		t.Fatalf("Failed to publish test message: %v", token.Error())
	}

	// Wait for output
	select {
	case output := <-outputChan:
		// Verify output contains original data
		if output.SerialNo != testInput.SerialNo {
			t.Errorf("Serial number mismatch: got %s, want %s", output.SerialNo, testInput.SerialNo)
		}
		if output.PM02Standard != testInput.PM02Standard {
			t.Errorf("PM2.5 mismatch: got %f, want %f", output.PM02Standard, testInput.PM02Standard)
		}

		// Verify AQI was calculated
		if output.AQI == 0 {
			t.Error("AQI was not calculated (is 0)")
		}

		// Based on PM2.5=35.7 and PM10=45, the AQI should be ~102 (from PM2.5)
		expectedAQI := 102
		tolerance := 2
		if output.AQI < expectedAQI-tolerance || output.AQI > expectedAQI+tolerance {
			t.Errorf("AQI calculation seems incorrect: got %d, expected ~%d", output.AQI, expectedAQI)
		}

		t.Logf("Successfully received AQI: %d", output.AQI)

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for output message")
	}
}

// TestAQICalculation tests the AQI calculation logic directly
func TestAQICalculation(t *testing.T) {
	testCases := []struct {
		name     string
		pm25     float64
		pm10     float64
		expected int
	}{
		{"Good air quality", 8.0, 20.0, 33},
		{"Moderate air quality", 35.4, 50.0, 100},
		{"Unhealthy for sensitive groups", 55.4, 100.0, 150},
		{"Very unhealthy", 250.4, 350.0, 300},
		{"Hazardous", 400.0, 500.0, 434},
		{"PM10 dominant", 10.0, 200.0, 123}, // PM10 AQI higher than PM2.5
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := computeAQI(tc.pm25, tc.pm10)
			// Allow small tolerance for rounding
			if result < tc.expected-1 || result > tc.expected+1 {
				t.Errorf("computeAQI(%f, %f) = %d, want ~%d", tc.pm25, tc.pm10, result, tc.expected)
			}
		})
	}
}

// TestAQIBreakpointEdgeCases tests edge cases in AQI calculation
func TestAQIBreakpointEdgeCases(t *testing.T) {
	// Test exact breakpoint values
	testCases := []struct {
		pm25     float64
		expected int
	}{
		{0.0, 0},     // Minimum
		{12.0, 50},   // Exact breakpoint
		{12.1, 51},   // Just over breakpoint
		{35.4, 100},  // Exact breakpoint
		{35.5, 101},  // Just over breakpoint
		{500.4, 500}, // Maximum defined breakpoint
		{600.0, 500}, // Beyond maximum (should cap at 500)
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("PM2.5=%.1f", tc.pm25), func(t *testing.T) {
			result := calculateAQI(tc.pm25, pm25Breakpoints)
			if result != tc.expected {
				t.Errorf("calculateAQI(%f) = %d, want %d", tc.pm25, result, tc.expected)
			}
		})
	}
}

// TestPM10BreakpointGap tests the critical gap between 54 and 55 for PM10
func TestPM10BreakpointGap(t *testing.T) {
	// Test PM10 values around the 54-55 boundary where the bug occurred
	testCases := []struct {
		pm10     float64
		expected int
	}{
		{53.0, 48},  // Just below first breakpoint upper bound
		{54.0, 49},  // At first breakpoint upper bound
		{54.5, 50},  // In the gap - should be in first tier
		{54.9, 50},  // Just below 55
		{55.0, 51},  // At second breakpoint lower bound
		{55.1, 51},  // Just above 55
		{100.0, 73}, // Middle value in second tier
		{154.0, 100}, // Near upper bound of second tier
		{154.5, 100}, // In the gap between 154 and 155
		{155.0, 101}, // At third breakpoint lower bound
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("PM10=%.1f", tc.pm10), func(t *testing.T) {
			result := calculateAQI(tc.pm10, pm10Breakpoints)
			if result != tc.expected {
				t.Errorf("calculateAQI(PM10=%f) = %d, want %d", tc.pm10, result, tc.expected)
			}
		})
	}
}
