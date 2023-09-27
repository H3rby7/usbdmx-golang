package dmxusbpro

import (
	"errors"
	"fmt"

	usbdmxconfig "github.com/H3rby7/usbdmx-golang/config"
	"github.com/google/gousb"
)

const (
	// "Start of Message" delimiter
	MSG_DELIM_START = 0x7E
	// Data length (fixed size in this implementation)
	DATA_LENGTH = 512
	// Least significant bytes to describe data length
	DATA_LENGTH_LSB = byte(DATA_LENGTH & 0xFF)
	// Most significant bytes to describe data length
	DATA_LENGTH_MSB = byte(DATA_LENGTH >> 8 & 0xFF)
	// Combined number of bytes BEFORE payload
	NUM_BYTES_BEFORE_PAYLOAD = 4
	// "End of Message" delimiter
	MSG_DELIM_END = 0xE7
	// Combined number of bytes AFTER payload
	NUM_BYTES_AFTER_PAYLOAD = 1
	// Size of full packed, payload, before and after
	PACKET_SIZE = DATA_LENGTH + NUM_BYTES_BEFORE_PAYLOAD + NUM_BYTES_AFTER_PAYLOAD
)

// Controller for Enttec DMX USB Pro device to handle comms
type EnttecDMXUSBProController struct {
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

// Helper function for creating a new DMX USB PRO controller
func NewEnttecDMXUSBProController(conf usbdmxconfig.ControllerConfig) (d *EnttecDMXUSBProController) {

	d.channels = make([]byte, DATA_LENGTH)

	d.packet = make([]byte, PACKET_SIZE)

	d.vid = conf.VID
	d.pid = conf.PID
	d.outputInterfaceID = conf.OutputInterfaceID
	d.inputInterfaceID = conf.InputInterfaceID
	d.ctx = gousb.NewContext()
	d.ctx.Debug(conf.DebugLevel)

	return d
}

// GetProduct returns a device product name
func (d *EnttecDMXUSBProController) GetProduct() (string, error) {
	return d.device.Product()
}

// GetSerial returns a device serial number
func (d *EnttecDMXUSBProController) GetSerial() (string, error) {
	return d.device.SerialNumber()
}

// Connect handles connection to a Enttec DMX USB Pro controller
func (d *EnttecDMXUSBProController) Connect() error {
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
func (d *EnttecDMXUSBProController) Disconnect() error {
	d.isDisconnected = true
	if d.device == nil {
		return nil
	}

	return d.device.Close()
}

// GetChannels gets a copy of all of the channels of a universe
func (d *EnttecDMXUSBProController) GetChannels() ([]byte, error) {
	channels := make([]byte, len(d.channels))

	copy(channels, d.channels)

	return channels, nil
}

// SetChannel sets a single DMX channel value
func (d *EnttecDMXUSBProController) SetChannel(index int16, data byte) error {
	if index < 1 || index > DATA_LENGTH {
		return fmt.Errorf("Index %d out of range, must be between 1 and %d", index, DATA_LENGTH)
	}

	d.channels[index-1] = data

	return nil
}

// GetChannel returns the value of a single DMX channel
func (d *EnttecDMXUSBProController) GetChannel(index int16) (byte, error) {
	if index < 1 || index > DATA_LENGTH {
		return 0, fmt.Errorf("Index %d out of range, must be between 1 and %d", index, DATA_LENGTH)
	}

	return d.channels[index-1], nil
}

// Render sends channel data to fixtures
func (d *EnttecDMXUSBProController) Render() error {
	// ENTTEC USB DMX PRO Start Message
	d.packet[0] = MSG_DELIM_START

	// Set our protocol
	d.packet[1] = 0x06

	d.packet[2] = DATA_LENGTH_LSB
	d.packet[3] = DATA_LENGTH_MSB

	// Set DMX Data
	for i := 0; i < DATA_LENGTH; i++ {
		d.packet[NUM_BYTES_BEFORE_PAYLOAD+i] = d.channels[i]
	}

	// ENTTEC USB DMX PRO End Message
	d.packet[PACKET_SIZE-1] = MSG_DELIM_END

	if _, err := d.output.Write(d.packet); err != nil {
		return err
	}

	return nil
}
