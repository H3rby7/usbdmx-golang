package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	usbdmxcontroller "github.com/H3rby7/usbdmx-golang/controller"
	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro"
	"github.com/tarm/serial"
)

var controller usbdmxcontroller.USBDMXController

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
			controller.Clear()
			controller.Commit()
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
	controller = dmxusbpro.NewEnttecDMXUSBProController(config, true)
	if err := controller.Connect(); err != nil {
		log.Fatalf("Failed to connect DMX Controller: %s", err)
	}
	handleCancel()

	// Open shutter
	controller.Stage(10, 255)
	// Open dimmer
	controller.Stage(11, 75)

	// Create an array of colours for our fixture to switch between (assume RGB)
	colours := [][]byte{
		{255, 0, 0},
		{255, 255, 0},
		{0, 255, 0},
		{0, 255, 255},
		{0, 0, 255},
		{255, 0, 255},
	}
	// Channels for RGB start at this Channel.
	rgbStartChannel := int16(6)

	// Create a loop that will cycle through all of the colours defined in the "colours"
	// array and set the channels on our controller. Once the channels have been set their
	// values are ouptut to stdout. Wait 2 seconds between updating our new channels
	for i := 0; true; i++ {
		colour := colours[i%len(colours)]
		controller.Stage(rgbStartChannel, colour[0])
		controller.Stage(rgbStartChannel+1, colour[1])
		controller.Stage(rgbStartChannel+2, colour[2])

		chans, _ := controller.GetStage()
		r := chans[rgbStartChannel]
		g := chans[rgbStartChannel+1]
		b := chans[rgbStartChannel+2]

		log.Printf("CHAN %d -> %d \t CHAN %d -> %d \t CHAN %d -> %d", rgbStartChannel, r, rgbStartChannel+1, g, rgbStartChannel+2, b)

		if err := controller.Commit(); err != nil {
			log.Fatalf("Failed to commit output: %s", err)
		}

		time.Sleep(time.Second * 2)
	}
}
