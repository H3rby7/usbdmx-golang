package dmxusbpro

const (
	// "Start of Message" delimiter
	MSG_DELIM_START = 0x7E
	// Combined number of bytes BEFORE payload
	NUM_BYTES_BEFORE_PAYLOAD = 4
	// "End of Message" delimiter
	MSG_DELIM_END = 0xE7
	// Combined number of bytes AFTER payload
	NUM_BYTES_AFTER_PAYLOAD = 1
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
	packetSize := NUM_BYTES_BEFORE_PAYLOAD + dataLength + NUM_BYTES_AFTER_PAYLOAD
	packet := make([]byte, packetSize)
	// ENTTEC USB DMX PRO Start Message
	packet[0] = MSG_DELIM_START
	// Set our protocol
	packet[1] = msg.label
	// Least significant bytes to describe data length
	packet[2] = byte(dataLength & 0xFF)
	// Most significant bytes to describe data length
	packet[3] = byte(dataLength >> 8 & 0xFF)

	// Set DMX Data
	for i := 0; i < DMX_DATA_LENGTH; i++ {
		packet[NUM_BYTES_BEFORE_PAYLOAD+i] = msg.payload[i]
	}

	// ENTTEC USB DMX PRO End Message
	packet[PACKET_SIZE-1] = MSG_DELIM_END

	return packet
}
