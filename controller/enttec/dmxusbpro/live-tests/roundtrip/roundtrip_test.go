//go:build hardware

package roundtrip

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro"
	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro/messages"
	"github.com/tarm/serial"
)

var (
	writerName     = flag.String("writer", "COM8", "serial port of the widget that writes DMX (e.g. COM8 or /dev/ttyUSB0)")
	readerName     = flag.String("reader", "COM7", "serial port of the widget that reads DMX (e.g. COM7 or /dev/ttyUSB1)")
	baud           = flag.Int("baud", 57600, "baud rate for both widgets")
	channelCount   = flag.Int("channels", 512, "number of DMX channels to write and confirm (1 to 512)")
	readIntervalMS = flag.Int("read-interval", 5, "read interval of the reader in milliseconds")
	confirmTimeout = flag.Duration("timeout", 3*time.Second, "how long the reader may take to confirm a written state")
	stopTimeout    = flag.Duration("stop-timeout", 5*time.Second, "how long the reader may take to stop after 'Disconnect'")
)

// The writer commits in this interval while waiting for the reader, like dmx-web does
const commitInterval = 40 * time.Millisecond

// Received DMX values, index 0 is the start code, as in the API's 'received_dmx_array'
type dmxState [513]byte

// Decodes a message from the reader into the state
type decoder func(t *testing.T, msg messages.EnttecDMXUSBProApplicationMessage, state *dmxState)

// Decoder for 'Received DMX Change Of State Packet' (label 9), used with 'SwitchReadMode(1)'
func applyChangeSet(t *testing.T, msg messages.EnttecDMXUSBProApplicationMessage, state *dmxState) {
	t.Helper()
	if msg.GetLabel() != messages.LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET {
		// e.g. 'Received DMX Packet's queued before 'SwitchReadMode(1)' took effect
		t.Logf("READER\tignoring message with label %d (%d bytes)", msg.GetLabel(), len(msg.GetPayload()))
		return
	}
	cs, err := messages.ToChangeSet(msg)
	if err != nil {
		t.Errorf("READER\tcould not decode change set (payload=%v): %v", msg.GetPayload(), err)
		return
	}
	for slot, value := range cs {
		if slot < 0 || slot >= len(state) {
			t.Errorf("READER\tchange set has slot %d, which is outside of a DMX universe: %v", slot, cs)
			continue
		}
		state[slot] = value
	}
}

// Decoder for 'Received DMX Packet' (label 5), used with 'SwitchReadMode(0)'
func applyDMXPacket(t *testing.T, msg messages.EnttecDMXUSBProApplicationMessage, state *dmxState) {
	t.Helper()
	if msg.GetLabel() != messages.LABEL_RECEIVED_DMX_PACKET {
		t.Logf("READER\tignoring message with label %d", msg.GetLabel())
		return
	}
	arr, err := messages.ToDMXArray(msg)
	if err != nil {
		// e.g. a receive overrun flagged by the widget, the next packet will do
		t.Logf("READER\tignoring DMX packet: %v", err)
		return
	}
	copy(state[:], arr)
}

// A reader widget, running 'OnDMXChangeContext' in the background
type reader struct {
	controller *dmxusbpro.EnttecDMXUSBProController
	msgs       chan messages.EnttecDMXUSBProApplicationMessage
	cancel     context.CancelFunc
	// closed once 'OnDMXChangeContext' returned, 'loopErr' is its result
	finished chan struct{}
	loopErr  error

	stopOnce   sync.Once
	stopErr    error
	stopTook   time.Duration
	stopClosed bool
}

/*
Disconnect the reader and wait for its loop to end.

Returns how long it took, whether it ended within 'stopTimeout' and the error of 'OnDMXChangeContext'.
Safe to call more than once, later calls return the results of the first one.
*/
func (r *reader) stop(t *testing.T) (time.Duration, bool, error) {
	r.stopOnce.Do(func() {
		start := time.Now()
		if err := r.controller.Disconnect(); err != nil {
			t.Logf("READER\tdisconnect: %v", err)
		}
		deadline := time.After(*stopTimeout)
		// keep draining, so the loop isn't blocked sending to us
		for {
			select {
			case _, ok := <-r.msgs:
				if ok {
					continue
				}
				<-r.finished
				r.stopErr = r.loopErr
				r.stopTook = time.Since(start)
				r.stopClosed = true
				return
			case <-deadline:
				r.stopTook = time.Since(start)
				// unblock the loop in case it is waiting to send or sleeping
				r.cancel()
				return
			}
		}
	})
	return r.stopTook, r.stopClosed, r.stopErr
}

