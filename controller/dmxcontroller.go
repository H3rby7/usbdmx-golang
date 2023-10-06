package dmxcontroller

// Generic interface for all USB DMX controllers
type DMXController interface {
	// Connect the device
	Connect() (err error)
	// Disconnect the device
	Disconnect() (err error)
	// Returns the device name
	GetName() string
	// Stage DMX value
	Stage(channel int16, value byte) error
	// Read from DMX
	Read() error
	// Get staged/last read DMX values
	GetStage() []byte
	// Commit the staged values to the DMX network
	Commit() error
	// Clear all staged values to 0
	Clear()
}
