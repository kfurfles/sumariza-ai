package log

import (
	"errors"
	"testing"
)

// mockTransporter is a test double for Transporter interface.
type mockTransporter struct {
	name     string
	entries  []Entry
	closed   bool
	writeErr error
	closeErr error
}

func (m *mockTransporter) Name() string {
	return m.name
}

func (m *mockTransporter) Write(entry Entry) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockTransporter) Close() error {
	m.closed = true
	return m.closeErr
}

func TestTransporter_MockImplementsInterface(t *testing.T) {
	var _ Transporter = &mockTransporter{}
}

func TestTransporter_Write_StoresEntry(t *testing.T) {
	mock := &mockTransporter{name: "test"}
	entry := *NewEntry(Info, "test message")

	err := mock.Write(entry)

	if err != nil {
		t.Errorf("Write() error = %v, want nil", err)
	}
	if len(mock.entries) != 1 {
		t.Errorf("entries count = %d, want 1", len(mock.entries))
	}
	if mock.entries[0].Message != "test message" {
		t.Errorf("entry.Message = %q, want %q", mock.entries[0].Message, "test message")
	}
}

func TestTransporter_Write_ReturnsError(t *testing.T) {
	expectedErr := errors.New("write failed")
	mock := &mockTransporter{name: "test", writeErr: expectedErr}

	err := mock.Write(*NewEntry(Info, "test"))

	if err != expectedErr {
		t.Errorf("Write() error = %v, want %v", err, expectedErr)
	}
}

func TestTransporter_Close_SetsClosed(t *testing.T) {
	mock := &mockTransporter{name: "test"}

	err := mock.Close()

	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
	if !mock.closed {
		t.Error("closed = false, want true")
	}
}

func TestTransporter_Name_ReturnsName(t *testing.T) {
	mock := &mockTransporter{name: "my-transporter"}

	if mock.Name() != "my-transporter" {
		t.Errorf("Name() = %q, want %q", mock.Name(), "my-transporter")
	}
}
