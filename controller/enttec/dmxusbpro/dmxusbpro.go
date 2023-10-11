package dmxusbpro

import (
	"fmt"
	"log"
	"time"

	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro/messages"
	"github.com/tarm/serial"
)

const (
	// Maximum DMX channels we can control (fixed size in this implementation)
	DMX_MAX_CHANNELS = 16
	// Data length = Channels + 1 as DMX works 1-indexed [not 0-indexed]
	DMX_DATA_LENGTH = DMX_MAX_CHANNELS + 1
)

// Controller for Enttec DMX USB Pro device to handle communication
type EnttecDMXUSBProController struct {
	// Holds DMX data, as DMX starts with channel '1' the index '0' is unused.
	channels []byte

	isWriter bool
	isReader bool
	// Is the widget in 'only read changes'-mode (as opposed to read everything)
	readOnChange bool

	isConnected bool
	conf        *serial.Config
	port        *serial.Port
}

// Helper function for creating a new DMX USB PRO controller
func NewEnttecDMXUSBProController(conf *serial.Config, isWriter bool) *EnttecDMXUSBProController {
	d := &EnttecDMXUSBProController{}
	d.channels = make([]byte, DMX_DATA_LENGTH)

	d.conf = conf
	d.isWriter = isWriter
	d.isReader = !isWriter
	d.readOnChange = false
	d.isConnected = false

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
		log.Printf("Not connected.")
		return nil
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
		return fmt.Errorf("controller is not in WRITE mode")
	}
	if channel < 1 || channel > DMX_MAX_CHANNELS {
		return fmt.Errorf("index %d out of range, must be between 1 and %d", channel, DMX_MAX_CHANNELS)
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
		return fmt.Errorf("controller is not in WRITE mode")
	}
	msg := messages.NewEnttecDMXUSBProApplicationMessage(messages.LABEL_OUTPUT_ONLY_SEND_DMX_PACKET_REQUESTS, d.channels)
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
	log.Printf("Writing \tlabel=%v \tdata=%v", msg.GetLabel(), msg.GetPayload())
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
		log.Panicf("invalid value, only 0 and 1 are allowed, but got '%d'", changesOnly)
	}
	if !d.isReader {
		return fmt.Errorf("controller is not in READ mode")
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
		return -1, fmt.Errorf("not connected")
	}
	if !d.isReader {
		return -1, fmt.Errorf("controller is not in READ mode")
	}
	return d.port.Read(buf)
}

/*
Expose serial write to be used directly
*/
func (d *EnttecDMXUSBProController) Write(buf []byte) (int, error) {
	if d.port == nil || !d.isConnected {
		return -1, fmt.Errorf("not connected")
	}
	return d.port.Write(buf)
}

/*
Start routine to read from DMX and get the results back via channel

Example useage:

	controller.SwitchReadMode(1) // read changes only
	c := make(chan messages.EnttecDMXUSBProApplicationMessage) // create channel
	go controller.OnDMXChange(c) // start routine
	for msg := range c { ... } // handle incoming data
*/
func (d *EnttecDMXUSBProController) OnDMXChange(c chan messages.EnttecDMXUSBProApplicationMessage) {
	if !d.readOnChange {
		log.Fatalf("controller is not in READ ON CHANGE mode!")
	}
	// Create a ring buffer that can store two reads, as we might get unlucky and read a start-byte just at the end of our current read.
	// We will switch the buffer to read into with every iteration. The other one will contain the last read, so we can reliably detect messages.
	ringbuff := [][]byte{
		make([]byte, messages.NUM_BYTES_WRAPPER+messages.MAXIMUM_DATA_LENGTH),
		make([]byte, messages.NUM_BYTES_WRAPPER+messages.MAXIMUM_DATA_LENGTH),
	}
	order := 0
	for {
		// Calculate order of buffers
		order = (order + 1) % 2
		bufNow := order
		bufOld := (order + 1) % 2
		// Read into current first buffer
		n, err := d.Read(ringbuff[bufNow])
		if err != nil {
			log.Panicf("error reading from serial, %v", err)
		}
		// Combine with the older buffer
		combined := append(ringbuff[bufNow][0:n], ringbuff[bufOld]...)
		// Try to extract a valid message
		msg, err := Extract(combined)
		if err == nil {
			// No error means there is a message
			c <- msg
		}
		time.Sleep(time.Millisecond * 30)
	}
}
