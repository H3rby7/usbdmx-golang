package dmxusbpro

import "fmt"

const (
	// "Start of Message" delimiter
	MSG_DELIM_START = 0x7E
	// Combined number of bytes BEFORE payload
	NUM_BYTES_BEFORE_PAYLOAD = 4
	// "End of Message" delimiter
	MSG_DELIM_END = 0xE7
	// Combined number of bytes AFTER payload
	NUM_BYTES_AFTER_PAYLOAD = 1
	// Number of bytes that are not payload
	NUM_BYTES_WRAPPER = NUM_BYTES_BEFORE_PAYLOAD + NUM_BYTES_AFTER_PAYLOAD
	// Smallest possible label-index to identify the message type
	SMALLEST_LABEL_INDEX = 1
	// Biggest possible label-index to identify the message type
	BIGGEST_LABEL_INDEX = 11
)

const (
	// Position of 'MSG_DELIM_START' in the byte array
	MSG_DELIM_START_INDEX = 0
	// Position of 'label' in the byte array
	MSG_LABEL_INDEX = 1
	// Position of 'data length LSB' in the byte array
	MSG_DATA_LENGTH_LSB_INDEX = 2
	// Position of 'data length MSB' in the byte array
	MSG_DATA_LENGTH_MSB_INDEX = 3
)

// Controller for Enttec DMX USB Pro device to handle comms
type EnttecDMXUSBProApplicationMessage struct {
	// Message Content
	payload []byte
	// Label to identify the type of message
	label byte
}

func (msg *EnttecDMXUSBProApplicationMessage) ToBytes() []byte {
	dataLength := len(msg.payload)
	packetSize := dataLength + NUM_BYTES_WRAPPER
	packet := make([]byte, packetSize)
	// Add 'start message'-delimiter
	packet[MSG_DELIM_START_INDEX] = MSG_DELIM_START
	// Set our protocol
	packet[MSG_LABEL_INDEX] = msg.label
	// Least significant bytes to describe data length
	packet[MSG_DATA_LENGTH_LSB_INDEX] = byte(dataLength & 0xFF)
	// Most significant bytes to describe data length
	packet[MSG_DATA_LENGTH_MSB_INDEX] = byte(dataLength >> 8 & 0xFF)

	// Set DMX Data
	for i := 0; i < dataLength; i++ {
		packet[NUM_BYTES_BEFORE_PAYLOAD+i] = msg.payload[i]
	}

	// Add 'end message'-delimiter
	packet[packetSize-1] = MSG_DELIM_END

	return packet
}

func FromBytes(raw []byte) (msg EnttecDMXUSBProApplicationMessage, err error) {
	if err = validateSchema(raw); err != nil {
		return
	}
	if err = validateSize(raw); err != nil {
		return
	}
	payloadStart := NUM_BYTES_BEFORE_PAYLOAD - 1
	payloadEnd := len(raw) - NUM_BYTES_AFTER_PAYLOAD - 1
	msg = EnttecDMXUSBProApplicationMessage{
		label:   raw[MSG_LABEL_INDEX],
		payload: raw[payloadStart:payloadEnd],
	}
	return
}

// Validate the bytes according to the message definition.
// Return error if any validation fails, else nil.
func validateSchema(raw []byte) error {
	size := len(raw)
	if size < NUM_BYTES_WRAPPER {
		return fmt.Errorf("message of size %d bytes is too small - must be at least %d bytes", size, NUM_BYTES_WRAPPER)
	}
	if raw[MSG_DELIM_START_INDEX] != MSG_DELIM_START {
		return fmt.Errorf("message must start with %X, but is %X", MSG_DELIM_START, raw[MSG_DELIM_START_INDEX])
	}
	if raw[size-1] != MSG_DELIM_END {
		return fmt.Errorf("message must end with %X, but is %X", MSG_DELIM_END, raw[size-1])
	}
	label := raw[MSG_LABEL_INDEX]
	if label < SMALLEST_LABEL_INDEX {
		return fmt.Errorf("message label must be at least %d, but is %d", SMALLEST_LABEL_INDEX, label)
	}
	if label > BIGGEST_LABEL_INDEX {
		return fmt.Errorf("message label must be at maximum %d, but is %d", BIGGEST_LABEL_INDEX, label)
	}
	return nil
}

// Validate the bytes according to the message definition.
// Return error if any validation fails, else nil.
func validateSize(raw []byte) error {
	lsb := raw[MSG_DATA_LENGTH_LSB_INDEX]
	msb := raw[MSG_DATA_LENGTH_MSB_INDEX]
	indicatedPayloadSize := int(lsb) + (256 * int(msb))
	actualPayloadSize := len(raw) - NUM_BYTES_WRAPPER
	if indicatedPayloadSize != actualPayloadSize {
		return fmt.Errorf("message declared payload size as %d, but is %d", indicatedPayloadSize, actualPayloadSize)
	}
	return nil
}
