package enttecdmxusbpro

import (
	"errors"
	"fmt"

	usbdmx "github.com/H3rby7/usbdmx-golang"
	"github.com/google/gousb"
)

// DMXController a real world Enttec DMX USB Pro Mk2 device to handle comms
type DMXController struct {
	vid      uint16
	pid      uint16
	channels [][]byte
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

// NewDMXController helper function for creating a new DMX USB PRO Mk2 controller
func NewDMXController(conf usbdmx.ControllerConfig) DMXController {
	d := DMXController{}

	// controller has multiple universes
	d.channels = make([][]byte, 2)
	d.channels[0] = make([]byte, 512)
	d.channels[1] = make([]byte, 512)

	d.packet = make([]byte, 518)

	d.vid = conf.VID
	d.pid = conf.PID
	d.outputInterfaceID = conf.OutputInterfaceID
	d.inputInterfaceID = conf.InputInterfaceID
	d.ctx = conf.Context

	return d
}

// GetProduct returns a device product name
func (d *DMXController) GetProduct() (info string, err error) {
	info, err = d.device.Product()
	return info, err
}

// GetSerial returns a device serial number
func (d *DMXController) GetSerial() (info string, err error) {
	info, err = d.device.SerialNumber()
	return info, err
}

// Connect handles connection to a Enttec DMX USB Pro Mk2 controller
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
	cfg, err := d.device.Config(1)
	if err != nil {
		d.hasError = true
		d.err = err
		return err
	}

	intf, err := cfg.Interface(0, 0)
	if err != nil {
		d.hasError = true
		d.err = err
		return err
	}

	d.output, err = intf.OutEndpoint(d.outputInterfaceID)
	if err != nil {
		d.hasError = true
		d.err = err
		return err
	}

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

	if err := sendControlHeaders(d.device); err != nil {
		d.hasError = true
		d.err = err
		return err
	}

	if err := openUniverses(d.output); err != nil {
		d.hasError = true
		d.err = err
		return err
	}

	d.hasError = false
	d.err = nil
	d.isDisconnected = false

	return nil
}

// Send our control headers for this device
func sendControlHeaders(device *gousb.Device) error {
	if _, err := device.Control(0x40, 0x00, 0x00, 0x00, nil); err != nil {
		return err
	}

	if _, err := device.Control(0x40, 0x03, 0x4138, 0x00, nil); err != nil {
		return err
	}

	if _, err := device.Control(0x40, 0x00, 0x00, 0x00, nil); err != nil {
		return err
	}

	if _, err := device.Control(0x40, 0x04, 0x1008, 0x00, nil); err != nil {
		return err
	}

	if _, err := device.Control(0x40, 0x02, 0x00, 0x00, nil); err != nil {
		return err
	}

	if _, err := device.Control(0x40, 0x03, 0x000c, 0x00, nil); err != nil {
		return err
	}
	if _, err := device.Control(0x40, 0x00, 0x0001, 0x00, nil); err != nil {
		return err
	}

	if _, err := device.Control(0x40, 0x00, 0x0002, 0x00, nil); err != nil {
		return err
	}

	if _, err := device.Control(0x40, 0x01, 0x0200, 0x00, nil); err != nil {
		return err
	}

	return nil
}

// Open universe 1 & 2 for writing
func openUniverses(out *gousb.OutEndpoint) error {
	if _, err := out.Write([]byte{0x7E, 0x0D, 0x04, 0x00, 0xAD, 0x88, 0xD0, 0xC8, 0xE7}); err != nil {
		return err
	}
	if _, err := out.Write([]byte{0x7E, 0xCB, 0x02, 0x00, 0x01, 0x01, 0xE7}); err != nil {
		return err
	}
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
func (d *DMXController) GetChannels(universe int16) ([]byte, error) {
	if universe < 1 || universe > int16(len(d.channels)) {
		return make([]byte, 0), fmt.Errorf("Universe %d does not exist", universe)
	}

	channels := make([]byte, len(d.channels[universe]))

	copy(channels, d.channels[universe])

	return channels, nil
}

// SetChannel sets a single DMX channel value
func (d *DMXController) SetChannel(universe int16, index int16, data byte) error {
	if universe < 1 || universe > int16(len(d.channels)) {
		return fmt.Errorf("Universe %d does not exist", universe)
	}

	if index < 1 || index > 512 {
		return fmt.Errorf("Index %d out of range, must be between 1 and 512", index)
	}

	d.channels[universe-1][index-1] = data

	return nil
}

// GetChannel returns the value of a single DMX channel
func (d *DMXController) GetChannel(universe int16, index int16) (byte, error) {
	if universe < 1 || universe > int16(len(d.channels)) {
		return 0, fmt.Errorf("Universe %d does not exist", universe)
	}

	if index < 1 || index > 512 {
		return 0, fmt.Errorf("Index %d out of range, must be between 1 and 512", index)
	}

	return d.channels[universe-1][index-1], nil
}

// Render sends channel data to fixtures
func (d *DMXController) Render() error {
	if err := d.renderUniverseOne(); err != nil {
		return fmt.Errorf("Error rendering universe 1: %s", err)
	}

	if err := d.renderUniverseTwo(); err != nil {
		return fmt.Errorf("Error rendering universe 2: %s", err)
	}

	return nil
}

func (d *DMXController) renderUniverseOne() error {
	// ENTTEC USB DMX PRO Start Message
	d.packet[0] = 0x7E

	// Set our protocol
	d.packet[1] = 0x06
	d.packet[2] = 0x01

	// Wat?
	d.packet[3] = 0x02
	d.packet[4] = 0x00

	// Set DMX Data
	for i := 0; i < 512; i++ {
		d.packet[i+5] = d.channels[0][i]
	}

	// ENTTEC USB DMX PRO End Message
	d.packet[517] = 0xE7

	if _, err := d.output.Write(d.packet); err != nil {
		return err
	}

	return nil
}

func (d *DMXController) renderUniverseTwo() error {
	// ENTTEC USB DMX PRO Start Message
	d.packet[0] = 0x7E

	// Set our protocol
	d.packet[1] = 0xa9

	// Wat?
	d.packet[2] = 0x01
	d.packet[3] = 0x02
	d.packet[4] = 0x00

	// Set DMX Data
	for i := 0; i < 512; i++ {
		d.packet[i+5] = d.channels[1][i]
	}

	// ENTTEC USB DMX PRO End Message
	d.packet[517] = 0xE7

	if _, err := d.output.Write(d.packet); err != nil {
		return err
	}

	return nil
}
