package usbdmxgolang

type DMXController interface {
	// Connect the device
	Connect() (err error)
	// Disconnect the device
	Disconnect() (err error)
	// Returns the device name
	GetName() string
	// Read raw from DMX
	Read(buf []byte) (int, error)
	// Write raw to DMX
	Write(buf []byte) (int, error)
	// Stage DMX value
	Stage(channel int16, value byte) error
	// Commit the staged values to the DMX network
	Commit() error
	// Get staged/last read DMX values
	GetStage() []byte
	// Clear all staged values to 0
	ClearStage()
	// Set debug printing ON/OFF
	SetDebug(bool)
}
