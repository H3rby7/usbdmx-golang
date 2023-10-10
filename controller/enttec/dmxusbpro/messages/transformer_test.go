package messages

import (
	"testing"
)

func TestByteToBools(t *testing.T) {
	res := byteToBools(128)
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
			input += 128
		}
		if one {
			input += 64
		}
		if two {
			input += 32
		}
		if three {
			input += 16
		}
		if four {
			input += 8
		}
		if five {
			input += 4
		}
		if six {
			input += 2
		}
		if seven {
			input += 1
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

// 1000 0000 => 128
func TestToChangeSetChannelZeroChanged(t *testing.T) {
	input := EnttecDMXUSBProApplicationMessage{
		label:   LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET,
		payload: []byte{0, 128, 0, 0, 0, 0, 69},
	}
	result, err := ToChangeSet(input)
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
	if result[0] != 69 {
		t.Errorf("expected channel[%d] to be %d, but was %d", 0, 69, result[0])
	}
}

// 0100 0000 => 64
func TestToChangeSetChannelOneChanged(t *testing.T) {
	input := EnttecDMXUSBProApplicationMessage{
		label:   LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET,
		payload: []byte{0, 64, 0, 0, 0, 0, 96},
	}
	result, err := ToChangeSet(input)
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
	if result[1] != 96 {
		t.Errorf("expected channel[%d] to be %d, but was %d", 1, 96, result[1])
	}
}
