package transporters

import (
	"encoding/json"
	"io"
	"os"

	"sumariza-ai/pkg/log"
)

// Stdout writes JSON log entries to stdout (or any io.Writer).
type Stdout struct {
	writer io.Writer
}

// NewStdout creates a new stdout transporter that writes to os.Stdout.
func NewStdout() *Stdout {
	return &Stdout{writer: os.Stdout}
}

// NewStdoutWithWriter creates a stdout transporter with a custom writer.
// Useful for testing.
func NewStdoutWithWriter(w io.Writer) *Stdout {
	return &Stdout{writer: w}
}

// Name returns the transporter identifier.
func (s *Stdout) Name() string {
	return "stdout"
}

// Write marshals the entry to JSON and writes it to stdout.
func (s *Stdout) Write(entry log.Entry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// Append newline for line-delimited JSON
	data = append(data, '\n')

	_, err = s.writer.Write(data)
	return err
}

// Close is a no-op for stdout.
func (s *Stdout) Close() error {
	return nil
}
