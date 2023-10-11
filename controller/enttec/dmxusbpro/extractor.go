package dmxusbpro

import (
	"fmt"

	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro/messages"
)

func Extract(serialData []byte) (msg messages.EnttecDMXUSBProApplicationMessage, err error) {
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
	// Exit early if there are no starting bytes
	if len(potentialStarts) < 1 {
		err = fmt.Errorf("could not detect a message start")
		return
	}
	// Exit early if there are no ending bytes
	if len(potentialEnds) < 1 {
		err = fmt.Errorf("could not detect a message end")
		return
	}
	// Check if a combination of Start/End Point gives us a valid DMX message.
	for _, start := range potentialStarts {
		for _, end := range potentialEnds {
			if end < start {
				// no need to check when end < start
				continue
			}
			msg, err := messages.FromBytes(serialData[start : end+1])
			if err == nil {
				return msg, err
			}
		}
	}
	err = fmt.Errorf("data does not contain a valid message")
	return
}
