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

// Test converting to bytes with DMX payload
func TestToBytesDMXPayload(t *testing.T) {
	input := EnttecDMXUSBProApplicationMessage{label: 6, payload: []byte{0x69}}
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
	if len(result) != 6 {
		t.Errorf("expected size to be '6' but was %d", len(result))
	}
}
