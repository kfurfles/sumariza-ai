package transporters

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"sumariza-ai/pkg/log"
)

func TestStdout_ImplementsTransporter(t *testing.T) {
	var _ log.Transporter = &Stdout{}
}

func TestStdout_Name_ReturnsStdout(t *testing.T) {
	s := NewStdout()
	if s.Name() != "stdout" {
		t.Errorf("Name() = %q, want %q", s.Name(), "stdout")
	}
}

func TestStdout_Write_OutputsJSON(t *testing.T) {
	var buf bytes.Buffer
	s := NewStdoutWithWriter(&buf)

	entry := log.Entry{
		Timestamp: time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC),
		Level:     log.Info,
		Message:   "test message",
	}

	err := s.Write(entry)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	output := buf.String()

	// Should be valid JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	// Check fields
	if result["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", result["level"])
	}
	if result["msg"] != "test message" {
		t.Errorf("msg = %v, want 'test message'", result["msg"])
	}
}

func TestStdout_Write_AppendsNewline(t *testing.T) {
	var buf bytes.Buffer
	s := NewStdoutWithWriter(&buf)

	entry := log.Entry{
		Timestamp: time.Now(),
		Level:     log.Info,
		Message:   "test",
	}

	s.Write(entry)

	if !strings.HasSuffix(buf.String(), "\n") {
		t.Error("output should end with newline")
	}
}

func TestStdout_Write_MultipleEntries_EachOnNewLine(t *testing.T) {
	var buf bytes.Buffer
	s := NewStdoutWithWriter(&buf)

	s.Write(log.Entry{Timestamp: time.Now(), Level: log.Info, Message: "first"})
	s.Write(log.Entry{Timestamp: time.Now(), Level: log.Error, Message: "second"})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestStdout_Close_ReturnsNil(t *testing.T) {
	s := NewStdout()
	err := s.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestNewStdout_WritesToStdout(t *testing.T) {
	s := NewStdout()
	// Should not panic when writing
	err := s.Write(log.Entry{
		Timestamp: time.Now(),
		Level:     log.Debug,
		Message:   "debug test",
	})
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
}