func validateFlags(t *testing.T) {
	t.Helper()
	if *channelCount < 1 || *channelCount > 512 {
		t.Fatalf("-channels must be between 1 and 512, but is %d", *channelCount)
	}
	if *writerName == *readerName {
		t.Fatalf("-writer and -reader must be different ports, but both are %q", *writerName)
	}
	t.Logf("writer=%s reader=%s baud=%d channels=%d read-interval=%dms", *writerName, *readerName, *baud, *channelCount, *readIntervalMS)
}

// Connect the writer, it gets cleared and disconnected when the test ends
func connectWriter(t *testing.T) *dmxusbpro.EnttecDMXUSBProController {
	t.Helper()
	w := dmxusbpro.NewEnttecDMXUSBProController(&serial.Config{Name: *writerName, Baud: *baud}, *channelCount, true)
	if err := w.Connect(); err != nil {
		t.Fatalf("WRITER\tcould not connect to %s: %v", *writerName, err)
	}
	t.Cleanup(func() {
		w.ClearStage()
		if err := w.Commit(); err != nil {
			t.Logf("WRITER\tcould not clear output: %v", err)
		}
		// give the widget time to send the cleared values, before the port gets closed
		time.Sleep(100 * time.Millisecond)
		if err := w.Disconnect(); err != nil {
			t.Logf("WRITER\tdisconnect: %v", err)
		}
	})
	return w
}

