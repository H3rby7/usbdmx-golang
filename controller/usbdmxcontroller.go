package usbdmxcontroller

// Generic interface for all USB DMX controllers
type USBDMXController interface {
	// Connect the device
	Connect() (err error)
	// Disconnect the device
	Disconnect() (err error)
	// Returns the device name
	GetName() string
	// Stage DMX value
	Stage(channel int16, value byte) error
	// Get staged/last read DMX values
	GetStage() ([]byte, error)
	// Commit the staged values to the DMX network
	Commit() error
	// Clear all staged values to 0
	Clear()
}
