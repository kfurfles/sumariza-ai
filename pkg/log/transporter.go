package log

// Transporter defines the interface for log output destinations.
// Implementations can write logs to stdout, files, Loki, etc.
type Transporter interface {
	// Name returns the identifier for this transporter.
	Name() string

	// Write sends a log entry to the destination.
	// Returns an error if the write fails.
	Write(entry Entry) error

	// Close releases any resources held by the transporter.
	// After Close is called, Write should not be called.
	Close() error
}
