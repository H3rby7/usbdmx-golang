package dmxusbpro

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro/messages"
	"github.com/tarm/serial"
)

// Prefix for any log and error messages
const ENTTEC_DMX_USB_PRO_LOG_PREFIX = "EDUP"

/*
Controller for Enttec DMX USB Pro device to handle communication

A controller is safe for concurrent use by multiple goroutines.
In particular 'Disconnect' may be called while 'OnDMXChange' is running in another goroutine.
*/
type EnttecDMXUSBProController struct {
	// Guards 'channels'
	stageMu sync.Mutex
	// Holds DMX data, as DMX starts with channel '1' the index '0' is unused.
	channels []byte

	isWriter bool
	isReader bool
	// Is the widget in 'only read changes'-mode (as opposed to read everything)
	readOnChange atomic.Bool

	// Guards 'port' and 'readErr'
	mu sync.Mutex
	// The serial port, nil if not connected
	port io.ReadWriteCloser
	// Error that ended the last 'OnDMXChange' loop
	readErr error

	conf         *serial.Config
	logVerbosity atomic.Uint32
}

// Helper function for creating a new DMX USB PRO controller
func NewEnttecDMXUSBProController(conf *serial.Config, dmxChannelCount int, isWriter bool) *EnttecDMXUSBProController {
	d := &EnttecDMXUSBProController{}
	d.channels = make([]byte, dmxChannelCount+1)

	d.conf = conf
	d.isWriter = isWriter
	d.isReader = !isWriter

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

Discards any data left in the serial buffers, e.g. DMX packets the widget sent
while no one was reading, so reading starts with fresh data.

Succeeded if no error is returned
*/
func (d *EnttecDMXUSBProController) Connect() error {
	s, err := serial.OpenPort(d.conf)
	if err != nil {
		return err
	}
	return d.attach(s)
}

// A port that can discard buffered data, like 'serial.Port'
type flusher interface {
	Flush() error
}

/*
Use the given port for communication. Used by 'Connect' and by tests to inject a fake port.

Flushes the port if it supports it. If that fails, the port gets closed and the error returned.
*/
func (d *EnttecDMXUSBProController) attach(port io.ReadWriteCloser) error {
	if f, ok := port.(flusher); ok {
		if err := f.Flush(); err != nil {
			port.Close()
			return d.errorf("could not flush serial port: %w", err)
		}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.port = port
	return nil
}

// Returns the port currently in use, nil if not connected.
func (d *EnttecDMXUSBProController) currentPort() io.ReadWriteCloser {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.port
}

/*
	Close the serial connection to Enttec DMX USB PRO Widget

Succeeded if no error is returned.
Returns an error if the controller is not connected (e.g. when called twice).

A running 'OnDMXChange' loop ends quietly and closes its channel.
*/
func (d *EnttecDMXUSBProController) Disconnect() error {
	d.mu.Lock()
	port := d.port
	d.port = nil
	d.mu.Unlock()
	if port == nil {
		return d.errorf("not connected")
	}
	// Close outside the lock, closing is what unblocks a pending read.
	return port.Close()
}

// Gets a copy of all staged channel values
func (d *EnttecDMXUSBProController) GetStage() []byte {
	d.stageMu.Lock()
	defer d.stageMu.Unlock()
	channels := make([]byte, len(d.channels))
	copy(channels, d.channels)
	return channels
}

/*
Prepare a channel to be changed to the given value

Valid channels are 1 to the 'dmxChannelCount' the controller was created with.

Note: This does not send out the changes, you must call the 'Commit' method to apply the stage live.
*/
func (d *EnttecDMXUSBProController) Stage(channel int16, value byte) error {
	if !d.isWriter {
		return d.errorf("controller is not in WRITE mode")
	}
	d.stageMu.Lock()
	defer d.stageMu.Unlock()
	// index '0' is unused
	highestChannel := int16(len(d.channels) - 1)
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
	msg := messages.NewEnttecDMXUSBProApplicationMessage(messages.LABEL_OUTPUT_ONLY_SEND_DMX_PACKET_REQUEST, d.GetStage())
	return d.writeMessage(msg)
}

// Set all values of the staged channels to '0'
func (d *EnttecDMXUSBProController) ClearStage() {
	d.stageMu.Lock()
	defer d.stageMu.Unlock()
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

Any other value returns an error.
*/
func (d *EnttecDMXUSBProController) SwitchReadMode(changesOnly byte) error {
	if changesOnly > 1 {
		return d.errorf("invalid value, only 0 and 1 are allowed, but got '%d'", changesOnly)
	}
	if !d.isReader {
		return d.errorf("controller is not in READ mode")
	}
	msg := messages.NewEnttecDMXUSBProApplicationMessage(messages.LABEL_RECEIVE_DMX_ON_CHANGE, []byte{changesOnly})
	if err := d.writeMessage(msg); err != nil {
		return err
	}
	d.readOnChange.Store(true)
	return nil
}

/*
Expose serial read to be used directly

Follows the 'io.Reader' contract, so returns 0 bytes and an error if not connected or not in READ mode.
*/
func (d *EnttecDMXUSBProController) Read(buf []byte) (int, error) {
	port := d.currentPort()
	if port == nil {
		return 0, d.errorf("not connected")
	}
	if !d.isReader {
		return 0, d.errorf("controller is not in READ mode")
	}
	return d.readFrom(port, buf)
}

func (d *EnttecDMXUSBProController) readFrom(port io.Reader, buf []byte) (int, error) {
	n, err := port.Read(buf)
	if n > 0 {
		d.printf(2, "Read %d bytes:\t%v", n, buf[0:n])
	}
	return n, err
}

/*
Expose serial write to be used directly

Follows the 'io.Writer' contract, so returns 0 bytes and an error if not connected.
*/
func (d *EnttecDMXUSBProController) Write(buf []byte) (int, error) {
	port := d.currentPort()
	if port == nil {
		return 0, d.errorf("not connected")
	}
	n, err := port.Write(buf)
	d.printf(2, "Wrote %d bytes:\t%v", n, buf[0:n])
	return n, err
}

/*
Start routine to read from DMX and get the results back via channel

Requires a successful call of 'SwitchReadMode(1)' before, panics otherwise.

The loop ends, and 'c' gets closed, when:

* the controller gets disconnected ('Disconnect'), the loop then ends quietly;

* reading from the serial port fails, the error is then available via 'LastReadError'.

Use 'OnDMXChangeContext' to stop the loop on purpose and get the error returned directly.

The messages are delivered raw. Use 'messages.ToChangeSet' to decode them,
mind that its keys are DMX slot indices (key '0' is the start code), not channels.

Example useage:

	c := make(chan messages.EnttecDMXUSBProApplicationMessage) // create channel
	go controller.OnDMXChange(c, 15) // start routine
	for msg := range c { ... } // handle incoming data, ends after 'Disconnect'
	if err := controller.LastReadError(); err != nil { ... }
*/
func (d *EnttecDMXUSBProController) OnDMXChange(c chan messages.EnttecDMXUSBProApplicationMessage, readIntervalMS int) {
	if !d.readOnChange.Load() {
		d.panicf("controller is not in READ ON CHANGE mode!")
	}
	defer close(c)
	d.setReadErr(nil)
	// Store the error before 'c' gets closed, so it is available once a 'range c' ends.
	d.setReadErr(d.readChanges(context.Background(), c, readIntervalMS))
}

/*
Like 'OnDMXChange', but can be stopped via 'ctx' and returns the error that ended the loop.

Returns:

* nil, if the controller got disconnected ('Disconnect');

* ctx.Err(), if 'ctx' is done;

* an error, if reading from the serial port fails, or 'SwitchReadMode(1)' wasn't called successfully before.

'c' is closed in any case, once this function returns.

Note: a pending read from the serial port can't be interrupted by 'ctx'.
'Disconnect' closes the port, which unblocks a pending read on most systems.
Otherwise the loop ends with the next read that returns, so consider setting a 'ReadTimeout' in the 'serial.Config'.
Read timeouts (an empty read, or 'io.EOF' without data) don't end the loop.

Example useage:

	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	go func() {
		if err := controller.OnDMXChangeContext(ctx, c, 15); err != nil { ... }
	}()
	for msg := range c { ... }
*/
func (d *EnttecDMXUSBProController) OnDMXChangeContext(ctx context.Context, c chan<- messages.EnttecDMXUSBProApplicationMessage, readIntervalMS int) error {
	defer close(c)
	return d.readChanges(ctx, c, readIntervalMS)
}

/*
Returns the error that ended the last 'OnDMXChange' loop.

Returns nil while the loop is running, and if it ended because of 'Disconnect'.
*/
func (d *EnttecDMXUSBProController) LastReadError() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.readErr
}

func (d *EnttecDMXUSBProController) setReadErr(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.readErr = err
}

// The read loop of 'OnDMXChange' and 'OnDMXChangeContext', without closing 'c'
func (d *EnttecDMXUSBProController) readChanges(ctx context.Context, c chan<- messages.EnttecDMXUSBProApplicationMessage, readIntervalMS int) error {
	if !d.readOnChange.Load() {
		return d.errorf("controller is not in READ ON CHANGE mode, call 'SwitchReadMode(1)' first")
	}
	port := d.currentPort()
	if port == nil {
		return d.errorf("not connected")
	}
	readInterval := time.Millisecond * time.Duration(readIntervalMS)
	// Buffer used for reading fresh data
	readBuf := make([]byte, messages.MAXIMUM_MESSAGE_LENGTH)
	// Buffer containing old, yet unused data
	oldBuf := make([]byte, 0)
	// Detected messages
	var msgs []messages.EnttecDMXUSBProApplicationMessage
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := d.readFrom(port, readBuf)
		if d.currentPort() != port {
			// Disconnected (or reconnected) while reading, so the read error (if any) is expected
			d.printf(1, "Stopped reading, as the controller got disconnected")
			return nil
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return d.errorf("error reading from serial: %w", err)
		}
		// Combine newly read data with yet unused data
		combined := append(oldBuf, readBuf[:n]...)
		// Try to extract valid messages
		msgs, oldBuf = Extract(combined)
		for _, msg := range msgs {
			d.printf(1, "Read \tlabel=%v \tdata=%v", msg.GetLabel(), msg.GetPayload())
			select {
			case c <- msg:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if len(oldBuf) > messages.MAXIMUM_MESSAGE_LENGTH {
			dropOldDataBefore := len(oldBuf) - messages.MAXIMUM_MESSAGE_LENGTH
			d.printf(1, "Dropping old, unused data:\t%v", oldBuf[:dropOldDataBefore])
			oldBuf = oldBuf[dropOldDataBefore:]
		}
		if readInterval > 0 {
			timer := time.NewTimer(readInterval)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
		}
	}
}

/*
Set log verbosity

0 = no logging

1 = message logging

2 = byte logging

Panics on any other value.
*/
func (d *EnttecDMXUSBProController) SetLogVerbosity(verbosity uint8) {
	if verbosity > 2 {
		d.panicf("invalid value, only 0, 1 and 2 are allowed, but got '%d'", verbosity)
	}
	d.logVerbosity.Store(uint32(verbosity))
}

func (d *EnttecDMXUSBProController) printf(level uint8, format string, v ...any) {
	if d.logVerbosity.Load() == uint32(level) {
		log.Printf(ENTTEC_DMX_USB_PRO_LOG_PREFIX+": "+format, v...)
	}
}

func (d *EnttecDMXUSBProController) errorf(format string, v ...any) error {
	return fmt.Errorf(ENTTEC_DMX_USB_PRO_LOG_PREFIX+": "+format, v...)
}

func (d *EnttecDMXUSBProController) panicf(format string, v ...any) {
	log.Panicf(ENTTEC_DMX_USB_PRO_LOG_PREFIX+": "+format, v...)
}