// Connect the reader and start reading, it gets disconnected when the test ends
func connectReader(t *testing.T, changesOnly bool) *reader {
	t.Helper()
	c := dmxusbpro.NewEnttecDMXUSBProController(&serial.Config{Name: *readerName, Baud: *baud}, *channelCount, false)
	if err := c.Connect(); err != nil {
		t.Fatalf("READER\tcould not connect to %s: %v", *readerName, err)
	}
	mode := byte(0)
	if changesOnly {
		mode = 1
	}
	if err := c.SwitchReadMode(mode); err != nil {
		c.Disconnect()
		t.Fatalf("READER\tcould not switch read mode to %d: %v", mode, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &reader{
		controller: c,
		msgs:       make(chan messages.EnttecDMXUSBProApplicationMessage),
		cancel:     cancel,
		finished:   make(chan struct{}),
	}
	go func() {
		r.loopErr = c.OnDMXChangeContext(ctx, r.msgs, *readIntervalMS)
		close(r.finished)
	}()
	t.Cleanup(func() {
		if _, closed, _ := r.stop(t); !closed {
			t.Logf("READER\tdid not stop within %v", *stopTimeout)
		}
		cancel()
	})
	return r
}

// Distinct, non-zero values that change on every channel from one step to the next
func pattern(step int) []byte {
	values := make([]byte, *channelCount+1)
	for channel := 1; channel <= *channelCount; channel++ {
		values[channel] = byte((channel*7+step*31)%255 + 1)
	}
	return values
}

func stage(t *testing.T, w *dmxusbpro.EnttecDMXUSBProController, values []byte) {
	t.Helper()
	for channel := 1; channel < len(values); channel++ {
		if err := w.Stage(int16(channel), values[channel]); err != nil {
			t.Fatalf("WRITER\tcould not stage channel %d: %v", channel, err)
		}
	}
}

// Describe up to 10 channels where the state differs from the expected values, "" if there are none
func mismatches(state *dmxState, expected []byte) string {
	var diffs []string
	count := 0
	for channel := 1; channel < len(expected); channel++ {
		if state[channel] != expected[channel] {
			count++
			if len(diffs) < 10 {
				diffs = append(diffs, fmt.Sprintf("channel %d: want %d, got %d", channel, expected[channel], state[channel]))
			}
		}
	}
	if count == 0 {
		return ""
	}
	return fmt.Sprintf("%d channels differ, e.g. %s", count, strings.Join(diffs, "; "))
}

/*
Write 'expected' and wait until the reader confirms all channels.

Returns the time from the first commit until the reader confirmed.
*/
func writeAndConfirm(t *testing.T, w *dmxusbpro.EnttecDMXUSBProController, r *reader, decode decoder, state *dmxState, expected []byte) time.Duration {
	t.Helper()
	stage(t, w, expected)
	start := time.Now()
	if err := w.Commit(); err != nil {
		t.Fatalf("WRITER\tcould not commit: %v", err)
	}
	ticker := time.NewTicker(commitInterval)
	defer ticker.Stop()
	deadline := time.NewTimer(*confirmTimeout)
	defer deadline.Stop()
	for mismatches(state, expected) != "" {
		select {
		case msg, ok := <-r.msgs:
			if !ok {
				<-r.finished
				t.Fatalf("READER\tstopped unexpectedly: %v", r.loopErr)
			}
			decode(t, msg, state)
		case <-ticker.C:
			if err := w.Commit(); err != nil {
				t.Fatalf("WRITER\tcould not commit: %v", err)
			}
		case <-deadline.C:
			t.Fatalf("READER\tdid not confirm within %v: %s", *confirmTimeout, mismatches(state, expected))
		}
	}
	return time.Since(start)
}

// Write several states, each confirmed by the reader
func runRoundTrip(t *testing.T, changesOnly bool, decode decoder) {
	validateFlags(t)
	w := connectWriter(t)
	r := connectReader(t, changesOnly)
	state := &dmxState{}

	steps := []struct {
		name   string
		values []byte
	}{
		{"pattern 0", pattern(0)},
		{"pattern 1", pattern(1)},
		{"pattern 2", pattern(2)},
	}
	// only the last channel changes
	lastOnly := pattern(2)
	lastOnly[*channelCount] ^= 0xFF
	steps = append(steps, struct {
		name   string
		values []byte
	}{"last channel only", lastOnly})
	// everything back to zero
	steps = append(steps, struct {
		name   string
		values []byte
	}{"all zero", make([]byte, *channelCount+1)})

	for _, step := range steps {
		took := writeAndConfirm(t, w, r, decode, state, step.values)
		t.Logf("%-18s confirmed after %v", step.name, took.Round(time.Millisecond))
	}

	if took, closed, err := r.stop(t); !closed {
		t.Errorf("READER\tdid not stop within %v after 'Disconnect'", *stopTimeout)
	} else if err != nil {
		t.Errorf("READER\tstopped with error: %v", err)
	} else {
		t.Logf("reader stopped %v after 'Disconnect'", took.Round(time.Millisecond))
	}
}

// Reader uses 'Receive DMX on Change' (SwitchReadMode(1)) and decodes change sets
func TestRoundTripChangesOnly(t *testing.T) {
	runRoundTrip(t, true, applyChangeSet)
}

// Reader gets every DMX packet (SwitchReadMode(0))
func TestRoundTripSendAlways(t *testing.T) {
	runRoundTrip(t, false, applyDMXPacket)
}

/*
'Disconnect' stops a reader while the DMX input doesn't change.

In 'changes only' mode the widget sends nothing when the input is static,
so the read loop is blocked in a pending serial read when 'Disconnect' is called.
*/
func TestReaderStopsOnDisconnectWhileIdle(t *testing.T) {
	validateFlags(t)
	w := connectWriter(t)
	r := connectReader(t, true)
	state := &dmxState{}
	writeAndConfirm(t, w, r, applyChangeSet, state, pattern(0))

	// let the line settle, until the reader receives nothing anymore
	quiet := 300 * time.Millisecond
	settle := time.NewTimer(quiet)
	deadline := time.After(*confirmTimeout)
	for settled := false; !settled; {
		select {
		case msg, ok := <-r.msgs:
			if !ok {
				<-r.finished
				t.Fatalf("READER\tstopped unexpectedly: %v", r.loopErr)
			}
			applyChangeSet(t, msg, state)
			settle.Reset(quiet)
		case <-settle.C:
			settled = true
		case <-deadline:
			t.Fatalf("READER\tkept receiving changes for %v, while the input should be static", *confirmTimeout)
		}
	}

	took, closed, err := r.stop(t)
	if !closed {
		t.Fatalf("READER\tdid not stop within %v after 'Disconnect' while idle, closing the port did not unblock the pending read", *stopTimeout)
	}
	if err != nil {
		t.Errorf("READER\tstopped with error: %v", err)
	}
	t.Logf("reader stopped %v after 'Disconnect' while idle", took.Round(time.Millisecond))
}
