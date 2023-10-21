package dmxusbpro

import (
	"fmt"
	"log"
	"time"

	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro/messages"
	"github.com/tarm/serial"
)

// Prefix for any log and error messages
const ENTTEC_DMX_USB_PRO_LOG_PREFIX = "EDUP"

// Controller for Enttec DMX USB Pro device to handle communication
type EnttecDMXUSBProController struct {
	// Holds DMX data, as DMX starts with channel '1' the index '0' is unused.
	channels []byte

	isWriter bool
	isReader bool
	// Is the widget in 'only read changes'-mode (as opposed to read everything)
	readOnChange bool

	isConnected  bool
	conf         *serial.Config
	port         *serial.Port
	logVerbosity uint8
}

// Helper function for creating a new DMX USB PRO controller
func NewEnttecDMXUSBProController(conf *serial.Config, dmxChannelCount int, isWriter bool) *EnttecDMXUSBProController {
	d := &EnttecDMXUSBProController{}
	d.channels = make([]byte, dmxChannelCount+1)

	d.conf = conf
	d.isWriter = isWriter
	d.isReader = !isWriter
	d.readOnChange = false
	d.isConnected = false
	d.logVerbosity = 0

	return d
}

/*
	Returns the name used for opening the connection

e.g. "COM4" or "/dev/tty.usbserial"
*/
func (d *EnttecDMXUSBProController) GetName() string {
	return d.conf.Name
}

/*
	Open the serial connection to Enttec DMX USB PRO Widget

Succeeded if no error is returned
*/
func (d *EnttecDMXUSBProController) Connect() error {
	s, err := serial.OpenPort(d.conf)
	if err != nil {
		return err
	}
	d.port = s
	d.isConnected = true
	return nil
}

/*
	Close the serial connection to Enttec DMX USB PRO Widget

Succeeded if no error is returned
*/
func (d *EnttecDMXUSBProController) Disconnect() error {
	if d.port == nil {
		return d.errorf("not connected.")
	}
	d.isConnected = false
	return d.port.Close()
}

// Gets a copy of all staged channel values
func (d *EnttecDMXUSBProController) GetStage() []byte {
	channels := make([]byte, len(d.channels))

	copy(channels, d.channels)

	return channels
}

/*
Prepare a channel to be changed to the given value

Note: This does not send out the changes, you must call the 'Commit' method to apply the stage live.
*/
func (d *EnttecDMXUSBProController) Stage(channel int16, value byte) error {
	if !d.isWriter {
		return d.errorf("controller is not in WRITE mode")
	}
	highestChannel := int16(len(d.channels))
	if channel < 1 || channel > highestChannel {
		return d.errorf("index %d out of range, must be between 1 and %d", channel, highestChannel)
	}
	d.channels[channel] = value
	return nil
}

/*
Apply the 'staged' values to go live.

Note: This does not clear the Stage!
*/
func (d *EnttecDMXUSBProController) Commit() error {
	if !d.isWriter {
		return d.errorf("controller is not in WRITE mode")
	}
	msg := messages.NewEnttecDMXUSBProApplicationMessage(messages.LABEL_OUTPUT_ONLY_SEND_DMX_PACKET_REQUEST, d.channels)
	return d.writeMessage(msg)
}

// Set all values of the staged channels to '0'
func (d *EnttecDMXUSBProController) ClearStage() {
	for i := range d.channels {
		d.channels[i] = 0
	}
}

/*
Helper function to check for connection, log and transform
*/
func (d *EnttecDMXUSBProController) writeMessage(msg messages.EnttecDMXUSBProApplicationMessage) error {
	packet, err := msg.ToBytes()
	if err != nil {
		return err
	}
	d.printf(1, "Writing \tlabel=%v \tdata=%v", msg.GetLabel(), msg.GetPayload())
	_, err = d.Write(packet)
	return err
}

