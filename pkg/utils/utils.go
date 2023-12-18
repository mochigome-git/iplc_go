package utils

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// Define the device struct with the address field
type Device struct {
	DeviceType      string
	DeviceNumber    string
	NumberRegisters uint16
}

// ParseDeviceAddresses parses the device addresses from the environment variable.
func ParseDeviceAddresses(envVar string, logger *log.Logger) ([]Device, error) {
	deviceStrings := strings.Split(envVar, ",")
	if len(deviceStrings)%3 != 0 {
		logger.Printf("Invalid DEVICES environment variable: %s", envVar)
		return nil, nil
	}
	var devices []Device
	for i := 0; i < len(deviceStrings); i += 3 {
		deviceNumber := deviceStrings[i+1]

		numberRegisters, err := strconv.ParseUint(deviceStrings[i+2], 10, 16)
		if err != nil {
			logger.Fatalf("Error parsing number of registers: %v", err)
		}
		devices = append(devices, Device{
			DeviceType:      deviceStrings[i],
			DeviceNumber:    fmt.Sprint(deviceNumber),
			NumberRegisters: uint16(numberRegisters),
		})
	}
	if len(devices) == 0 {
		logger.Fatalf("No devices found in DEVICES environment variable: %s", envVar)
	}
	logger.Printf("Loaded %d device(s) from DEVICES environment variable", len(devices))
	return devices, nil
}
