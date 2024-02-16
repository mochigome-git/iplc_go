package plc

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"

	"inkjet-PLCcapture-go/pkg/mcp"
)

type mspClient struct {
	client mcp.Client
}

var msp *mspClient

func InitMSPClient(plcHost string, plcPort int) error {
	if msp != nil {
		return nil
	}
	// Connect to the PLC with MC protocol
	client, err := mcp.New3EClient(plcHost, plcPort, mcp.NewLocalStation())
	if err != nil {
		return err
	}
	msp = &mspClient{client: client}
	return nil
}

// ReadData reads data from the PLC for the specified device.
func ReadData(ctx context.Context, deviceType string, deviceNumber string, numberRegisters uint16, fx bool) (interface{}, error) {
	if msp == nil {
		return nil, fmt.Errorf("MSP client not initialized")
	}

	// Create a channel to receive the result or context cancellation
	resultCh := make(chan interface{})
	errCh := make(chan error)

	deviceNumberInt64, err := strconv.ParseInt(deviceNumber, 10, 64)
	if err != nil || deviceType == "Y" {

		// Convert the offset string to an integer
		deviceNumberInt64, err = strconv.ParseInt(deviceNumber, 16, 64)
		if err != nil {
			return nil, err
		}

		//return nil, err
	}

	// Start a goroutine to perform the data reading
	go func() {
		// Read data from the PLC
		data, err := msp.client.Read(deviceType, deviceNumberInt64, int64(numberRegisters), fx)
		if err != nil {
			errCh <- err
			return
		}

		// Parse the data based on the number of registers and fx condition
		value, err := ParseData(data, int(numberRegisters), fx)
		if err != nil {
			errCh <- err
			return
		}

		// Send the result on the channel
		resultCh <- value
	}()

	select {
	case <-ctx.Done():
		// Context is canceled before the operation completes
		return nil, ctx.Err()
	case err := <-errCh:
		// Error occurred during the data reading operation
		return nil, err
	case value := <-resultCh:
		// Data reading operation completed successfully
		return value, nil
	}
}

// ParseData parses the data based on the specified number of registers and fx condition
func ParseData(data []byte, numberRegisters int, fx bool) (interface{}, error) {
	registerBinary, _ := mcp.NewParser().Do(data)
	if fx {
		registerBinary, _ = mcp.NewParser().DoFx(data)
	}
	data = registerBinary.Payload

	switch numberRegisters {
	case 1, 5: // 16-bit devices
		var val uint16
		for i := 0; i < len(data); i++ {
			val |= uint16(data[i]) << uint(8*i)
		}
		if numberRegisters == 5 {
			return int16(val), nil // For case 5, return as int16
		}
		return val, nil
	case 2: // 32-bit device
		var val uint32
		for i := 0; i < len(data); i++ {
			val |= uint32(data[i]) << uint(8*i)
		}
		floatValue := math.Float32frombits(val)
		floatString := fmt.Sprintf("%.6f", floatValue)
		firstSixDigits := ""
		numDigits := 0
		for _, c := range floatString {
			if c == '-' || c == '.' {
				firstSixDigits += string(c)
			} else if numDigits < 6 {
				firstSixDigits += string(c)
				numDigits++
			}
		}
		return firstSixDigits, nil
	case 3: // 2-bit device
		var val uint8
		if len(data) >= 1 {
			val = uint8(data[0] & 0x01)
		}
		return val, nil
	case 4: // ASCII hex device
		var val uint16
		for i := 0; i < len(data); i++ {
			val |= uint16(data[i]) << uint(8*i)
		}
		text := fmt.Sprintf("%X", val)
		hexBytes, err := hex.DecodeString(text)
		if err != nil {
			return nil, fmt.Errorf("error decoding hexadecimal string: %s", err)
		}
		return string(hexBytes), nil
	case 6:
		var val uint16
		for i := 0; i < len(data); i++ {
			val |= uint16(data[i]/10) << uint(8*i)
		}

		hexadecimalString := fmt.Sprintf("%X", val)
		return hexadecimalString, nil
	default:
		return nil, fmt.Errorf("invalid number of registers: %d", numberRegisters)
	}
}
