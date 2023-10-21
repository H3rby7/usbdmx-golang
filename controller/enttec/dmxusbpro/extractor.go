package dmxusbpro

import (
	"fmt"

	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro/messages"
)

func Extract(serialData []byte) (msgs []messages.EnttecDMXUSBProApplicationMessage, err error) {
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
	// TODO: We tend to run late by one message!
	/*
		16:53:25.608119 EDUP: Read 1 bytes:     [126]
		16:53:25.630279 EDUP: Read 24 bytes:    [9 8 0 0 6 0 0 0 0 108 54 231 126 9 7 0 0 8 0 0 0 0 27 231]
		16:53:25.632095 READER  Changeset is:   map[1:108 2:54]
																														(reads first msg, but second one would already be there)
		16:53:25.646188 EDUP: Read 13 bytes:    [126 9 8 0 0 6 0 0 0 0 110 55 231]
		16:53:25.646593 READER  Changeset is:   map[3:27] ()
																														(reads second msg of last read, but third one would already be there)
		16:53:25.663382 READER  Changeset is:   map[1:110 2:55]
	*/
	// **** Check if a combination of Start/End Point gives us a valid DMX message. ****
	// List of extracted messages
	msgs = make([]messages.EnttecDMXUSBProApplicationMessage, 0, 1)
	for _, start := range potentialStarts {
		for _, end := range potentialEnds {
			if end < start {
				// no need to check when end < start
				continue
			}
			msg, err := messages.FromBytes(serialData[start : end+1])
			if err == nil {
				// no error means this is a valid message
				msgs = append(msgs, msg)
			}
		}
	}
	if len(msgs) < 1 {
		err = fmt.Errorf("data does not contain a valid message")
	}
	return
}
