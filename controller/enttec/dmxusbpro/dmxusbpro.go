package dmxusbpro

import (
	"fmt"
	"log"

	"github.com/tarm/serial"
)

const (
	// "Start of Message" delimiter
	MSG_DELIM_START = 0x7E
	// Maximum DMX channels we can control (fixed size in this implementation)
	DMX_MAX_CHANNELS = 510
	// Data length = Channels + 1 as DMX works 1-indexed [not 0-indexed]
	DMX_DATA_LENGTH = DMX_MAX_CHANNELS + 1
	// Least significant bytes to describe data length
	DMX_DATA_LENGTH_LSB = byte(DMX_DATA_LENGTH & 0xFF)
	// Most significant bytes to describe data length
	DMX_DATA_LENGTH_MSB = byte(DMX_DATA_LENGTH >> 8 & 0xFF)
	// Combined number of bytes BEFORE payload
	NUM_BYTES_BEFORE_PAYLOAD = 4
	// "End of Message" delimiter
	MSG_DELIM_END = 0xE7
	// Combined number of bytes AFTER payload
	NUM_BYTES_AFTER_PAYLOAD = 1
	// Size of full packed, payload, before and after
	PACKET_SIZE = NUM_BYTES_BEFORE_PAYLOAD + DMX_DATA_LENGTH + NUM_BYTES_AFTER_PAYLOAD
)

// Controller for Enttec DMX USB Pro device to handle comms
type EnttecDMXUSBProController struct {
	// Holds DMX data, as DMX starts with channel '1' the index '0' is unused.
	channels []byte
	// Packet containing control data and the DMX data
	packet []byte

	isWriter bool
	isReader bool
	conf     *serial.Config
	port     *serial.Port
}

// Helper function for creating a new DMX USB PRO controller
func NewEnttecDMXUSBProController(conf *serial.Config, isWriter bool) *EnttecDMXUSBProController {
	d := &EnttecDMXUSBProController{}
	d.channels = make([]byte, DMX_DATA_LENGTH)
	d.packet = make([]byte, PACKET_SIZE)

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
func (d *EnttecDMXUSBProController) GetStage() ([]byte, error) {
	channels := make([]byte, len(d.channels))

	copy(channels, d.channels)

	return channels, nil
}

func (d *EnttecDMXUSBProController) Stage(index int16, data byte) error {
	if d.isReader {
		return fmt.Errorf("controller in READ mode must not WRITE")
	}

	if index < 1 || index > DMX_MAX_CHANNELS {
		return fmt.Errorf("index %d out of range, must be between 1 and %d", index, DMX_MAX_CHANNELS)
	}

	d.channels[index] = data

	return nil
}

func (d *EnttecDMXUSBProController) Commit() error {

	if d.isReader {
		return fmt.Errorf("controller in READ mode and must not WRITE")
	}

	if d.port == nil {
		return fmt.Errorf("not connected")
	}

	// ENTTEC USB DMX PRO Start Message
	d.packet[0] = MSG_DELIM_START

	// Set our protocol
	d.packet[1] = 0x06

	d.packet[2] = DMX_DATA_LENGTH_LSB
	d.packet[3] = DMX_DATA_LENGTH_MSB

	// Set DMX Data
	for i := 0; i < DMX_DATA_LENGTH; i++ {
		d.packet[NUM_BYTES_BEFORE_PAYLOAD+i] = d.channels[i]
	}

	// ENTTEC USB DMX PRO End Message
	d.packet[PACKET_SIZE-1] = MSG_DELIM_END

	log.Printf("Writing %v", d.packet)

	_, err := d.port.Write(d.packet)
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

func (d *EnttecDMXUSBProController) Read() error {
	_, err := d.port.Read(d.packet)
	if err != nil {
		log.Fatal(err)
	}
	// TODO: extract channels from packet
	return err
}
