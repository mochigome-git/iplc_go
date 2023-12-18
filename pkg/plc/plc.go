package plc

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"

	"nk2-PLCcapture-go/pkg/mcp"
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
func ReadData(ctx context.Context, deviceType string, deviceNumber string, numberRegisters uint16) (interface{}, error) {
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
		data, err := msp.client.Read(deviceType, deviceNumberInt64, int64(numberRegisters))
		if err != nil {
			errCh <- err
			return
		}

		var value interface{}
		switch numberRegisters {
		case 1: // 16-bit device 0 to 65535
			// Parse 16-bit data
			registerBinary, _ := mcp.NewParser().Do(data)
			data = registerBinary.Payload
			var val uint16
			for i := 0; i < len(data); i++ {
				val |= uint16(data[i]) << uint(8*i)
			}
			value = val
		case 2: // 32-bit device
			// Parse 32-bit data
			var val uint32
			registerBinary, _ := mcp.NewParser().Do(data)
			data = registerBinary.Payload
			for i := 0; i < len(data); i++ {
				val |= uint32(data[i]) << uint(8*i)
			}
			floatValue := math.Float32frombits(val)
			floatString := fmt.Sprintf("%.6f", floatValue)
			firstSixDigits := ""
			numDigits := 0
			for _, c := range floatString {
				if c == '-' || c == '.' {
					// Include minus sign and decimal point
					firstSixDigits += string(c)
				} else if numDigits < 6 {
					// Only include the first 6 digits
					firstSixDigits += string(c)
					numDigits++
				}
			}
			value = firstSixDigits
		case 3: // 2-bit device
			// Parse 2-bit data
			registerBinary, _ := mcp.NewParser().Do(data)
			data = registerBinary.Payload

			var val uint8
			if len(data) >= 1 {
				// Extract the 2-bit value from the 8-bit data
				val = uint8(data[0] & 0x01)
			}
			value = val
		case 4: // ASCII hex device
			// Parse 16-bit data
			registerBinary, _ := mcp.NewParser().Do(data)
			data = registerBinary.Payload
			var val uint16
			for i := 0; i < len(data); i++ {
				val |= uint16(data[i]) << uint(8*i)
			}
			text := fmt.Sprintf("%X", val)

			// Decode the hexadecimal string into bytes
			hexBytes, err := hex.DecodeString(text)
			if err != nil {
				fmt.Errorf("error decoding hexadecimal string: %s", err)
			}

			// Convert the bytes to a string
			value = string(hexBytes)
		case 5: // 16-bit device -37268 to 32767
			// Parse 16-bit data with negative value
			registerBinary, _ := mcp.NewParser().Do(data)
			data = registerBinary.Payload
			var val int16
			for i := 0; i < len(data); i++ {
				val |= int16(data[i]) << int(8*i)
			}
			value = val
		default:
			// Invalid number of registers
			errCh <- fmt.Errorf("invalid number of registers: %d", numberRegisters)
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
