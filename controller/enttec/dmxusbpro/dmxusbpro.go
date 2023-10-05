package dmxusbpro

import (
	"fmt"
	"log"

	"github.com/tarm/serial"
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
	channels []byte
	packet   []byte

	isWriter bool
	isReader bool
	conf     *serial.Config
	port     *serial.Port
}

// Helper function for creating a new DMX USB PRO controller
func NewEnttecDMXUSBProController(conf *serial.Config, isWriter bool) (d *EnttecDMXUSBProController) {
	d.channels = make([]byte, DATA_LENGTH)
	d.packet = make([]byte, PACKET_SIZE)

	d.conf = conf
	d.isWriter = isWriter
	d.isReader = !isWriter

	return d
}

// GetProduct returns a device product name
func (d *EnttecDMXUSBProController) GetProduct() (string, error) {
	return "product", nil
}

// GetSerial returns a device serial number
func (d *EnttecDMXUSBProController) GetSerial() (string, error) {
	return "serial", nil
}

// Connect handles connection to a Enttec DMX USB Pro controller
func (d *EnttecDMXUSBProController) Connect() error {
	s, err := serial.OpenPort(d.conf)
	if err != nil {
		log.Fatal(err)
	}
	d.port = s

	return nil
}

// Disconnect disconnects the usb device
func (d *EnttecDMXUSBProController) Disconnect() error {
	if d.port == nil {
		return nil
	}

	return d.port.Close()
}

// GetChannels gets a copy of all of the channels of a universe
// Returns our internal state of the channels
// In write mode that means whatever we have set so far
// In read mode that means whatever we read last.
func (d *EnttecDMXUSBProController) GetChannels() ([]byte, error) {
	channels := make([]byte, len(d.channels))

	copy(channels, d.channels)

	return channels, nil
}

// SetChannel sets a single DMX channel value
func (d *EnttecDMXUSBProController) SetChannel(index int16, data byte) error {
	if d.isReader {
		return fmt.Errorf("controller in READ mode must not WRITE")
	}

	if index < 1 || index > DATA_LENGTH {
		return fmt.Errorf("index %d out of range, must be between 1 and %d", index, DATA_LENGTH)
	}

	d.channels[index-1] = data

	return nil
}

// Render sends channel data to fixtures
func (d *EnttecDMXUSBProController) Render() error {

	if d.isReader {
		return fmt.Errorf("controller in READ mode must not WRITE")
	}

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

	if _, err := d.port.Write(d.packet); err != nil {
		return err
	}

	return nil
}

func (d *EnttecDMXUSBProController) Read() error {
	_, err := d.port.Read(d.packet)
	if err != nil {
		log.Fatal(err)
	}
	// TODO: extract channels from packet
	return err
}
