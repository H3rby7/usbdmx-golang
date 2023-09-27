package usbdmxcontroller

// Generic interface for all USB DMX controllers
type USBDMXController interface {
	Connect() (err error)
	GetSerial() (info string, err error)
	GetProduct() (info string, err error)
	SetChannel(channel int16, value byte) error
	GetChannel(channel int16) (byte, error)
	GetChannels() ([]byte, error)
	Render() error
}
