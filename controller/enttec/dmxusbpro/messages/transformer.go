package messages

import (
	"fmt"
)

/*
	Convert a message according to the 'Received DMX Change of State Packet' structure.

Message must have label '9' and at least 7 bytes

0    - Start Changed byte number

1 - 5  - Changed bit array, where array bit 0 is bit 0 of first byte and array bit 39 is bit 7 of last
byte

6 to 45 - Changed DMX data byte array. One byte is present for each set bit in the Changed bit
array

Returns an error if the changed bit array has more bits set than data bytes were sent.

The keys of the returned map are DMX slot indices, as in the API's 513 byte 'received_dmx_array':

* key '0' is the DMX start code, not a channel. Keys '1' to '512' are DMX channels '1' to '512'.

* keys range from 'startChangedByteNumber*8' to 'startChangedByteNumber*8 + 39',
so a malformed message can yield keys above '512'.

* keys do not depend on the 'dmxChannelCount' of the controller, so check them before using them as index.
*/
func ToChangeSet(msg EnttecDMXUSBProApplicationMessage) (map[int]byte, error) {
	m := make(map[int]byte)
	if msg.label != LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET {
		return nil, fmt.Errorf("wrong label, expected '%d', but got '%d'", LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET, msg.label)
	}
	if len(msg.payload) < 7 {
		return nil, fmt.Errorf("payload must be at least '%d' bytes, but was '%d'", 7, len(msg.payload))
	}
	// START GOLANG implementation of pseudo-code in API docs
	startChangedByteNumber := int(msg.payload[0])
	changedBitArray := bytesToBools(msg.payload[1:6])
	changedDMXDataArray := msg.payload[6:]
	if setBits := countTrue(changedBitArray); setBits > len(changedDMXDataArray) {
		return nil, fmt.Errorf("changed bit array has %d bits set, but only %d data bytes were sent", setBits, len(changedDMXDataArray))
	}
	changedByteIndex := 0
	// API docs: 'For bit_array_index = 0 to 39', so all 40 bits
	for bitArrayIndex := 0; bitArrayIndex < len(changedBitArray); bitArrayIndex++ {
		if changedBitArray[bitArrayIndex] {
			m[startChangedByteNumber*8+bitArrayIndex] = changedDMXDataArray[changedByteIndex]
			changedByteIndex++
		}
	}
	// END GOLANG implementation of pseudo-code in API docs
	return m, nil
}

/*
	Convert a message according to the 'Received DMX Packet' structure.

Message must have label '5' and at least 1 byte

0    - DMX receive status Bit 0: 0=No error,1=Widget receive queue overflowed. Bit 1: 0=No error,1=Widget receive overrun occurred

1 - 513  - Received DMX data beginning with the start code. Get Size from overall msg size.
*/
func ToDMXArray(msg EnttecDMXUSBProApplicationMessage) ([]byte, error) {
	if msg.label != LABEL_RECEIVED_DMX_PACKET {
		return nil, fmt.Errorf("wrong label, expected '%d', but got '%d'", LABEL_RECEIVED_DMX_PACKET, msg.label)
	}
	if len(msg.payload) < 2 {
		return nil, fmt.Errorf("payload must be at least '%d' bytes, but was '%d'", 2, len(msg.payload))
	}
	if msg.payload[0] != 0 {
		return nil, fmt.Errorf("DMX receive status (payload[0]) should be '%d', but was '%d'", 0, msg.payload[0])
	}
	if msg.payload[1] != 0 {
		return nil, fmt.Errorf("DMX start byte (payload[1]) should be '%d', but was '%d'", 0, msg.payload[1])
	}
	return msg.payload[1:], nil
}

// LSB first, so out[0] is bit 0 of the input
func byteToBools(input byte) []bool {
	out := make([]bool, 8)
	for i := uint(0); i < 8; i++ {
		mask := byte(1 << i)
		masked := (input & mask)
		out[i] = (masked >> i) == 1
	}
	return out
}

// LSB of the first byte first, so out[0] is bit 0 of the first byte and out[8] is bit 0 of the second byte
func bytesToBools(input []byte) []bool {
	out := make([]bool, 0, 8*len(input))
	for i := 0; i < len(input); i++ {
		out = append(out, byteToBools(input[i])...)
	}
	return out
}

func countTrue(bools []bool) int {
	count := 0
	for _, b := range bools {
		if b {
			count++
		}
	}
	return count
}
