package dmxusbpro

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro/messages"
	"github.com/tarm/serial"
)

// How long tests wait for something that is expected to happen 'immediately'
const testTimeout = 2 * time.Second

var errFakeRead = errors.New("fake read error")

/*
fakePort replaces the serial port in tests.

Read returns the chunks sent to 'data'. Once 'data' is closed, Read returns 'errFakeRead'.
Close unblocks a pending Read, which then returns an error, like a real port that gets closed.
If 'timeout' is set, Read returns (0, io.EOF) when no data arrives within that time,
like a tarm/serial port on Linux with a 'ReadTimeout'.
*/
type fakePort struct {
	data      chan []byte
	timeout   time.Duration
	closed    chan struct{}
	closeOnce sync.Once

	mu      sync.Mutex
	written [][]byte
	// returned by Read before anything else, like data left in the serial buffer from a previous session
	leftover [][]byte
}

// A fakePort that, like a 'serial.Port', can discard its buffered data
type flushablePort struct {
	*fakePort
	flushErr error
	flushes  int
}

func (p *flushablePort) Flush() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.flushes++
	if p.flushErr != nil {
		return p.flushErr
	}
	p.leftover = nil
	return nil
}

func (p *fakePort) isClosed() bool {
	select {
	case <-p.closed:
		return true
	default:
		return false
	}
}

func newFakePort() *fakePort {
	return &fakePort{
		data:   make(chan []byte),
		closed: make(chan struct{}),
	}
}

func (p *fakePort) Read(b []byte) (int, error) {
	p.mu.Lock()
	if len(p.leftover) > 0 {
		chunk := p.leftover[0]
		p.leftover = p.leftover[1:]
		p.mu.Unlock()
		return copy(b, chunk), nil
	}
	p.mu.Unlock()
	var timeout <-chan time.Time
	if p.timeout > 0 {
		timeout = time.After(p.timeout)
	}
	select {
	case chunk, ok := <-p.data:
		if !ok {
			return 0, errFakeRead
		}
		return copy(b, chunk), nil
	case <-p.closed:
		return 0, errors.New("fake port closed")
	case <-timeout:
		return 0, io.EOF
	}
}

func (p *fakePort) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.written = append(p.written, append([]byte(nil), b...))
	return len(b), nil
}

func (p *fakePort) Close() error {
	p.closeOnce.Do(func() { close(p.closed) })
	return nil
}

func (p *fakePort) writeCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.written)
}

// Create a reader on a fake port that is in 'read on change' mode
func newFakeReader(t *testing.T) (*EnttecDMXUSBProController, *fakePort) {
	t.Helper()
	port := newFakePort()
	d := NewEnttecDMXUSBProController(&serial.Config{Name: "fake"}, 16, false)
	d.attach(port)
	if err := d.SwitchReadMode(1); err != nil {
		t.Fatalf("expected no error switching read mode, but got %v", err)
	}
	return d, port
}

// A valid 'Received DMX Change Of State Packet' with slot 0 set to 'value'
func changeOfStatePacket(t *testing.T, value byte) []byte {
	t.Helper()
	msg := messages.NewEnttecDMXUSBProApplicationMessage(
		messages.LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET,
		[]byte{0, 1, 0, 0, 0, 0, value},
	)
	packet, err := msg.ToBytes()
	if err != nil {
		t.Fatalf("could not create packet: %v", err)
	}
	return packet
}

// Call f and turn a panic into an error, so a test can fail instead of crash
func catchPanic(f func() error) (err error, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
			panicked = true
		}
	}()
	return f(), false
}

// Wait until 'c' is closed, ignoring (but counting) messages still delivered
func awaitClosed(t *testing.T, c <-chan messages.EnttecDMXUSBProApplicationMessage) {
	t.Helper()
	deadline := time.After(testTimeout)
	for {
		select {
		case _, ok := <-c:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatalf("expected channel to be closed within %v", testTimeout)
		}
	}
}

