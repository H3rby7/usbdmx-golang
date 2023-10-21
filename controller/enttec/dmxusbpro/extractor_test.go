package dmxusbpro

import (
	"testing"
)

// raw data is a valid payloadless message, no more, no less
func TestExtractorSuccessNoPaddingNoPayload(t *testing.T) {
	label := byte(1)
	runAndAssertValid(t, label, []byte{}, []byte{0x7E, label, 0, 0, 0xE7})
}

// raw data has payload.
func TestExtractorSuccessWithPayload(t *testing.T) {
	label := byte(2)
	runAndAssertValid(t, label, []byte{69, 66, 96}, []byte{0x7E, label, 3, 0, 69, 66, 96, 0xE7})
}

// raw data is padded with rubbish
func TestExtractorSuccessPadded(t *testing.T) {
	label := byte(3)
	runAndAssertValid(t, label, []byte{69, 66, 96}, []byte{0x11, 0x7E, label, 3, 0, 69, 66, 96, 0xE7, 0xEE})
}

// raw data is padded and has multiple potential starts
func TestExtractorSuccessMultipleStarts(t *testing.T) {
	label := byte(4)
	runAndAssertValid(t, label, []byte{69, 66, 96}, []byte{0x7E, 0x11, 0x7E, label, 3, 0, 69, 66, 96, 0xE7, 0xEE})
}

// raw data is padded and has multiple potential starts
func TestExtractorSuccessMultipleEnds(t *testing.T) {
	label := byte(5)
	runAndAssertValid(t, label, []byte{69, 66, 96}, []byte{0x11, 0xE7, 0x7E, label, 3, 0, 69, 66, 96, 0xE7, 0xEE, 0xE7})
}

// raw data is padded and has potential starts inside the data
func TestExtractorSuccessWithStartsInData(t *testing.T) {
	label := byte(6)
	runAndAssertValid(t, label, []byte{69, 0x7E, 96}, []byte{0x11, 0x7E, label, 3, 0, 69, 0x7E, 96, 0xE7, 0xEE})
}

// raw data is padded and has potential ends inside the data
func TestExtractorSuccessWithEndsInData(t *testing.T) {
	label := byte(7)
	runAndAssertValid(t, label, []byte{69, 0xE7, 96}, []byte{0x11, 0x7E, label, 3, 0, 69, 0xE7, 96, 0xE7, 0xEE})
}

// raw data is padded and the content would also be a valid message
// we expect the outer message to be returned
func TestExtractorSuccessWithDataInData(t *testing.T) {
	label := byte(9)
	runAndAssertValid(
		t,
		label,
		[]byte{0x7E, 5, 3, 0, 69, 66, 96, 0xE7},
		[]byte{0x7E, label, 8, 0, 0x7E, 5, 3, 0, 69, 66, 96, 0xE7, 0xE7},
	)
}

// raw data is padded and the content is the beginning of another valid message
// we expect the first message to be returned
func TestExtractorSuccessWithDataBeginningInData(t *testing.T) {
	label := byte(11)
	runAndAssertValid(
		t,
		label,
		[]byte{69, 0x7E, 5, 2, 0},
		[]byte{0x7E, label, 5, 0, 69, 0x7E, 5, 2, 0, 0xE7, 0x11, 0xE7},
	)
}

func TestExtractorFailsEmptySlice(t *testing.T) {
	if msg, err := Extract([]byte{}); err == nil {
		t.Errorf("expected an error, however got a message: %v", msg)
	}
}

func TestExtractorFailsNoStart(t *testing.T) {
	if msg, err := Extract([]byte{0x00, 1, 0, 0, 0xE7}); err == nil {
		t.Errorf("expected an error, however got a message: %v", msg)
	}
}

func TestExtractorFailsNoEnd(t *testing.T) {
	if msg, err := Extract([]byte{0x7E, 1, 0, 0, 0x00}); err == nil {
		t.Errorf("expected an error, however got a message: %v", msg)
	}
}

func TestExtractorFailsEndBeforeStart(t *testing.T) {
	if msg, err := Extract([]byte{0xE7, 1, 0, 0, 0x7E}); err == nil {
		t.Errorf("expected an error, however got a message: %v", msg)
	}
}

func runAndAssertValid(t *testing.T, label byte, payload []byte, input []byte) {
	msgs, err := Extract(input)
	// Should not error
	if err != nil {
		t.Errorf("expected a valid message, but got error %v", err)
	}
	msg := msgs[0]
	// check Label
	if msg.GetLabel() != label {
		t.Errorf("expected label to be %X, but was %X", label, msg.GetLabel())
	}
	// Compare Payloads
	for i := 0; i < len(payload); i++ {
		if msg.GetPayload()[i] != payload[i] {
			t.Errorf("expected payload byte[%d] to be '%X' but was %X", i, payload[i], msg.GetPayload()[i])
		}
	}
}
