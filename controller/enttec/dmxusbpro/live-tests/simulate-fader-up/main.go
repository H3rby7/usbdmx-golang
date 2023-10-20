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
var c chan os.Signal

func main() {
	baud := flag.Int("baud", 57600, "Baudrate for the devices")
	readerName := flag.String("reader", "", "Input interface (e.g. COM4 OR /dev/tty1.usbserial)")
	writerName := flag.String("writer", "", "Output interface (e.g. COM5 OR /dev/tty2.usbserial)")
	readInterval := flag.Int("read-interval", 30, "Interval between reads in MS")
	writeInterval := flag.Int("write-interval", 30, "Interval between writes in MS")
	flag.Parse()
	go read(&serial.Config{Name: *readerName, Baud: *baud}, *readInterval)
	go fadeUp(&serial.Config{Name: *writerName, Baud: *baud}, *writeInterval)

	c = make(chan os.Signal, 1)
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
	readController.Disconnect()
	log.Printf("Finished.")
}

func read(config *serial.Config, interval int) {
	// Create a controller and connect to it
	readController = dmxusbpro.NewEnttecDMXUSBProController(config, 4, false)
	if err := readController.Connect(); err != nil {
		log.Fatalf("READER\tFailed to connect DMX Controller: %s", err)
	}
	// readController.SetLogVerbosity(1)
	readController.SwitchReadMode(0)
	c := make(chan messages.EnttecDMXUSBProApplicationMessage)
	go readController.OnDMXChange(c, interval)
	for msg := range c {
		cs, err := messages.ToDMXArray(msg)
		if err != nil {
			log.Printf("READER\tCould not convert to changeset, but read \tlabel=%v \tdata=%v", msg.GetLabel(), msg.GetPayload())
		} else {
			log.Printf("READER\tDMX values are:\t%v", cs)
		}
	}
}

func fadeUp(config *serial.Config, interval int) {
	// Create a controller and connect to it
	writeController = dmxusbpro.NewEnttecDMXUSBProController(config, 4, true)
	if err := writeController.Connect(); err != nil {
		log.Fatalf("WRITER\tFailed to connect DMX Controller: %s", err)
	}
	// writeController.SetLogVerbosity(1)
	isRunning = true
	// The DMX values for the fader
	group := []byte{240, 120, 60}
	// Slowly pull up fader
	for i := 0; isRunning; i++ {
		perc := float32(i) / 255
		vals := []byte{
			0,
			byte(float32(group[0]) * perc),
			byte(float32(group[1]) * perc),
			byte(float32(group[2]) * perc),
		}
		writeController.Stage(1, vals[1])
		writeController.Stage(2, vals[2])
		writeController.Stage(3, vals[3])

		log.Printf("WRITER\tDMX values are:\t%v", vals)

		if err := writeController.Commit(); err != nil {
			log.Fatalf("WRITER\tFailed to commit output: %s", err)
		}

		time.Sleep(time.Millisecond * time.Duration(interval))
		if i > 254 {
			log.Print("WRITER\tReached fader MAX, exiting.")
			break
		}
	}
	signal.Stop(c)
	close(c)
}