func TestStageValidChannels(t *testing.T) {
	d := NewEnttecDMXUSBProController(&serial.Config{}, 16, true)
	for _, channel := range []int16{1, 16} {
		err, panicked := catchPanic(func() error { return d.Stage(channel, 255) })
		if panicked || err != nil {
			t.Errorf("Stage(%d): expected no error, but got %v", channel, err)
		}
	}
	stage := d.GetStage()
	if stage[1] != 255 || stage[16] != 255 {
		t.Errorf("expected channels 1 and 16 to be staged, but stage is %v", stage)
	}
}

func TestStageInvalidChannels(t *testing.T) {
	d := NewEnttecDMXUSBProController(&serial.Config{}, 16, true)
	for _, channel := range []int16{0, -1, 17} {
		err, panicked := catchPanic(func() error { return d.Stage(channel, 255) })
		if panicked {
			t.Errorf("Stage(%d): expected an error, but got %v", channel, err)
			continue
		}
		if err == nil {
			t.Errorf("Stage(%d): expected an error, but got none", channel)
			continue
		}
		if !strings.Contains(err.Error(), "between 1 and 16") {
			t.Errorf("Stage(%d): expected error to state 'between 1 and 16', but got %q", channel, err)
		}
	}
}

// Stage, ClearStage, Commit and GetStage from several goroutines. Meant to be run with '-race'.
func TestStageAndCommitConcurrently(t *testing.T) {
	port := newFakePort()
	d := NewEnttecDMXUSBProController(&serial.Config{}, 16, true)
	d.attach(port)
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				_ = d.Stage(int16(1+(g+i)%16), byte(i))
				if i%10 == 0 {
					d.ClearStage()
				}
				if err := d.Commit(); err != nil {
					t.Errorf("expected no error on commit, but got %v", err)
					return
				}
				_ = d.GetStage()
			}
		}(g)
	}
	wg.Wait()
	if port.writeCount() == 0 {
		t.Errorf("expected commits to be written to the port")
	}
}

func TestWriteNotConnectedHasPrefix(t *testing.T) {
	d := NewEnttecDMXUSBProController(&serial.Config{}, 16, true)
	_, err := d.Write([]byte{1})
	if err == nil {
		t.Fatalf("expected an error, but got none")
	}
	if !strings.HasPrefix(err.Error(), ENTTEC_DMX_USB_PRO_LOG_PREFIX+": ") {
		t.Errorf("expected error to start with %q, but got %q", ENTTEC_DMX_USB_PRO_LOG_PREFIX+": ", err)
	}
}

func TestSwitchReadModeInvalidValueReturnsError(t *testing.T) {
	d := NewEnttecDMXUSBProController(&serial.Config{}, 16, false)
	d.attach(newFakePort())
	err, panicked := catchPanic(func() error { return d.SwitchReadMode(2) })
	if panicked || err == nil {
		t.Errorf("expected an error (and no panic), but got %v", err)
	}
}

// Delivers messages, then Disconnect (from another goroutine) ends the loop quietly and closes the channel.
func TestOnDMXChangeClosesChannelOnDisconnect(t *testing.T) {
	d, port := newFakeReader(t)
	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	go d.OnDMXChange(c, 1)

	port.data <- changeOfStatePacket(t, 69)
	select {
	case msg := <-c:
		if msg.GetLabel() != messages.LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET {
			t.Errorf("expected label %d, but got %d", messages.LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET, msg.GetLabel())
		}
	case <-time.After(testTimeout):
		t.Fatalf("expected a message within %v", testTimeout)
	}

	if err := d.Disconnect(); err != nil {
		t.Fatalf("expected no error on disconnect, but got %v", err)
	}
	awaitClosed(t, c)
}

// Disconnect ends the loop quietly, so no error is kept
func TestOnDMXChangeNoErrorAfterDisconnect(t *testing.T) {
	d, _ := newFakeReader(t)
	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	go d.OnDMXChange(c, 1)
	// give the loop time to start reading
	time.Sleep(20 * time.Millisecond)
	if err := d.Disconnect(); err != nil {
		t.Fatalf("expected no error on disconnect, but got %v", err)
	}
	awaitClosed(t, c)
	if err := d.LastReadError(); err != nil {
		t.Errorf("expected no read error after disconnect, but got %v", err)
	}
}

