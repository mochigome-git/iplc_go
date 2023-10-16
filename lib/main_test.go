package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"nk2-PLCcapture-go/pkg/config"
	"nk2-PLCcapture-go/pkg/mqtt"
	"nk2-PLCcapture-go/pkg/plc"
	"nk2-PLCcapture-go/pkg/utils"

	jsoniter "github.com/json-iterator/go"
)

func init() {
	config.LoadEnv(".env.local")
	mqttHost = os.Getenv("MQTT_HOST")
	plcHost = os.Getenv("PLC_HOST")
	plcPort = config.GetEnvAsInt("PLC_PORT", 5011)
	devices16 = os.Getenv("DEVICES_16bit")
	devices32 = os.Getenv("DEVICES_32bit")
	//devices2 = os.Getenv("DEVICES_2bit")
	devicesAscii = os.Getenv("DEVICES_ASCII")
}

func TestConcurrentWorkers(t *testing.T) {

	// Create a logger to use for logging messages
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Connect to the MQTT server
	mqttclient := mqtt.NewMQTTClient(mqttHost, logger)
	defer mqttclient.Disconnect(250)

	// Parse the device addresses for 16-bit devices
	devices16Parsed, err := utils.ParseDeviceAddresses(devices16, logger)
	if err != nil {
		logger.Fatalf("Error parsing device addresses: %v", err)
	}

	// Parse the device addresses for 32-bit devices
	devices32Parsed, err := utils.ParseDeviceAddresses(devices32, logger)
	if err != nil {
		logger.Fatalf("Error parsing device addresses: %v", err)
	}

	// Parse the device addresses for text devices
	devicesAsciiParsed, err := utils.ParseDeviceAddresses(devicesAscii, logger)
	if err != nil {
		logger.Fatalf("Error parsing device addresses: %v", err)
	}

	// Parse the device addresses for text devices
	//devices2Parsed, err := utils.ParseDeviceAddresses(devices2, logger)
	//if err != nil {
	//	logger.Fatalf("Error parsing device addresses: %v", err)
	//}

	// Combine the 16-bit and 32-bit devices into a single slice
	devices := append(devices16Parsed, devicesAsciiParsed...)
	devices = append(devices, devices32Parsed...)

	// Initialize the MSP client
	err = plc.InitMSPClient(plcHost, plcPort)
	if err != nil {
		logger.Fatalf("Failed to initialize MSP client: %v", err)
	} else {
		log.Printf("Start collecting data from %s", plcHost)
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
					topic := "test" + message["address"].(string)
					mqtt.PublishMessage(mqttclient, topic, string(messageJSON), logger)
				}
			}()
		}

		// Run the main loop in a separate goroutine
		go func() {
			for {

				// Initialize the context with a timeout of 20 seconds
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				// Read data from devices and send it to dataCh
				for _, device := range devices {
					select {
					case <-ctx.Done():
						logger.Printf("%s timed out. error: %s\n", device.DeviceType+strconv.Itoa(int(device.DeviceNumber)), ctx.Err())
						logger.Println("Program terminated by os.Exit")
						os.Exit(1)
						return
					default:
						value, err := plc.ReadData(device.DeviceType, device.DeviceNumber, device.NumberRegisters)
						if err != nil {
							logger.Printf("Error reading data from PLC for device %s: %s", device.DeviceType+strconv.Itoa(int(device.DeviceNumber)), err)
							continue // Skip this device and move to the next
						}
						message := map[string]interface{}{
							"address": device.DeviceType + strconv.Itoa(int(device.DeviceNumber)),
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
