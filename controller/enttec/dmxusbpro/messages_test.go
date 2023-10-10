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
	// 7 sample payloads
	testPayloads := [][]byte{
		{},
		{0x69},
		{0x00, 0x01, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x80, 0xFF},
		{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		{255, 254, 253, 252, 251, 250, 249},
		{0xE7, 0x7E},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	// 7 test labels
	testLabels := []byte{1, 2, 3, 6, 9, 10, 11}
	for i := 0; i < 7; i++ {
		f.Add(testPayloads[i], testLabels[i])
	}
	f.Fuzz(func(t *testing.T, payload []byte, label byte) {
		// ************* SETUP *********************
		input := EnttecDMXUSBProApplicationMessage{label: label, payload: payload}
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
		if result[1] != label {
			t.Errorf("expected byte[1] (the label) to be '%d' but was %d", label, result[1])
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

// Test converting from bytes without payload
func TestFromBytesNoPayload(t *testing.T) {
	inputLabel := byte(9)
	input := []byte{0x7E, byte(inputLabel), 0, 0, 0xE7}
	result, err := FromBytes(input)
	if err != nil {
		t.Errorf("did not expect error, but got '%v'", err)
	}
	if result.label != inputLabel {
		t.Errorf("expected result label to be %X, but got %X", inputLabel, result.label)
	}
	resultPayloadLength := len(result.payload)
	if resultPayloadLength != 0 {
		t.Errorf("expected payload length to be 0, instead got %d", resultPayloadLength)
	}
}

func TestFromBytesBadMsgStart(t *testing.T) {
	for i := 0; i < 256; i++ {
		input := []byte{byte(i), 1, 0, 0, 0xE7}
		_, err := FromBytes(input)
		if i == 0x7E {
			// In this case it works
			continue
		}
		if err == nil {
			t.Errorf("expected error, because messages must start with 0x7E")
		}
	}
}

func TestFromBytesBadMsgEnd(t *testing.T) {
	for i := 0; i < 256; i++ {
		input := []byte{0x7E, 1, 0, 0, byte(i)}
		_, err := FromBytes(input)
		if i == 0xE7 {
			// In this case it works
			continue
		}
		if err == nil {
			t.Errorf("expected error, because messages must end with 0x7E")
		}
	}
}

func TestFromBytesMsgTooSmall(t *testing.T) {
	input := []byte{0x7E, 0xE7}
	_, err := FromBytes(input)
	if err == nil {
		t.Errorf("expected error, because messages is too small (packet wrapper is already 5 bytes)")
	}
}

func TestFromBytesLabelTooSmall(t *testing.T) {
	input := []byte{0x7E, 0, 0, 0, 0xE7}
	_, err := FromBytes(input)
	if err == nil {
		t.Errorf("expected error, because message label must be bigger than '0'")
	}
}

func TestFromBytesLabelTooBig(t *testing.T) {
	for i := 12; i < 256; i++ {
		input := []byte{0x7E, byte(i), 0, 0, 0xE7}
		_, err := FromBytes(input)
		if err == nil {
			t.Errorf("expected error, because message label must be smaller than '12'")
		}
	}
}

func TestFromBytesLimitError(t *testing.T) {
	input := make([]byte, 601)
	_, err := FromBytes(input)
	if err == nil {
		t.Errorf("expected error as payload exceeded limits")
	}
}

func TestFromBytesIndicatedLengthMismatchLSB(t *testing.T) {
	input := []byte{0x7E, 1, 1, 0, 0xE7}
	_, err := FromBytes(input)
	if err == nil {
		t.Errorf("expected error, because indicated datalength (1) mismatches actual length (0)")
	}
}

func TestFromBytesIndicatedLengthMismatchMSB(t *testing.T) {
	input := []byte{0x7E, 1, 0, 1, 0xE7}
	_, err := FromBytes(input)
	if err == nil {
		t.Errorf("expected error, because indicated datalength (256) mismatches actual length (0)")
	}
}

func FuzzTestFromBytes(f *testing.F) {
	testPayloads := [][]byte{
		{},
		{69},
		{96, 69},
		{66, 69, 96},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}
	for _, v := range testPayloads {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, payload []byte) {
		size := len(payload)
		if size > 600 {
			// As payloads > 600 will result in validation errors
			return
		}
		// ************* SETUP *********************
		input := make([]byte, size+5)
		input[0] = 0x7E                   // START
		input[1] = 9                      // LABEL
		input[2] = byte(size & 0xFF)      // LSB
		input[3] = byte(size >> 8 & 0xFF) // MSB
		copy(input[4:size+4], payload)    // PAYLOAD
		input[size+4] = 0xE7              // END
		// ************* ACTION *********************
		msg, err := FromBytes(input)
		// ************* ASSERTIONS *********************
		// should work
		if err != nil {
			t.Errorf("should not error, but got '%v'", err)
		}
		// Compare Payloads
		for i := 0; i < len(payload); i++ {
			if msg.payload[i] != payload[i] {
				t.Errorf("expected payload byte[%d] to be '%X' but was %X", i, payload[i], msg.payload[i])
			}
		}
	})
}
