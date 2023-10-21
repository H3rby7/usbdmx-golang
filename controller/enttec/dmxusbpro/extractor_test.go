package dmxusbpro

import (
	"testing"
)

// raw data is a valid payloadless message, no more, no less
func TestExtractorSuccessNoPaddingNoPayload(t *testing.T) {
	label := byte(1)
	runAndAssertValid(
		t,
		label,
		[]byte{},
		[]byte{},
		[]byte{0x7E, label, 0, 0, 0xE7},
	)
}

// raw data has payload.
func TestExtractorSuccessWithPayload(t *testing.T) {
	label := byte(2)
	runAndAssertValid(
		t,
		label,
		[]byte{69, 66, 96},
		[]byte{},
		[]byte{0x7E, label, 3, 0, 69, 66, 96, 0xE7},
	)
}

// raw data is padded with rubbish
func TestExtractorSuccessPadded(t *testing.T) {
	label := byte(3)
	runAndAssertValid(
		t,
		label,
		[]byte{69, 66, 96},
		[]byte{0xEE},
		[]byte{0x11, 0x7E, label, 3, 0, 69, 66, 96, 0xE7, 0xEE},
	)
}

// raw data is padded and has multiple potential starts
func TestExtractorSuccessMultipleStarts(t *testing.T) {
	label := byte(4)
	runAndAssertValid(
		t,
		label,
		[]byte{69, 66, 96},
		[]byte{0xEE},
		[]byte{0x7E, 0x11, 0x7E, label, 3, 0, 69, 66, 96, 0xE7, 0xEE},
	)
}

// raw data is padded and has multiple potential starts
func TestExtractorSuccessMultipleEnds(t *testing.T) {
	label := byte(5)
	runAndAssertValid(
		t,
		label,
		[]byte{69, 66, 96},
		[]byte{0xEE, 0xE7},
		[]byte{0x11, 0xE7, 0x7E, label, 3, 0, 69, 66, 96, 0xE7, 0xEE, 0xE7},
	)
}

// raw data is padded and has potential starts inside the data
func TestExtractorSuccessWithStartsInData(t *testing.T) {
	label := byte(6)
	runAndAssertValid(
		t,
		label,
		[]byte{69, 0x7E, 96},
		[]byte{0xEE},
		[]byte{0x11, 0x7E, label, 3, 0, 69, 0x7E, 96, 0xE7, 0xEE},
	)
}

// raw data is padded and has potential ends inside the data
func TestExtractorSuccessWithEndsInData(t *testing.T) {
	label := byte(7)
	runAndAssertValid(
		t,
		label,
		[]byte{69, 0xE7, 96},
		[]byte{0xEE},
		[]byte{0x11, 0x7E, label, 3, 0, 69, 0xE7, 96, 0xE7, 0xEE},
	)
}

// raw data is padded and the content would also be a valid message
// we expect the outer message to be returned
func TestExtractorSuccessWithDataInData(t *testing.T) {
	label := byte(9)
	runAndAssertValid(
		t,
		label,
		[]byte{0x7E, 5, 3, 0, 69, 66, 96, 0xE7},
		[]byte{},
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
		[]byte{0x11, 0xE7},
		[]byte{0x7E, label, 5, 0, 69, 0x7E, 5, 2, 0, 0xE7, 0x11, 0xE7},
	)
}

// Input bytes are empty
func TestExtractorFailsEmptySlice(t *testing.T) {
	runAndAssertNoMessages(t, []byte{})
}

// Input bytes contain no 'START' symbol
func TestExtractorFailsNoStart(t *testing.T) {
	runAndAssertNoMessages(t, []byte{0x00, 1, 0, 0, 0xE7})
}

// Input bytes contain no 'END' symbol
func TestExtractorFailsNoEnd(t *testing.T) {
	runAndAssertNoMessages(t, []byte{0x7E, 1, 0, 0, 0x00})
}

// Input byte starts with 'END' symbol and ends with 'START' symbol
func TestExtractorFailsEndBeforeStart(t *testing.T) {
	runAndAssertNoMessages(t, []byte{0xE7, 1, 0, 0, 0x7E})
}

func runAndAssertNoMessages(t *testing.T, input []byte) {
	msgs, unUsedBytes := Extract(input)
	if len(msgs) != 0 {
		t.Errorf("expected no messages, however received: %v", msgs)
	}
	// Compare unused bytes length
	if len(unUsedBytes) != len(input) {
		t.Errorf("expected unused bytes to be of length '%d', instead found '%d'", len(input), len(unUsedBytes))
	}
	// Compare unused bytes content
	for i := 0; i < len(input); i++ {
		if unUsedBytes[i] != input[i] {
			t.Errorf("expected unused byte[%d] to be '%X' but was %X", i, input[i], unUsedBytes[i])
		}
	}
}

func runAndAssertValid(t *testing.T, expectedLabel byte, expectedPayload []byte, expectedUnUsedBytes []byte, input []byte) {
	msgs, unUsedBytes := Extract(input)
	// Should have one message
	if len(msgs) != 1 {
		t.Errorf("expected one message, however received: %v", msgs)
	}
	msg := msgs[0]
	// check Label
	if msg.GetLabel() != expectedLabel {
		t.Errorf("expected label to be %X, but was %X", expectedLabel, msg.GetLabel())
	}
	// Compare payload lengths
	if len(msg.GetPayload()) != len(expectedPayload) {
		t.Errorf("expected payload length to be '%d', instead found '%d'", len(expectedPayload), len(msg.GetPayload()))
	}
	// Compare payload contents
	for i := 0; i < len(expectedPayload); i++ {
		if msg.GetPayload()[i] != expectedPayload[i] {
			t.Errorf("expected payload byte[%d] to be '%X' but was %X", i, expectedPayload[i], msg.GetPayload()[i])
		}
	}
	// Compare unused bytes length
	if len(unUsedBytes) != len(expectedUnUsedBytes) {
		t.Errorf("expected unused bytes to be of length '%d', instead found '%d'", len(expectedUnUsedBytes), len(unUsedBytes))
	}
	// Compare unused bytes content
	for i := 0; i < len(expectedUnUsedBytes); i++ {
		if unUsedBytes[i] != expectedUnUsedBytes[i] {
			t.Errorf("expected unused byte[%d] to be '%X' but was %X", i, expectedUnUsedBytes[i], unUsedBytes[i])
		}
	}
}
