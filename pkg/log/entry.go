package log

import (
	"encoding/json"
	"time"
)

// Entry represents a structured log entry.
type Entry struct {
	Timestamp time.Time
	Level     Level
	Caller    string
	RequestID string
	Message   string
	Fields    map[string]any
}

// NewEntry creates a new log entry with the current timestamp.
func NewEntry(level Level, msg string) *Entry {
	return &Entry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
		Fields:    make(map[string]any),
	}
}

// With adds key-value pairs to the entry's fields.
// Keys and values are provided as alternating arguments.
// If an odd number of arguments is provided, the last key is ignored.
func (e *Entry) With(keysAndValues ...any) *Entry {
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		e.Fields[key] = keysAndValues[i+1]
	}
	return e
}

// MarshalJSON implements json.Marshaler for structured JSON output.
// Fields are flattened into the root object.
// Empty optional fields (caller, request_id) are omitted.
func (e Entry) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)

	m["timestamp"] = e.Timestamp.UTC().Format(time.RFC3339)
	m["level"] = e.Level.String()
	m["msg"] = e.Message

	if e.Caller != "" {
		m["caller"] = e.Caller
	}

	if e.RequestID != "" {
		m["request_id"] = e.RequestID
	}

	// Flatten fields into root
	for k, v := range e.Fields {
		m[k] = v
	}

	return json.Marshal(m)
}
