package dmxusbpro

import (
	"testing"
)

// Test converting to bytes without payload
func TestToBytesNoPayload(t *testing.T) {
	input := EnttecDMXUSBProApplicationMessage{label: 1, payload: []byte{}}
	result := input.ToBytes()
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
		t.Errorf("expected byte[3] (LSB of data_length) to be '0' but was %d", result[3])
	}
	if len(result) != 5 {
		t.Errorf("expected size to be '5' but was %d", len(result))
	}
}

// Test converting to bytes with 1 byte payload
func TestToBytesSmallPayload(t *testing.T) {
	payload := byte(0x69)
	input := EnttecDMXUSBProApplicationMessage{label: 6, payload: []byte{payload}}
	result := input.ToBytes()
	lastIndex := len(result) - 1
	if result[0] != 0x7E {
		t.Errorf("expected first byte to be '7E' but was %X", result[0])
	}
	if result[lastIndex] != 0xE7 {
		t.Errorf("expected last byte to be 'E7' but was %X", result[lastIndex])
	}
	if result[1] != 6 {
		t.Errorf("expected byte[1] (the label) to be '6' but was %d", result[1])
	}
	if result[2] != 1 {
		t.Errorf("expected byte[2] (LSB of data_length) to be '1' but was %d", result[2])
	}
	if result[3] != 0 {
		t.Errorf("expected byte[3] (LSB of data_length) to be '0' but was %d", result[3])
	}
	if result[4] != payload {
		t.Errorf("expected byte[4] to be our payload '%X' but was %X", payload, result[4])
	}
	if len(result) != 6 {
		t.Errorf("expected size to be '6' but was %d", len(result))
	}
}

// Test converting to bytes with DMX payload
func TestToBytesBigPayload(t *testing.T) {
	payload := []byte{0x00, 0x01, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x80, 0xFF}
	input := EnttecDMXUSBProApplicationMessage{label: 6, payload: payload}
	result := input.ToBytes()
	lastIndex := len(result) - 1
	if result[0] != 0x7E {
		t.Errorf("expected first byte to be '7E' but was %X", result[0])
	}
	if result[lastIndex] != 0xE7 {
		t.Errorf("expected last byte to be 'E7' but was %X", result[lastIndex])
	}
	if result[1] != 6 {
		t.Errorf("expected byte[1] (the label) to be '6' but was %d", result[1])
	}
	if result[2] != 10 {
		t.Errorf("expected byte[2] (LSB of data_length) to be '10' but was %d", result[2])
	}
	if result[3] != 0 {
		t.Errorf("expected byte[3] (LSB of data_length) to be '0' but was %d", result[3])
	}
	if len(result) != 15 {
		t.Errorf("expected size to be '15' but was %d", len(result))
	}
	// Compare Payloads
	resPayload := result[4:lastIndex]
	for i := 0; i < len(payload); i++ {
		if resPayload[i] != payload[i] {
			t.Errorf("expected byte[%d] (payload) to be '%X' but was %X", i, payload[i], resPayload[i])
		}
	}
}

// e.G.: go test -v --fuzz=Fuzz .\controller\enttec\dmxusbpro
func FuzzTestToBytes(f *testing.F) {
	testCases := [][]byte{
		{},
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
		input := EnttecDMXUSBProApplicationMessage{label: 6, payload: payload}
		result := input.ToBytes()
		lastIndex := len(result) - 1
		if result[0] != 0x7E {
			t.Errorf("expected first byte to be '7E' but was %X", result[0])
		}
		if result[lastIndex] != 0xE7 {
			t.Errorf("expected last byte to be 'E7' but was %X", result[lastIndex])
		}
		if result[1] != 6 {
			t.Errorf("expected byte[1] (the label) to be '6' but was %d", result[1])
		}
		if len(payload) < 256 {
			if result[3] != 0 {
				t.Errorf("expected byte[3] (LSB of data_length) to be '0' but was %d", result[3])
			}
		}
		expectedLen := 5 + len(payload)
		if len(result) != expectedLen {
			t.Errorf("expected size to be '%d' but was %d", expectedLen, len(result))
		}
		// Compare Payloads
		resPayload := result[4:lastIndex]
		for i := 0; i < len(payload); i++ {
			if resPayload[i] != payload[i] {
				t.Errorf("expected byte[%d] (payload) to be '%X' but was %X", i, payload[i], resPayload[i])
			}
		}
	})
}
