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
