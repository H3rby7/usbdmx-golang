package enttecdmxusbpro

import (
	"errors"
	"fmt"

	usbdmx "github.com/H3rby7/usbdmx-golang"
	"github.com/google/gousb"
)

const (
	START_VAL       = 0x7E
	END_VAL         = 0xE7
	FRAME_SIZE      = 512
	FRAME_SIZE_LOW  = byte(FRAME_SIZE & 0xFF)
	FRAME_SIZE_HIGH = byte(FRAME_SIZE >> 8 & 0xFF)
	PACKET_HEADER   = 4
	PACKET_SIZE     = FRAME_SIZE + PACKET_HEADER + 1
)

// DMXController a real world Enttec DMX USB Pro device to handle comms
type DMXController struct {
	vid      uint16
	pid      uint16
	channels []byte
	packet   []byte

	ctx               *gousb.Context
	device            *gousb.Device
	output            *gousb.OutEndpoint
	input             *gousb.InEndpoint
	outputInterfaceID int
	inputInterfaceID  int

	hasError bool
	err      error

	isDisconnected bool
}

// NewDMXController helper function for creating a new DMX USB PRO controller
func NewDMXController(conf usbdmx.ControllerConfig) DMXController {
	d := DMXController{}

	d.channels = make([]byte, FRAME_SIZE)

	d.packet = make([]byte, PACKET_SIZE)

	d.vid = conf.VID
	d.pid = conf.PID
	d.outputInterfaceID = conf.OutputInterfaceID
	d.inputInterfaceID = conf.InputInterfaceID
	d.ctx = conf.Context

	return d
}

// GetProduct returns a device product name
func (d *DMXController) GetProduct() (string, error) {
	return d.device.Product()
}

// GetSerial returns a device serial number
func (d *DMXController) GetSerial() (string, error) {
	return d.device.SerialNumber()
}

// Connect handles connection to a Enttec DMX USB Pro controller
func (d *DMXController) Connect() error {
	if d.ctx == nil {
		return errors.New("the libusb context is missing")
	}
	// try to connect to device
	device, err := d.ctx.OpenDeviceWithVIDPID(gousb.ID(d.vid), gousb.ID(d.pid))
	if err != nil {
		d.hasError = true
		d.err = err
		return err
	}

	// ensure we have the device
	if device == nil {
		d.hasError = true
		d.err = errors.New("usb device not connected/found")
		return d.err
	}
	d.device = device

	// make this device ours, even if it is being used elsewhere
	if err := d.device.SetAutoDetach(true); err != nil {
		d.hasError = true
		d.err = err
		return err
	}

	// open communications
	commsInterface, _, err := d.device.DefaultInterface()
	if err != nil {
		d.hasError = true
		d.err = err
		return err
	}

	d.output, err = commsInterface.OutEndpoint(d.outputInterfaceID)
	if err != nil {
		d.hasError = true
		d.err = err
		return err
	}

	d.hasError = false
	d.err = nil
	d.isDisconnected = false

	return nil
}

// Disconnect disconnects the usb device
func (d *DMXController) Disconnect() error {
	d.isDisconnected = true
	if d.device == nil {
		return nil
	}

	return d.device.Close()
}

// GetChannels gets a copy of all of the channels of a universe
func (d *DMXController) GetChannels() ([]byte, error) {
	channels := make([]byte, len(d.channels))

	copy(channels, d.channels)

	return channels, nil
}

// SetChannel sets a single DMX channel value
func (d *DMXController) SetChannel(index int16, data byte) error {
	if index < 1 || index > FRAME_SIZE {
		return fmt.Errorf("Index %d out of range, must be between 1 and %d", index, FRAME_SIZE)
	}

	d.channels[index-1] = data

	return nil
}

// GetChannel returns the value of a single DMX channel
func (d *DMXController) GetChannel(index int16) (byte, error) {
	if index < 1 || index > FRAME_SIZE {
		return 0, fmt.Errorf("Index %d out of range, must be between 1 and %d", index, FRAME_SIZE)
	}

	return d.channels[index-1], nil
}

// Render sends channel data to fixtures
func (d *DMXController) Render() error {
	// ENTTEC USB DMX PRO Start Message
	d.packet[0] = START_VAL

	// Set our protocol
	d.packet[1] = 0x06

	d.packet[2] = FRAME_SIZE_LOW
	d.packet[3] = FRAME_SIZE_HIGH

	// Set DMX Data
	for i := 0; i < FRAME_SIZE; i++ {
		d.packet[PACKET_HEADER+i] = d.channels[i]
	}

	// ENTTEC USB DMX PRO End Message
	d.packet[PACKET_SIZE-1] = END_VAL

	if _, err := d.output.Write(d.packet); err != nil {
		return err
	}

	return nil
}
