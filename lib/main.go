package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"inkjet-PLCcapture-go/pkg/config"
	"inkjet-PLCcapture-go/pkg/mqtt"
	"inkjet-PLCcapture-go/pkg/plc"
	"inkjet-PLCcapture-go/pkg/utils"

	jsoniter "github.com/json-iterator/go"
)

var (
	mqttHost     string
	plcHost      string
	plcPort      int
	devices16    string
	devices32    string
	devices2     string
	devicesAscii string
	mqttTopic    string
	fxStr        string
	fx           bool
)

func init() {
	config.LoadEnv(".env.local")
	mqttHost = os.Getenv("MQTT_HOST")
	plcHost = os.Getenv("PLC_HOST")
	plcPort = config.GetEnvAsInt("PLC_PORT", 5011)
	devices16 = os.Getenv("DEVICES_16bit")
	devices32 = os.Getenv("DEVICES_32bit")
	devices2 = os.Getenv("DEVICES_2bit")
	mqttTopic = os.Getenv("MQTT_TOPIC")
	devicesAscii = os.Getenv("DEVICES_ASCII")
	fxStr = os.Getenv("PLC_MODEL")

}

func main() {

	// Create a logger to use for logging messages
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Connect to the MQTT server
	mqttclient := mqtt.NewMQTTClient(mqttHost, logger)
	defer mqttclient.Disconnect(250)

	// Parse the device addresses for 16-bit devices
	devices16Parsed, _ := ParseAndLogError(devices16, logger)
	devices32Parsed, _ := ParseAndLogError(devices32, logger)
	devices2Parsed, _ := ParseAndLogError(devices2, logger)
	devicesAsciiParsed, _ := ParseAndLogError(devicesAscii, logger)

	// Set fx to false as default
	fx, err := strconv.ParseBool(fxStr)
	if err != nil || fxStr == "fx" {
		fx = (fxStr == "fx") // Set fx to true if fxStr equals "fx"
		// Handle the error, for example, set a default value or log a message
		logger.Println("Error parsing fx:", err)
	}

	// Combine the 2-bit, 16-bit and 32-bit devices into a single slice
	devices := append(devices16Parsed, append(append(devices2Parsed, devices32Parsed...), devicesAsciiParsed...)...)

	// Initialize the MSP client
	err = plc.InitMSPClient(plcHost, plcPort)
	if err != nil {
		logger.Fatalf("Failed to initialize MSP client: %v", err)
	} else {
		logger.Printf("Start collecting data from %s", plcHost)
	}

	for {
		workerCount := 15
		// Use a buffered channel to store the data to be processed
		dataCh := make(chan map[string]interface{}, workerCount) // Buffered channel with capacity equal to the number of workers

		// Start the worker goroutines before reading data from the devices
		// Spawn multiple worker goroutines that read the data from the channel, process it, and send it to MQTT
		var wg sync.WaitGroup
		wg.Add(workerCount)

		for i := 0; i < workerCount; i++ {
			go func() {
				defer wg.Done()
				for message := range dataCh {

					// Convert the message to a JSON string
					messageJSON, err := jsoniter.Marshal(message)
					if err != nil {
						logger.Printf("Error marshaling message to JSON:%s", err)
						continue
					}

					// Publish the message to the MQTT server
					topic := mqttTopic + message["address"].(string)
					mqtt.PublishMessage(mqttclient, topic, string(messageJSON), logger)
				}
			}()
		}

		// Run the main loop in a separate goroutine
		go func() {
			for {

				// Initialize the context with a timeout of 20 seconds
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()

				// Read data from devices and send it to dataCh
				for _, device := range devices {
					select {
					case <-ctx.Done():
						logger.Printf("%s timed out. error: %s\n", device.DeviceType+device.DeviceNumber, ctx.Err())
						logger.Println("Program terminated by os.Exit")
						os.Exit(1)
						return
					default:
						value, err := ReadDataWithContext(ctx, device.DeviceType, device.DeviceNumber, device.NumberRegisters, fx)
						if err != nil {
							logger.Printf("Error reading data from PLC for device %s: %s", device.DeviceType+device.DeviceNumber, err)
							break // Skip this device and move to the next
						}
						message := map[string]interface{}{
							"address": device.DeviceType + device.DeviceNumber,
							"value":   value,
						}
						dataCh <- message
					}
				}

			}
		}()

		<-dataCh

		// dataCh is closed, all workers are done
		wg.Wait()

	}

}

func ReadDataWithContext(ctx context.Context, deviceType string, deviceNumber string, numRegisters uint16, fx bool) (value interface{}, err error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Perform the actual data reading operation
		value, err = plc.ReadData(ctx, deviceType, deviceNumber, numRegisters, fx)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
}

func ParseAndLogError(devices string, logger *log.Logger) ([]utils.Device, error) {
	parsed, err := utils.ParseDeviceAddresses(devices, logger)
	if err != nil {
		logger.Printf("Error parsing device addresses: %v", err)
	}
	return parsed, err
}
