package main

import (
	"encoding/json"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// SensorReading represents the incoming sensor data
type SensorReading struct {
	PM01            float64 `json:"pm01"`
	PM02            float64 `json:"pm02"`
	PM10            float64 `json:"pm10"`
	PM01Standard    float64 `json:"pm01Standard"`
	PM02Standard    float64 `json:"pm02Standard"`
	PM10Standard    float64 `json:"pm10Standard"`
	PM003Count      float64 `json:"pm003Count"`
	PM005Count      float64 `json:"pm005Count"`
	PM01Count       float64 `json:"pm01Count"`
	PM02Count       float64 `json:"pm02Count"`
	Atmp            float64 `json:"atmp"`
	AtmpCompensated float64 `json:"atmpCompensated"`
	Rhum            float64 `json:"rhum"`
	RhumCompensated float64 `json:"rhumCompensated"`
	PM02Compensated float64 `json:"pm02Compensated"`
	RCO2            float64 `json:"rco2"`
	TVOCIndex       float64 `json:"tvocIndex"`
	TVOCRaw         float64 `json:"tvocRaw"`
	NOXIndex        float64 `json:"noxIndex"`
	NOXRaw          float64 `json:"noxRaw"`
	Boot            int     `json:"boot"`
	BootCount       int     `json:"bootCount"`
	Wifi            int     `json:"wifi"`
	SerialNo        string  `json:"serialno"`
	Firmware        string  `json:"firmware"`
	Model           string  `json:"model"`
}

// AQIReading extends SensorReading with AQI value
type AQIReading struct {
	SensorReading
	AQI int `json:"aqi"`
}

// AQI breakpoint structure for calculations
type AQIBreakpoint struct {
	ConcLow  float64
	ConcHigh float64
	AQILow   int
	AQIHigh  int
}

// PM2.5 AQI breakpoints based on EPA standards
// Source: https://www.airnow.gov/sites/default/files/2020-05/aqi-technical-assistance-document-sept2018.pdf
var pm25Breakpoints = []AQIBreakpoint{
	{0.0, 12.0, 0, 50},
	{12.1, 35.4, 51, 100},
	{35.5, 55.4, 101, 150},
	{55.5, 150.4, 151, 200},
	{150.5, 250.4, 201, 300},
	{250.5, 350.4, 301, 400},
	{350.5, 500.4, 401, 500},
}

// PM10 AQI breakpoints based on EPA standards
var pm10Breakpoints = []AQIBreakpoint{
	{0, 54, 0, 50},
	{55, 154, 51, 100},
	{155, 254, 101, 150},
	{255, 354, 151, 200},
	{355, 424, 201, 300},
	{425, 504, 301, 400},
	{505, 604, 401, 500},
}

// calculateAQI computes the Air Quality Index
// Based on EPA formula: AQI = ((IHi - ILo) / (BPHi - BPLo)) * (Cp - BPLo) + ILo
// Where:
// - IHi = AQI value corresponding to BPHi
// - ILo = AQI value corresponding to BPLo
// - BPHi = Concentration breakpoint >= Cp
// - BPLo = Concentration breakpoint <= Cp
// - Cp = Pollutant concentration
// Source: https://www.airnow.gov/sites/default/files/2020-05/aqi-technical-assistance-document-sept2018.pdf
func calculateAQI(concentration float64, breakpoints []AQIBreakpoint) int {
	// Truncate to one decimal place as per EPA guidelines
	concentration = math.Floor(concentration*10) / 10

	for _, bp := range breakpoints {
		if concentration >= bp.ConcLow && concentration <= bp.ConcHigh {
			// Apply EPA AQI formula
			aqi := ((float64(bp.AQIHigh-bp.AQILow) / (bp.ConcHigh - bp.ConcLow)) *
				(concentration - bp.ConcLow)) + float64(bp.AQILow)
			return int(math.Round(aqi))
		}
	}

	// If concentration exceeds all breakpoints, return 500+ (hazardous)
	return 500
}

// computeAQI calculates AQI from PM2.5 and PM10 values
// Returns the higher of the two AQI values as per EPA guidelines
func computeAQI(pm25, pm10 float64) int {
	aqiPM25 := calculateAQI(pm25, pm25Breakpoints)
	aqiPM10 := calculateAQI(pm10, pm10Breakpoints)

	// Return the maximum AQI value
	if aqiPM25 > aqiPM10 {
		return aqiPM25
	}
	return aqiPM10
}

func main() {
	// MQTT configuration
	broker := "tcp://192.168.2.71:1883"
	inputTopic := "airgradient/readings/d83bda1d7660"
	outputTopic := "aqi"
	clientID := "aqi-calculator"

	// Configure MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetDefaultPublishHandler(messageHandler)
	opts.SetConnectionLostHandler(connectionLostHandler)

	// Create MQTT client
	client := mqtt.NewClient(opts)

	// Connect to MQTT broker
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	log.Printf("Connected to MQTT broker at %s", broker)

	// Subscribe to input topic
	if token := client.Subscribe(inputTopic, 1, func(client mqtt.Client, msg mqtt.Message) {
		handleMessage(client, msg, outputTopic)
	}); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to subscribe to topic %s: %v", inputTopic, token.Error())
	}

	log.Printf("Subscribed to topic: %s", inputTopic)
	log.Printf("Publishing AQI data to topic: %s", outputTopic)

	// Wait for interrupt signal to gracefully shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	// Unsubscribe and disconnect
	client.Unsubscribe(inputTopic)
	client.Disconnect(250)

	log.Println("Shutdown complete")
}

func messageHandler(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message on topic %s: %s", msg.Topic(), msg.Payload())
}

func connectionLostHandler(client mqtt.Client, err error) {
	log.Printf("Connection lost: %v", err)
}

func handleMessage(client mqtt.Client, msg mqtt.Message, outputTopic string) {
	log.Printf("Processing message from topic: %s", msg.Topic())

	// Parse JSON message
	var reading SensorReading
	if err := json.Unmarshal(msg.Payload(), &reading); err != nil {
		log.Printf("Error parsing JSON: %v", err)
		return
	}

	// Calculate AQI using PM2.5 and PM10 values
	// Using the standard values as they represent ambient conditions
	aqi := computeAQI(reading.PM02Standard, reading.PM10Standard)

	// Create output message with AQI
	aqiReading := AQIReading{
		SensorReading: reading,
		AQI:           aqi,
	}

	// Marshal to JSON
	outputJSON, err := json.Marshal(aqiReading)
	if err != nil {
		log.Printf("Error marshaling output JSON: %v", err)
		return
	}

	// Publish to output topic
	token := client.Publish(outputTopic, 1, false, outputJSON)
	token.Wait()

	if token.Error() != nil {
		log.Printf("Error publishing to topic %s: %v", outputTopic, token.Error())
	} else {
		log.Printf("Published AQI=%d to topic %s", aqi, outputTopic)
	}
}
