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

// Controller for Enttec DMX USB Pro device to handle comms
type EnttecDMXUSBProController struct {
	// Holds DMX data, as DMX starts with channel '1' the index '0' is unused.
	channels []byte

	isWriter     bool
	isReader     bool
	readOnChange bool
	conf         *serial.Config
	port         *serial.Port
}

// Helper function for creating a new DMX USB PRO controller
func NewEnttecDMXUSBProController(conf *serial.Config, isWriter bool) *EnttecDMXUSBProController {
	d := &EnttecDMXUSBProController{}
	d.channels = make([]byte, DMX_DATA_LENGTH)

	d.conf = conf
	d.isWriter = isWriter
	d.isReader = !isWriter
	d.readOnChange = false

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

	msg := messages.NewEnttecDMXUSBProApplicationMessage(messages.LABEL_OUTPUT_ONLY_SEND_DMX_PACKET_REQUESTS, d.channels)
	packet, err := msg.ToBytes()
	if err != nil {
		return err
	}

	log.Printf("Writing \tlabel=%v \tdata=%v", msg.GetLabel(), msg.GetPayload())

	_, err = d.port.Write(packet)
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
	if !d.isReader {
		return fmt.Errorf("controller is not in READ mode")
	}
	msg := messages.NewEnttecDMXUSBProApplicationMessage(messages.LABEL_RECEIVE_DMX_ON_CHANGE, []byte{1})
	packet, err := msg.ToBytes()
	if err != nil {
		return err
	}
	log.Printf("Writing %v", packet)
	_, err = d.port.Write(packet)
	if err != nil {
		return err
	}
	d.readOnChange = true
	return nil
}

func (d *EnttecDMXUSBProController) Read(buf []byte) (int, error) {
	if !d.isReader {
		return -1, fmt.Errorf("controller is not in READ mode")
	}
	return d.port.Read(buf)
}

func (d *EnttecDMXUSBProController) OnDMXChange(c chan messages.EnttecDMXUSBProApplicationMessage) {
	if !d.readOnChange {
		log.Fatalf("controller is not in READ ON CHANGE mode!")
	}
	ringbuff := [][]byte{
		make([]byte, messages.NUM_BYTES_WRAPPER+messages.MAXIMUM_DATA_LENGTH),
		make([]byte, messages.NUM_BYTES_WRAPPER+messages.MAXIMUM_DATA_LENGTH),
	}
	order := 0
	for {
		// Calculate order of buffers
		order = (order + 1) % 2
		first := order
		second := (order + 1) % 2
		// Read into first buffer
		n, err := d.port.Read(ringbuff[first])
		if err != nil {
			log.Panicf("error reading from serial, %v", err)
		}
		// Combine with the second (older) buffer
		combined := append(ringbuff[first][0:n], ringbuff[second]...)
		// Try to extract a valid message
		msg, err := Extract(combined)
		if err == nil {
			c <- msg
		}
		time.Sleep(time.Millisecond * 30)
	}
}
