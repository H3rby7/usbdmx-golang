package dmxusbpro

import "fmt"

func Extract(serialData []byte) (msg EnttecDMXUSBProApplicationMessage, err error) {
	// Find potential start/end points
	potentialStarts := make([]int, 0, 1)
	potentialEnds := make([]int, 0, 1)
	for i, v := range serialData {
		if v == MSG_DELIM_START {
			potentialStarts = append(potentialStarts, i)
		}
		if v == MSG_DELIM_END {
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
			msg, err := FromBytes(serialData[start : end+1])
			if err == nil {
				return msg, err
			}
		}
	}
	err = fmt.Errorf("data does not contain a valid message")
	return
}
