package plc

import (
	"encoding/hex"
	"fmt"
	"math"

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
func ReadData(deviceType string, deviceNumber uint16, numberRegisters uint16) (interface{}, error) {
	if msp == nil {
		return nil, fmt.Errorf("MSP client not initialized")
	}
	// Read data from the PLC
	data, err := msp.client.Read(deviceType, int64(deviceNumber), int64(numberRegisters))
	if err != nil {
		return nil, err
	}

	var value interface{}
	switch numberRegisters {
	case 1: // 16-bit device
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
			fmt.Errorf("error decoding hexadecimal string: %d", err)
		}

		// Convert the bytes to a string
		value = string(hexBytes)
	default:
		// Invalid number of registers
		return nil, fmt.Errorf("invalid number of registers: %d", numberRegisters)
	}

	return value, nil
}