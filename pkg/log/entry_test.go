package log

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEntry_MarshalJSON_ContainsAllFields(t *testing.T) {
	ts := time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC)
	entry := Entry{
		Timestamp: ts,
		Level:     Info,
		Caller:    "main.go:42",
		RequestID: "req-123",
		Message:   "test message",
		Fields:    map[string]any{"user": "john", "count": 5},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Check timestamp is ISO8601
	if result["timestamp"] != "2026-01-03T12:00:00Z" {
		t.Errorf("timestamp = %v, want %v", result["timestamp"], "2026-01-03T12:00:00Z")
	}

	// Check level is string
	if result["level"] != "INFO" {
		t.Errorf("level = %v, want %v", result["level"], "INFO")
	}

	// Check caller
	if result["caller"] != "main.go:42" {
		t.Errorf("caller = %v, want %v", result["caller"], "main.go:42")
	}

	// Check request_id
	if result["request_id"] != "req-123" {
		t.Errorf("request_id = %v, want %v", result["request_id"], "req-123")
	}

	// Check message
	if result["msg"] != "test message" {
		t.Errorf("msg = %v, want %v", result["msg"], "test message")
	}

	// Check fields are flattened
	if result["user"] != "john" {
		t.Errorf("user = %v, want %v", result["user"], "john")
	}
	if result["count"] != float64(5) { // JSON numbers are float64
		t.Errorf("count = %v, want %v", result["count"], 5)
	}
}

func TestEntry_MarshalJSON_OmitsEmptyRequestID(t *testing.T) {
	entry := Entry{
		Timestamp: time.Now(),
		Level:     Error,
		Message:   "error occurred",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, exists := result["request_id"]; exists {
		t.Error("request_id should be omitted when empty")
	}
}

func TestEntry_MarshalJSON_OmitsEmptyCaller(t *testing.T) {
	entry := Entry{
		Timestamp: time.Now(),
		Level:     Info,
		Message:   "test",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, exists := result["caller"]; exists {
		t.Error("caller should be omitted when empty")
	}
}

func TestEntry_MarshalJSON_NilFieldsHandled(t *testing.T) {
	entry := Entry{
		Timestamp: time.Now(),
		Level:     Info,
		Message:   "test",
		Fields:    nil,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Should not panic and produce valid JSON
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
}

func TestNewEntry_SetsTimestampAutomatically(t *testing.T) {
	before := time.Now()
	entry := NewEntry(Info, "test message")
	after := time.Now()

	if entry.Timestamp.Before(before) || entry.Timestamp.After(after) {
		t.Errorf("Timestamp should be between %v and %v, got %v", before, after, entry.Timestamp)
	}
}

func TestNewEntry_SetsLevelAndMessage(t *testing.T) {
	entry := NewEntry(Warn, "warning message")

	if entry.Level != Warn {
		t.Errorf("Level = %v, want %v", entry.Level, Warn)
	}
	if entry.Message != "warning message" {
		t.Errorf("Message = %q, want %q", entry.Message, "warning message")
	}
}

func TestEntry_With_AddsFields(t *testing.T) {
	entry := NewEntry(Info, "test").With("key1", "value1", "key2", 42)

	if entry.Fields["key1"] != "value1" {
		t.Errorf("Fields[key1] = %v, want %v", entry.Fields["key1"], "value1")
	}
	if entry.Fields["key2"] != 42 {
		t.Errorf("Fields[key2] = %v, want %v", entry.Fields["key2"], 42)
	}
}

func TestEntry_With_OddNumberOfArgs_IgnoresLastKey(t *testing.T) {
	entry := NewEntry(Info, "test").With("key1", "value1", "orphan")

	if entry.Fields["key1"] != "value1" {
		t.Errorf("Fields[key1] = %v, want %v", entry.Fields["key1"], "value1")
	}
	if _, exists := entry.Fields["orphan"]; exists {
		t.Error("orphan key should not exist")
	}
}

func TestEntry_MarshalJSON_ErrorFieldConvertsToString(t *testing.T) {
	entry := Entry{
		Timestamp: time.Now(),
		Level:     Error,
		Message:   "operation failed",
		Fields:    map[string]any{"error": testError{"something went wrong"}},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Error should be serialized as string, not empty object
	errorVal, ok := result["error"].(string)
	if !ok {
		t.Errorf("error field should be string, got %T: %v", result["error"], result["error"])
	}
	if errorVal != "something went wrong" {
		t.Errorf("error = %q, want %q", errorVal, "something went wrong")
	}
}

// testError implements error interface for testing
type testError struct {
	msg string
}

func (e testError) Error() string {
	return e.msg
}
