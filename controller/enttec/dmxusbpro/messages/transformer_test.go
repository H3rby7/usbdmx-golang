package messages

import (
	"testing"
)

func TestByteToBools(t *testing.T) {
	res := byteToBools(1)
	if !res[0] {
		t.Errorf("expected res[%d] to be true", 0)
	}
	for i := 1; i < 8; i++ {
		if res[i] {
			t.Errorf("expected res[%d] to be false", i)
		}
	}
}

func FuzzTestByteToBools(f *testing.F) {
	f.Add(true, false, false, false, false, false, false, false)
	f.Add(true, true, false, false, false, false, false, false)
	f.Add(true, false, false, false, true, false, true, false)
	f.Add(false, true, false, false, true, false, false, true)
	f.Fuzz(func(t *testing.T, zero bool, one bool, two bool, three bool, four bool, five bool, six bool, seven bool) {
		// *** SETUP ***
		expected := []bool{zero, one, two, three, four, five, six, seven}
		// create an Input LSB first
		input := byte(0)
		if zero {
			input += 1
		}
		if one {
			input += 2
		}
		if two {
			input += 4
		}
		if three {
			input += 8
		}
		if four {
			input += 16
		}
		if five {
			input += 32
		}
		if six {
			input += 64
		}
		if seven {
			input += 128
		}
		// *** TRANSFORM ***
		res := byteToBools(input)
		// *** ASSERT ***
		for i := 0; i < len(expected); i++ {
			if res[i] != expected[i] {
				t.Errorf("expected res[%d] to be '%v' but was %v", i, expected[i], res[i])
			}
		}
	})
}

// 0000 0001 => 1
func TestToChangeSetChannelZeroChanged(t *testing.T) {
	input := EnttecDMXUSBProApplicationMessage{
		label:   LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET,
		payload: []byte{0, 1, 0, 0, 0, 0, 69},
	}
	result, err := ToChangeSet(input)
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
	if result[0] != 69 {
		t.Errorf("expected channel[%d] to be %d, but was %d", 0, 69, result[0])
	}
}

// 0000 0010 => 2
func TestToChangeSetChannelOneChanged(t *testing.T) {
	input := EnttecDMXUSBProApplicationMessage{
		label:   LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET,
		payload: []byte{0, 2, 0, 0, 0, 0, 96},
	}
	result, err := ToChangeSet(input)
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
	if result[1] != 96 {
		t.Errorf("expected channel[%d] to be %d, but was %d", 1, 96, result[1])
	}
}

// 0000 0000 => 0
// 0000 0001 => 1
func TestToChangeSetChannelEightChanged(t *testing.T) {
	input := EnttecDMXUSBProApplicationMessage{
		label:   LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET,
		payload: []byte{0, 0, 1, 0, 0, 0, 161},
	}
	result, err := ToChangeSet(input)
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
	if result[8] != 161 {
		t.Errorf("expected channel[%d] to be %d, but was %d", 8, 161, result[8])
	}
}

// 0001 1000 => 24
// 0100 0011 => 67
func TestToChangeSetMultipleChannelsChanged(t *testing.T) {
	input := EnttecDMXUSBProApplicationMessage{
		label:   LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET,
		payload: []byte{0, 24, 67, 0, 0, 0, 13, 14, 18, 19, 114},
	}
	result, err := ToChangeSet(input)
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
	if result[3] != 13 {
		t.Errorf("expected channel[%d] to be %d, but was %d", 3, 13, result[3])
	}
	if result[4] != 14 {
		t.Errorf("expected channel[%d] to be %d, but was %d", 4, 14, result[4])
	}
	if result[8] != 18 {
		t.Errorf("expected channel[%d] to be %d, but was %d", 8, 18, result[8])
	}
	if result[9] != 19 {
		t.Errorf("expected channel[%d] to be %d, but was %d", 9, 19, result[9])
	}
	if result[14] != 114 {
		t.Errorf("expected channel[%d] to be %d, but was %d", 14, 114, result[14])
	}
}

// Last byte of the changed bit array is 1000 0000 => array bit 39
func TestToChangeSetBit39Changed(t *testing.T) {
	startChangedByteNumber := 2
	input := EnttecDMXUSBProApplicationMessage{
		label:   LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET,
		payload: []byte{byte(startChangedByteNumber), 0, 0, 0, 0, 0x80, 42},
	}
	result, err := ToChangeSet(input)
	if err != nil {
		t.Fatalf("expected no error, but got %v", err)
	}
	slot := startChangedByteNumber*8 + 39
	if len(result) != 1 || result[slot] != 42 {
		t.Errorf("expected changeset to be map[%d:42], but was %v", slot, result)
	}
}

// All 40 bits set, with 40 data bytes
func TestToChangeSetAllBitsChanged(t *testing.T) {
	payload := []byte{1, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	for i := 0; i < 40; i++ {
		payload = append(payload, byte(100+i))
	}
	result, err := ToChangeSet(EnttecDMXUSBProApplicationMessage{label: LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET, payload: payload})
	if err != nil {
		t.Fatalf("expected no error, but got %v", err)
	}
	if len(result) != 40 {
		t.Errorf("expected 40 changes, but got %d: %v", len(result), result)
	}
	for i := 0; i < 40; i++ {
		if result[8+i] != byte(100+i) {
			t.Errorf("expected slot %d to be %d, but was %d", 8+i, 100+i, result[8+i])
		}
	}
}

// 0000 0011 => 2 set bits, but only 1 data byte
func TestToChangeSetTooFewDataBytes(t *testing.T) {
	input := EnttecDMXUSBProApplicationMessage{
		label:   LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET,
		payload: []byte{0, 3, 0, 0, 0, 0, 69},
	}
	var result map[int]byte
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected an error, but got a panic: %v", r)
			}
		}()
		result, err = ToChangeSet(input)
	}()
	if err == nil {
		t.Fatalf("expected an error, but got changeset %v", result)
	}
	if result != nil {
		t.Errorf("expected no changeset, but got %v", result)
	}
}
