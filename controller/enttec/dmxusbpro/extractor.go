package dmxusbpro

import (
	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro/messages"
)

/*
Detect valid Messages in an array of bytes.

Returns:

* found messages

* unUsedBytes (any bytes AFTER the last detected message that have not yet been used)
*/
func Extract(serialData []byte) (msgs []messages.EnttecDMXUSBProApplicationMessage, unUsedBytes []byte) {
	// List of extracted messages
	msgs = make([]messages.EnttecDMXUSBProApplicationMessage, 0, 1)
	findDataFromByte := 0
	// Find potential start/end points
	potentialStarts := make([]int, 0, 1)
	potentialEnds := make([]int, 0, 1)
	for i, v := range serialData {
		if v == messages.MSG_DELIM_START {
			potentialStarts = append(potentialStarts, i)
		}
		if v == messages.MSG_DELIM_END {
			potentialEnds = append(potentialEnds, i)
		}
	}
	// **** Check if a combination of Start/End Point gives us a valid DMX message. ****
	for _, start := range potentialStarts {
		for _, end := range potentialEnds {
			if end < start || start < findDataFromByte {
				// no need to check when potential message ends before its start
				// no need to check when potential message starts within already used data
				continue
			}
			msg, err := messages.FromBytes(serialData[start : end+1])
			if err == nil {
				// no error means this is a valid message
				findDataFromByte = end + 1
				msgs = append(msgs, msg)
			}
		}
	}
	if len(serialData) > findDataFromByte {
		unUsedBytes = serialData[findDataFromByte:]
	} else {
		unUsedBytes = make([]byte, 0)
	}
	return
}
