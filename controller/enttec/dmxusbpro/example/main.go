package main

import (
	"flag"
	"log"
	"time"

	usbdmxconfig "github.com/H3rby7/usbdmx-golang/config"
	usbdmxcontroller "github.com/H3rby7/usbdmx-golang/controller"
	"github.com/H3rby7/usbdmx-golang/controller/enttec/dmxusbpro"
)

func main() {
	var controller usbdmxcontroller.USBDMXController
	// Constants, these should really be defined in the module and will be
	// as of the next release
	vid := uint16(0x0403)
	pid := uint16(0x6001)
	outputInterfaceID := flag.Int("output-id", 0, "Output interface ID for device")
	inputInterfaceID := flag.Int("input-id", 0, "Input interface ID for device")
	debugLevel := flag.Int("debug", 0, "Debug level for USB context")
	flag.Parse()

	// Create a configuration from our flags
	config := usbdmxconfig.NewConfig(vid, pid, *outputInterfaceID, *inputInterfaceID, *debugLevel)

	// Create a controller and connect to it
	controller = dmxusbpro.NewEnttecDMXUSBProController(config)
	if err := controller.Connect(); err != nil {
		log.Fatalf("Failed to connect DMX Controller: %s", err)
	}

	// Open any shutters / dimmers as needed
	controller.SetChannel(10, 255)
	controller.SetChannel(11, 75)

	// Create an array of colours for our fixture to switch between (assume RGB)
	colours := [][]byte{
		[]byte{255, 0, 0},
		[]byte{255, 255, 0},
		[]byte{0, 255, 0},
		[]byte{0, 255, 255},
		[]byte{0, 0, 255},
		[]byte{255, 0, 255},
	}
	// Channels for RGB start at this Channel.
	rgbStartChannel := int16(6)

	// Create a go routine that will ensure our controller keeps sending data
	// to our fixture with a short delay. No delay, or too much delay, may cause
	// flickering in fixtures. Check the specification of your fixtures and controller
	go func(c *usbdmxcontroller.USBDMXController) {
		for {
			if err := controller.Render(); err != nil {
				log.Fatalf("Failed to render output: %s", err)
			}

			time.Sleep(30 * time.Millisecond)
		}
	}(&controller)

	// Create a loop that will cycle through all of the colours defined in the "colours"
	// array and set the channels on our controller. Once the channels have been set their
	// values are ouptut to stdout. Wait 2 seconds between updating our new channels
	for i := 0; true; i++ {
		colour := colours[i%len(colours)]
		controller.SetChannel(rgbStartChannel, colour[0])
		controller.SetChannel(rgbStartChannel+1, colour[1])
		controller.SetChannel(rgbStartChannel+2, colour[2])

		r, _ := controller.GetChannel(rgbStartChannel)
		g, _ := controller.GetChannel(rgbStartChannel + 1)
		b, _ := controller.GetChannel(rgbStartChannel + 2)

		log.Printf("CHAN %d -> %d \t CHAN %d -> %d \t CHAN %d -> %d", rgbStartChannel, r, rgbStartChannel+1, g, rgbStartChannel+2, b)
		time.Sleep(time.Second * 2)
	}
}
