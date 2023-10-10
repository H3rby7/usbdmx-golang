package dmxusbpro

import (
	"testing"
)

// Test converting to bytes without payload
func TestToBytesNoPayload(t *testing.T) {
	input := EnttecDMXUSBProApplicationMessage{label: 1, payload: []byte{}}
	result, _ := input.ToBytes()
	lastIndex := len(result) - 1
	if result[0] != 0x7E {
		t.Errorf("expected first byte to be '7E' but was %X", result[0])
	}
	if result[lastIndex] != 0xE7 {
		t.Errorf("expected last byte to be 'E7' but was %X", result[lastIndex])
	}
	if result[1] != 1 {
		t.Errorf("expected byte[1] (the label) to be '1' but was %d", result[1])
	}
	if result[2] != 0 {
		t.Errorf("expected byte[2] (LSB of data_length) to be '0' but was %d", result[2])
	}
	if result[3] != 0 {
		t.Errorf("expected byte[3] (MSB of data_length) to be '0' but was %d", result[3])
	}
	if len(result) != 5 {
		t.Errorf("expected size to be '5' but was %d", len(result))
	}
}

// Test converting to bytes with payload exceeding limits expecting error
func TestToBytesLimitError(t *testing.T) {
	payload := make([]byte, 601)
	input := EnttecDMXUSBProApplicationMessage{label: 1, payload: payload}
	_, err := input.ToBytes()
	if err == nil {
		t.Errorf("expected error as payload exceeded limits")
	}
}

// e.G.: go test -v --fuzz=Fuzz .\controller\enttec\dmxusbpro
func FuzzTestToBytes(f *testing.F) {
	testCases := [][]byte{
		{},
		{0x69},
		{0x00, 0x01, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x80, 0xFF},
		{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		{255, 254, 253, 252, 251, 250, 249},
		{0xE7, 0x7E},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	for _, v := range testCases {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, payload []byte) {
		// ************* SETUP *********************
		input := EnttecDMXUSBProApplicationMessage{label: 6, payload: payload}
		inputDataLength := len(payload)
		// ************* ACTION *********************
		result, err := input.ToBytes()
		// ************* ASSERTIONS *********************
		// Check for error case
		if inputDataLength > 600 {
			if err == nil {
				t.Errorf("expected error as payload exceeded limits")
			} else {
				// We have an error (and no data to check)
				return
			}
		}
		lastIndex := len(result) - 1
		// PACKET ASSERTIONS
		// message start byte
		if result[0] != 0x7E {
			t.Errorf("expected first byte to be '7E' but was %X", result[0])
		}
		// message end byte
		if result[lastIndex] != 0xE7 {
			t.Errorf("expected last byte to be 'E7' but was %X", result[lastIndex])
		}
		// check label
		if result[1] != 6 {
			t.Errorf("expected byte[1] (the label) to be '6' but was %d", result[1])
		}
		// data length checks
		resultDataLength := int(result[2]) + 256*int(result[3])
		if resultDataLength != inputDataLength {
			t.Errorf("expected indicated datalength to be '%d' but was %d", inputDataLength, resultDataLength)
		}
		// Verify length of the whole packet
		expectedLen := 5 + inputDataLength
		if len(result) != expectedLen {
			t.Errorf("expected size to be '%d' but was %d", expectedLen, len(result))
		}
		// Compare Payloads
		resPayload := result[4:lastIndex]
		for i := 0; i < inputDataLength; i++ {
			if resPayload[i] != payload[i] {
				t.Errorf("expected byte[%d] (payload) to be '%X' but was %X", i, payload[i], resPayload[i])
			}
		}
	})
}