// Any other read error ends the loop, closes the channel and is kept
func TestOnDMXChangeReadErrorClosesChannel(t *testing.T) {
	d, port := newFakeReader(t)
	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	go d.OnDMXChange(c, 1)
	close(port.data)
	awaitClosed(t, c)
	if err := d.LastReadError(); !errors.Is(err, errFakeRead) {
		t.Errorf("expected read error to wrap %v, but got %v", errFakeRead, err)
	}
}

func TestOnDMXChangeContextReturnsReadError(t *testing.T) {
	d, port := newFakeReader(t)
	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	done := make(chan error, 1)
	go func() { done <- d.OnDMXChangeContext(context.Background(), c, 1) }()
	close(port.data)
	awaitClosed(t, c)
	select {
	case err := <-done:
		if !errors.Is(err, errFakeRead) {
			t.Errorf("expected error to wrap %v, but got %v", errFakeRead, err)
		}
	case <-time.After(testTimeout):
		t.Fatalf("expected OnDMXChangeContext to return within %v", testTimeout)
	}
}

func TestOnDMXChangeContextReturnsNilOnDisconnect(t *testing.T) {
	d, port := newFakeReader(t)
	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	done := make(chan error, 1)
	go func() { done <- d.OnDMXChangeContext(context.Background(), c, 1) }()
	port.data <- changeOfStatePacket(t, 1)
	<-c
	go d.Disconnect()
	awaitClosed(t, c)
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected no error, but got %v", err)
		}
	case <-time.After(testTimeout):
		t.Fatalf("expected OnDMXChangeContext to return within %v", testTimeout)
	}
}

// Cancelling stops the loop, even when nobody receives from the channel anymore
func TestOnDMXChangeContextCancelWhileSending(t *testing.T) {
	d, port := newFakeReader(t)
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	done := make(chan error, 1)
	go func() { done <- d.OnDMXChangeContext(ctx, c, 1) }()
	// the loop now blocks sending this message, as nobody receives
	port.data <- changeOfStatePacket(t, 1)
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected %v, but got %v", context.Canceled, err)
		}
	case <-time.After(testTimeout):
		t.Fatalf("expected OnDMXChangeContext to return within %v", testTimeout)
	}
	awaitClosed(t, c)
}

// A read timeout (tarm/serial on Linux returns (0, io.EOF)) is no error; the loop keeps going until cancelled
func TestOnDMXChangeContextReadTimeoutIsNoError(t *testing.T) {
	d, port := newFakeReader(t)
	port.timeout = 5 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	done := make(chan error, 1)
	go func() { done <- d.OnDMXChangeContext(ctx, c, 1) }()
	// several timeouts happen in the meantime
	time.Sleep(50 * time.Millisecond)
	port.data <- changeOfStatePacket(t, 7)
	select {
	case <-c:
	case err := <-done:
		t.Fatalf("expected the loop to keep running, but it returned %v", err)
	case <-time.After(testTimeout):
		t.Fatalf("expected a message within %v", testTimeout)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected %v, but got %v", context.Canceled, err)
		}
	case <-time.After(testTimeout):
		t.Fatalf("expected OnDMXChangeContext to return within %v", testTimeout)
	}
	awaitClosed(t, c)
}

func TestOnDMXChangeContextNotInReadOnChangeMode(t *testing.T) {
	d := NewEnttecDMXUSBProController(&serial.Config{}, 16, false)
	d.attach(newFakePort())
	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	done := make(chan error, 1)
	go func() { done <- d.OnDMXChangeContext(context.Background(), c, 1) }()
	select {
	case err := <-done:
		if err == nil {
			t.Errorf("expected an error, but got none")
		}
	case <-time.After(testTimeout):
		t.Fatalf("expected OnDMXChangeContext to return within %v", testTimeout)
	}
	awaitClosed(t, c)
}