/*
Change the receive mode of the widget.

Setting it to '0' will allow reading always and reads everything

Setting it to '1' will only read when the data changes and only reads the changeset (label=9)
*/
func (d *EnttecDMXUSBProController) SwitchReadMode(changesOnly byte) error {
	if changesOnly > 1 {
		d.panicf("invalid value, only 0 and 1 are allowed, but got '%d'", changesOnly)
	}
	if !d.isReader {
		return d.errorf("controller is not in READ mode")
	}
	msg := messages.NewEnttecDMXUSBProApplicationMessage(messages.LABEL_RECEIVE_DMX_ON_CHANGE, []byte{changesOnly})
	if err := d.writeMessage(msg); err != nil {
		return err
	}
	d.readOnChange = true
	return nil
}

/*
Expose serial read to be used directly
*/
func (d *EnttecDMXUSBProController) Read(buf []byte) (int, error) {
	if d.port == nil || !d.isConnected {
		return -1, d.errorf("not connected")
	}
	if !d.isReader {
		return -1, d.errorf("controller is not in READ mode")
	}
	n, err := d.port.Read(buf)
	d.printf(2, "Read %d bytes:\t%v", n, buf[0:n])
	return n, err
}

/*
Expose serial write to be used directly
*/
func (d *EnttecDMXUSBProController) Write(buf []byte) (int, error) {
	if d.port == nil || !d.isConnected {
		return -1, fmt.Errorf("not connected")
	}
	n, err := d.port.Write(buf)
	d.printf(2, "Wrote %d bytes:\t%v", n, buf[0:n])
	return n, err
}

/*
Start routine to read from DMX and get the results back via channel

Example useage:

	c := make(chan messages.EnttecDMXUSBProApplicationMessage) // create channel
	go controller.OnDMXChange(c) // start routine
	for msg := range c { ... } // handle incoming data
*/
func (d *EnttecDMXUSBProController) OnDMXChange(c chan messages.EnttecDMXUSBProApplicationMessage, readIntervalMS int) {
	if !d.readOnChange {
		d.panicf("controller is not in READ ON CHANGE mode!")
	}
	// Buffer used for reading fresh data
	readBuf := make([]byte, messages.MAXIMUM_MESSAGE_LENGTH)
	// Buffer containing old, yet unused data
	oldBuf := make([]byte, 0)
	// Detected messages
	var msgs []messages.EnttecDMXUSBProApplicationMessage
	for {
		n, err := d.Read(readBuf)
		if err != nil {
			d.panicf("error reading from serial, %v", err)
		}
		// Combine newly read data with yet unused data
		combined := append(oldBuf, readBuf[:n]...)
		// Try to extract valid messages
		msgs, oldBuf = Extract(combined)
		for _, msg := range msgs {
			d.printf(1, "Read \tlabel=%v \tdata=%v", msg.GetLabel(), msg.GetPayload())
			c <- msg
		}
		if len(oldBuf) > messages.MAXIMUM_MESSAGE_LENGTH {
			dropOldDataBefore := len(oldBuf) - messages.MAXIMUM_MESSAGE_LENGTH
			d.printf(1, "Dropping old, unused data:\t%v", oldBuf[:dropOldDataBefore])
			oldBuf = oldBuf[dropOldDataBefore:]
		}
		time.Sleep(time.Millisecond * time.Duration(readIntervalMS))
	}
}

/*
Set log verbosity

0 = no logging

1 = message logging

2 = byte logging
*/
func (d *EnttecDMXUSBProController) SetLogVerbosity(verbosity uint8) {
	if verbosity > 2 {
		d.panicf("invalid value, only 0, 1 and 2 are allowed, but got '%d'", verbosity)
	}
	d.logVerbosity = verbosity
}

func (d *EnttecDMXUSBProController) printf(level uint8, format string, v ...any) {
	if d.logVerbosity == level {
		log.Printf(ENTTEC_DMX_USB_PRO_LOG_PREFIX+": "+format, v...)
	}
}

func (d *EnttecDMXUSBProController) errorf(format string, v ...any) error {
	return fmt.Errorf(ENTTEC_DMX_USB_PRO_LOG_PREFIX+": "+format, v...)
}

func (d *EnttecDMXUSBProController) panicf(format string, v ...any) {
	log.Panicf(ENTTEC_DMX_USB_PRO_LOG_PREFIX+": "+format, v...)
}
