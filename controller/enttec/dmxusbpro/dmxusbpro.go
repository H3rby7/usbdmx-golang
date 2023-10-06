package dmxusbpro

import (
	"fmt"
	"log"

	"github.com/tarm/serial"
)

const (
	// Maximum DMX channels we can control (fixed size in this implementation)
	DMX_MAX_CHANNELS = 16
	// Data length = Channels + 1 as DMX works 1-indexed [not 0-indexed]
	DMX_DATA_LENGTH = DMX_MAX_CHANNELS + 1
)

// Controller for Enttec DMX USB Pro device to handle comms
type EnttecDMXUSBProController struct {
	// Holds DMX data, as DMX starts with channel '1' the index '0' is unused.
	channels []byte

	isWriter bool
	isReader bool
	conf     *serial.Config
	port     *serial.Port
}

// Helper function for creating a new DMX USB PRO controller
func NewEnttecDMXUSBProController(conf *serial.Config, isWriter bool) *EnttecDMXUSBProController {
	d := &EnttecDMXUSBProController{}
	d.channels = make([]byte, DMX_DATA_LENGTH)

	d.conf = conf
	d.isWriter = isWriter
	d.isReader = !isWriter

	return d
}

func (d *EnttecDMXUSBProController) GetName() string {
	return d.conf.Name
}

// Connect handles connection to a Enttec DMX USB Pro controller
func (d *EnttecDMXUSBProController) Connect() error {
	s, err := serial.OpenPort(d.conf)
	if err != nil {
		return err
	}
	d.port = s

	return nil
}

func (d *EnttecDMXUSBProController) Disconnect() error {
	if d.port == nil {
		log.Printf("Not connected.")
		return nil
	}

	return d.port.Close()
}

// Gets a copy of all of the channels of a universe
// Returns our internal state of the channels
// In write mode that means whatever we have set so far
// In read mode that means whatever we read last.
func (d *EnttecDMXUSBProController) GetStage() []byte {
	channels := make([]byte, len(d.channels))

	copy(channels, d.channels)

	return channels
}

func (d *EnttecDMXUSBProController) Stage(index int16, data byte) error {
	if !d.isWriter {
		return fmt.Errorf("controller is not in WRITE mode")
	}

	if index < 1 || index > DMX_MAX_CHANNELS {
		return fmt.Errorf("index %d out of range, must be between 1 and %d", index, DMX_MAX_CHANNELS)
	}

	d.channels[index] = data

	return nil
}

func (d *EnttecDMXUSBProController) Commit() error {

	if !d.isWriter {
		return fmt.Errorf("controller is not in WRITE mode")
	}

	if d.port == nil {
		return fmt.Errorf("not connected")
	}

	msg := EnttecDMXUSBProApplicationMessage{payload: d.channels, label: 6}
	packet := msg.ToBytes()

	log.Printf("Writing %v", packet)

	_, err := d.port.Write(packet)
	if err != nil {
		return err
	}

	return nil
}

func (d *EnttecDMXUSBProController) Clear() {
	for i := range d.channels {
		d.channels[i] = 0
	}
}

func (d *EnttecDMXUSBProController) ReadOnChangeOnly() error {
	packet := []byte{
		MSG_DELIM_START,
		8,
		1,
		0,
		1,
		MSG_DELIM_END,
	}
	log.Printf("Writing %v", packet)

	_, err := d.port.Write(packet)
	if err != nil {
		return err
	}

	return nil
}

func (d *EnttecDMXUSBProController) Read() error {
	if !d.isReader {
		return fmt.Errorf("controller is not in READ mode")
	}
	packet := make([]byte, 512)
	// Read until we get a 126 (7E)
	// Assume next byte is 9 (change packet) or 5 (update)
	// datasize LSB
	// datasize MSB
	// calculate position of expected 231 (E7)
	// If 231 is at expected position we can interpret the byte array.
	n, err := d.port.Read(packet)
	if err != nil {
		log.Fatal(err)
	}
	// TODO: extract channels from packet
	log.Printf("Read %d bytes -> %v", n, packet[0:n])
	return err
}
