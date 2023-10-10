package dmxusbpro

import (
	"testing"
)

// Test extracting from bytes
func TestExtractorSuccessNoPayload(t *testing.T) {
	runAndAssertValid(t, 5, []byte{}, []byte{0x7E, 5, 0, 0, 0xE7})
}

func runAndAssertValid(t *testing.T, label byte, payload []byte, input []byte) {
	msg, err := Extract(input)
	// Should not error
	if err != nil {
		t.Errorf("expected a valid message, but got error %v", err)
	}
	// check Label
	if msg.label != label {
		t.Errorf("expected label to be %X, but was %X", label, msg.label)
	}
	// Compare Payloads
	for i := 0; i < len(payload); i++ {
		if msg.payload[i] != payload[i] {
			t.Errorf("expected payload byte[%d] to be '%X' but was %X", i, payload[i], msg.payload[i])
		}
	}
}
