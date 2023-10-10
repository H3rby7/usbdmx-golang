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
*/
func ToChangeSet(msg EnttecDMXUSBProApplicationMessage) (map[int]byte, error) {
	m := make(map[int]byte)
	if msg.label != LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET {
		return nil, fmt.Errorf("wrong label, expected '%d', but got '%d", LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET, msg.label)
	}
	if len(msg.payload) < 7 {
		return nil, fmt.Errorf("payload must be at least '%d' bytes, but was '%d", 7, len(msg.payload))
	}
	// START GOLANG implementation of pseudo-code in API docs
	startChangedByteNumber := int(msg.payload[0])
	changedBitArray := bytesToBools(msg.payload[1:6])
	changedDMXDataArray := msg.payload[6:]
	changedByteIndex := 0
	for bitArrayIndex := 0; bitArrayIndex < 39; bitArrayIndex++ {
		if changedBitArray[bitArrayIndex] {
			m[startChangedByteNumber*8+bitArrayIndex] = changedDMXDataArray[changedByteIndex]
			changedByteIndex++
		}
	}
	// END GOLANG implementation of pseudo-code in API docs
	return m, nil
}

// MSBs first
func byteToBools(input byte) []bool {
	out := make([]bool, 8)
	for i := uint(0); i < 8; i++ {
		mask := byte(1 << i)
		masked := (input & mask)
		out[i] = (masked >> i) == 1
	}
	return out
}

// MSBs first
func bytesToBools(input []byte) []bool {
	out := make([]bool, 0, 8*len(input))
	for i := 0; i < len(input); i++ {
		out = append(out, byteToBools(input[i])...)
	}
	return out
}