func TestDisconnectTwice(t *testing.T) {
	d, _ := newFakeReader(t)
	if err := d.Disconnect(); err != nil {
		t.Fatalf("expected no error on first disconnect, but got %v", err)
	}
	if err := d.Disconnect(); err == nil {
		t.Errorf("expected an error on second disconnect, but got none")
	}
	if _, err := d.Read(make([]byte, 1)); err == nil {
		t.Errorf("expected an error reading after disconnect, but got none")
	}
}

// io.Reader and io.Writer require 0 <= n <= len(buf), also on error
func TestReadWriteErrorsReturnZeroBytes(t *testing.T) {
	buf := make([]byte, 4)
	notConnected := NewEnttecDMXUSBProController(&serial.Config{}, 16, false)
	if n, err := notConnected.Read(buf); err == nil || n != 0 {
		t.Errorf("Read when not connected: expected (0, error), but got (%d, %v)", n, err)
	}
	if n, err := notConnected.Write(buf); err == nil || n != 0 {
		t.Errorf("Write when not connected: expected (0, error), but got (%d, %v)", n, err)
	}
	writer := NewEnttecDMXUSBProController(&serial.Config{}, 16, true)
	writer.attach(newFakePort())
	if n, err := writer.Read(buf); err == nil || n != 0 {
		t.Errorf("Read when not in READ mode: expected (0, error), but got (%d, %v)", n, err)
	}
}

// A 'Received DMX Packet' (label 5), as the widget sends it in 'send always' mode
func dmxPacket(t *testing.T) []byte {
	t.Helper()
	msg := messages.NewEnttecDMXUSBProApplicationMessage(messages.LABEL_RECEIVED_DMX_PACKET, []byte{1, 0, 0, 0})
	packet, err := msg.ToBytes()
	if err != nil {
		t.Fatalf("could not create packet: %v", err)
	}
	return packet
}

// Data left in the serial buffer (e.g. from a previous session in 'send always' mode) is not delivered
func TestConnectFlushesLeftoverInput(t *testing.T) {
	port := &flushablePort{fakePort: newFakePort()}
	port.leftover = [][]byte{dmxPacket(t)}
	d := NewEnttecDMXUSBProController(&serial.Config{}, 16, false)
	if err := d.attach(port); err != nil {
		t.Fatalf("expected no error, but got %v", err)
	}
	if port.flushes != 1 {
		t.Errorf("expected port to be flushed once, but was flushed %d times", port.flushes)
	}
	if err := d.SwitchReadMode(1); err != nil {
		t.Fatalf("expected no error switching read mode, but got %v", err)
	}
	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	go d.OnDMXChangeContext(context.Background(), c, 1)
	defer awaitClosed(t, c)
	defer d.Disconnect()

	// send from another goroutine, so a delivered leftover can't block this test
	fresh := changeOfStatePacket(t, 42)
	go func() {
		select {
		case port.data <- fresh:
		case <-port.closed:
		}
	}()
	select {
	case msg := <-c:
		if msg.GetLabel() != messages.LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET {
			t.Errorf("expected first message to have label %d, but got leftover with label %d", messages.LABEL_RECEIVED_DMX_CHANGE_OF_STATE_PACKET, msg.GetLabel())
		}
	case <-time.After(testTimeout):
		t.Fatalf("expected a message within %v", testTimeout)
	}
}

// If flushing fails, connecting fails and the port is closed again
func TestConnectFlushErrorClosesPort(t *testing.T) {
	errFlush := errors.New("fake flush error")
	port := &flushablePort{fakePort: newFakePort(), flushErr: errFlush}
	d := NewEnttecDMXUSBProController(&serial.Config{}, 16, true)
	if err := d.attach(port); !errors.Is(err, errFlush) {
		t.Errorf("expected error to wrap %v, but got %v", errFlush, err)
	}
	if !port.isClosed() {
		t.Errorf("expected port to be closed")
	}
	if _, err := d.Write([]byte{1}); err == nil {
		t.Errorf("expected controller to not be connected")
	}
}
