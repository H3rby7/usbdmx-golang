package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro"
	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro/messages"
	"github.com/tarm/serial"
)

var readController *dmxusbpro.EnttecDMXUSBProController
var writeController *dmxusbpro.EnttecDMXUSBProController
var isRunning bool

func main() {
	baud := flag.Int("baud", 57600, "Baudrate for the devices")
	readerName := flag.String("reader", "", "Input interface (e.g. COM4 OR /dev/tty1.usbserial)")
	writerName := flag.String("writer", "", "Output interface (e.g. COM5 OR /dev/tty2.usbserial)")
	readInterval := flag.Int("read-interval", 30, "Interval between reads in MS")
	writeInterval := flag.Int("write-interval", 30, "Interval between writes in MS")
	flag.Parse()
	go read(&serial.Config{Name: *readerName, Baud: *baud}, *readInterval)
	go write(&serial.Config{Name: *writerName, Baud: *baud}, *writeInterval)

	c := make(chan os.Signal, 1)
	signal.Notify(c,
		// https://www.gnu.org/software/libc/manual/html_node/Termination-Signals.html
		syscall.SIGTERM, // "the normal way to politely ask a program to terminate"
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGQUIT, // Ctrl-\
		syscall.SIGHUP,  // "terminal is disconnected"
	)
	<-c
	log.Printf("Stopping...")
	isRunning = false
	writeController.ClearStage()
	writeController.Commit()
	log.Printf("... clearing DMX output (1 second) ...")
	time.Sleep(time.Second * 1)
	writeController.Disconnect()
	log.Printf("... reading for 3 more seconds ...")
	time.Sleep(time.Second * 3)
	readController.Disconnect()
	log.Printf("Finished.")
}

func read(config *serial.Config, interval int) {
	// Create a controller and connect to it
	readController = dmxusbpro.NewEnttecDMXUSBProController(config, 16, false)
	if err := readController.Connect(); err != nil {
		log.Fatalf("Failed to connect DMX Controller: %s", err)
	}
	readController.SwitchReadMode(1)
	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	go readController.OnDMXChange(c, 30)
	for msg := range c {
		cs, err := messages.ToChangeSet(msg)
		if err != nil {
			log.Printf("Could not convert to changeset, but read \tlabel=%v \tdata=%v", msg.GetLabel(), msg.GetPayload())
		} else {
			log.Printf("Changeset is \t%v", cs)
		}
	}
}

func write(config *serial.Config, interval int) {
	// Create a controller and connect to it
	writeController = dmxusbpro.NewEnttecDMXUSBProController(config, 16, true)
	if err := writeController.Connect(); err != nil {
		log.Fatalf("Failed to connect DMX Controller: %s", err)
	}
	isRunning = true

	// Open shutter
	writeController.Stage(10, 255)
	// Open dimmer
	writeController.Stage(11, 75)

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

	// Constantly change
	for i := 0; isRunning; i++ {
		colour := colours[i%len(colours)]
		writeController.Stage(rgbStartChannel, colour[0])
		writeController.Stage(rgbStartChannel+1, colour[1])
		writeController.Stage(rgbStartChannel+2, colour[2])

		chans := writeController.GetStage()
		r := chans[rgbStartChannel]
		g := chans[rgbStartChannel+1]
		b := chans[rgbStartChannel+2]

		log.Printf("CHAN %d -> %d \t CHAN %d -> %d \t CHAN %d -> %d", rgbStartChannel, r, rgbStartChannel+1, g, rgbStartChannel+2, b)

		if err := writeController.Commit(); err != nil {
			log.Fatalf("Failed to commit output: %s", err)
		}

		time.Sleep(time.Millisecond * time.Duration(interval))
	}
}
