package log

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// captureTransporter captures entries for testing
type captureTransporter struct {
	mu      sync.Mutex
	entries []Entry
}

func (c *captureTransporter) Name() string { return "capture" }

func (c *captureTransporter) Write(entry Entry) error {
	c.mu.Lock()
	c.entries = append(c.entries, entry)
	c.mu.Unlock()
	return nil
}

func (c *captureTransporter) Close() error { return nil }

func (c *captureTransporter) Entries() []Entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Entry{}, c.entries...)
}

func (c *captureTransporter) Last() *Entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) == 0 {
		return nil
	}
	e := c.entries[len(c.entries)-1]
	return &e
}

func (c *captureTransporter) Clear() {
	c.mu.Lock()
	c.entries = nil
	c.mu.Unlock()
}

func setupTestLogger() (*Logger, *captureTransporter) {
	capture := &captureTransporter{}
	logger := New(Info, capture)
	return logger, capture
}

func TestLogger_Info_CreatesInfoEntry(t *testing.T) {
	logger, capture := setupTestLogger()
	defer logger.Close()

	logger.Info("test message")
	time.Sleep(50 * time.Millisecond)

	entry := capture.Last()
	if entry == nil {
		t.Fatal("no entry captured")
	}
	if entry.Level != Info {
		t.Errorf("Level = %v, want %v", entry.Level, Info)
	}
	if entry.Message != "test message" {
		t.Errorf("Message = %q, want %q", entry.Message, "test message")
	}
}

func TestLogger_Error_CreatesErrorEntry(t *testing.T) {
	logger, capture := setupTestLogger()
	defer logger.Close()

	logger.Error("error occurred")
	time.Sleep(50 * time.Millisecond)

	entry := capture.Last()
	if entry == nil {
		t.Fatal("no entry captured")
	}
	if entry.Level != Error {
		t.Errorf("Level = %v, want %v", entry.Level, Error)
	}
}

func TestLogger_Debug_BelowMinLevel_NotLogged(t *testing.T) {
	logger, capture := setupTestLogger() // Min level is Info
	defer logger.Close()

	logger.Debug("debug message")
	time.Sleep(50 * time.Millisecond)

	if len(capture.Entries()) != 0 {
		t.Error("debug should not be logged when min level is Info")
	}
}

func TestLogger_SetLevel_ChangesMinLevel(t *testing.T) {
	logger, capture := setupTestLogger()
	defer logger.Close()

	logger.SetLevel(Debug)
	logger.Debug("now visible")
	time.Sleep(50 * time.Millisecond)

	if len(capture.Entries()) != 1 {
		t.Error("debug should be logged after SetLevel(Debug)")
	}
}

func TestLogger_Info_WithFields_AddsFields(t *testing.T) {
	logger, capture := setupTestLogger()
	defer logger.Close()

	logger.Info("test", "key1", "value1", "key2", 42)
	time.Sleep(50 * time.Millisecond)

	entry := capture.Last()
	if entry == nil {
		t.Fatal("no entry captured")
	}
	if entry.Fields["key1"] != "value1" {
		t.Errorf("Fields[key1] = %v, want %q", entry.Fields["key1"], "value1")
	}
	if entry.Fields["key2"] != 42 {
		t.Errorf("Fields[key2] = %v, want %d", entry.Fields["key2"], 42)
	}
}

func TestLogger_InfoCtx_ExtractsRequestID(t *testing.T) {
	logger, capture := setupTestLogger()
	defer logger.Close()

	ctx := WithRequestID(context.Background(), "req-abc")
	logger.InfoCtx(ctx, "with request id")
	time.Sleep(50 * time.Millisecond)

	entry := capture.Last()
	if entry == nil {
		t.Fatal("no entry captured")
	}
	if entry.RequestID != "req-abc" {
		t.Errorf("RequestID = %q, want %q", entry.RequestID, "req-abc")
	}
}

func TestLogger_InfoCtx_ExtractsContextFields(t *testing.T) {
	logger, capture := setupTestLogger()
	defer logger.Close()

	ctx := WithFields(context.Background(), "ctx_field", "ctx_value")
	logger.InfoCtx(ctx, "with context fields")
	time.Sleep(50 * time.Millisecond)

	entry := capture.Last()
	if entry == nil {
		t.Fatal("no entry captured")
	}
	if entry.Fields["ctx_field"] != "ctx_value" {
		t.Errorf("Fields[ctx_field] = %v, want %q", entry.Fields["ctx_field"], "ctx_value")
	}
}

func TestLogger_Info_IncludesCaller(t *testing.T) {
	logger, capture := setupTestLogger()
	defer logger.Close()

	logger.Info("test")
	time.Sleep(50 * time.Millisecond)

	entry := capture.Last()
	if entry == nil {
		t.Fatal("no entry captured")
	}
	if entry.Caller == "" {
		t.Error("Caller should not be empty")
	}
	if !strings.Contains(entry.Caller, "logger_test.go") {
		t.Errorf("Caller = %q, should contain 'logger_test.go'", entry.Caller)
	}
}

func TestLogger_With_CreatesChildLogger(t *testing.T) {
	logger, capture := setupTestLogger()
	defer logger.Close()

	child := logger.With("component", "scraper")
	child.Info("from child")
	time.Sleep(50 * time.Millisecond)

	entry := capture.Last()
	if entry == nil {
		t.Fatal("no entry captured")
	}
	if entry.Fields["component"] != "scraper" {
		t.Errorf("Fields[component] = %v, want %q", entry.Fields["component"], "scraper")
	}
}

func TestLogger_AllLevels_Work(t *testing.T) {
	capture := &captureTransporter{}
	logger := New(Trace, capture) // Allow all levels
	defer logger.Close()

	logger.Trace("trace")
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	time.Sleep(100 * time.Millisecond)

	entries := capture.Entries()
	if len(entries) != 5 {
		t.Errorf("entries count = %d, want 5", len(entries))
	}
}

func TestLogger_Fatal_LogsAtFatalLevel(t *testing.T) {
	logger, capture := setupTestLogger()
	defer logger.Close()

	// Note: We don't actually exit in tests
	logger.Fatal("fatal error")
	time.Sleep(50 * time.Millisecond)

	entry := capture.Last()
	if entry == nil {
		t.Fatal("no entry captured")
	}
	if entry.Level != Fatal {
		t.Errorf("Level = %v, want %v", entry.Level, Fatal)
	}
}

// Test global logger functions
func TestGlobal_SetDefault_ConfiguresGlobal(t *testing.T) {
	capture := &captureTransporter{}
	logger := New(Info, capture)

	SetDefault(logger)

	// Use global functions
	GlobalInfo("global test")
	time.Sleep(50 * time.Millisecond)

	entry := capture.Last()
	if entry == nil {
		t.Fatal("no entry captured via global")
	}
	if entry.Message != "global test" {
		t.Errorf("Message = %q, want %q", entry.Message, "global test")
	}

	logger.Close()
}
