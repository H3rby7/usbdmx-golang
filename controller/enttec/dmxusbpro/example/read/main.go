package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro"
	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro/messages"
	"github.com/tarm/serial"
)

var controller *dmxusbpro.EnttecDMXUSBProController

func handleCancel() {
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		// https://www.gnu.org/software/libc/manual/html_node/Termination-Signals.html
		syscall.SIGTERM, // "the normal way to politely ask a program to terminate"
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGQUIT, // Ctrl-\
		syscall.SIGHUP,  // "terminal is disconnected"
	)
	go func() {
		for range c {
			controller.Disconnect()
			break
		}
	}()
}

func main() {
	baud := flag.Int("baud", 57600, "Baudrate for the device")
	name := flag.String("name", "", "Input interface (e.g. COM4 OR /dev/tty.usbserial)")
	flag.Parse()

	// Create a configuration from our flags
	config := &serial.Config{Name: *name, Baud: *baud}

	// Create a controller and connect to it
	controller = dmxusbpro.NewEnttecDMXUSBProController(config, false)
	if err := controller.Connect(); err != nil {
		log.Fatalf("Failed to connect DMX Controller: %s", err)
	}
	handleCancel()
	controller.ReadOnChangeOnly()
	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	go controller.OnDMXChange(c)
	for msg := range c {
		cs, err := messages.ToChangeSet(msg)
		log.Printf("Read \tlabel=%v \tdata=%v", msg.GetLabel(), msg.GetPayload())
		if err == nil {
			log.Printf("Read \t%v", cs)
		}
	}
}
