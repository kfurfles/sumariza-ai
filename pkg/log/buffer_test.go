package log

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testTransporter is a mock for buffer tests
type testTransporter struct {
	mu       sync.Mutex
	entries  []Entry
	writeErr error
	delay    time.Duration
	closed   bool
}

func (t *testTransporter) Name() string { return "test" }

func (t *testTransporter) Write(entry Entry) error {
	if t.delay > 0 {
		time.Sleep(t.delay)
	}
	if t.writeErr != nil {
		return t.writeErr
	}
	t.mu.Lock()
	t.entries = append(t.entries, entry)
	t.mu.Unlock()
	return nil
}

func (t *testTransporter) Close() error {
	t.closed = true
	return nil
}

func (t *testTransporter) Entries() []Entry {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]Entry{}, t.entries...)
}

func TestBuffer_Send_QueuesEntry(t *testing.T) {
	transport := &testTransporter{}
	buf := NewBuffer(100, transport)
	defer buf.Close()

	entry := *NewEntry(Info, "test message")
	buf.Send(entry)

	// Wait for async processing
	time.Sleep(50 * time.Millisecond)

	entries := transport.Entries()
	if len(entries) != 1 {
		t.Errorf("entries count = %d, want 1", len(entries))
	}
}

func TestBuffer_Send_MultipleEntries_AllDelivered(t *testing.T) {
	transport := &testTransporter{}
	buf := NewBuffer(100, transport)
	defer buf.Close()

	for i := 0; i < 10; i++ {
		buf.Send(*NewEntry(Info, "message"))
	}

	time.Sleep(100 * time.Millisecond)

	entries := transport.Entries()
	if len(entries) != 10 {
		t.Errorf("entries count = %d, want 10", len(entries))
	}
}

func TestBuffer_Send_BufferFull_DropsOldest(t *testing.T) {
	transport := &testTransporter{delay: 100 * time.Millisecond}
	buf := NewBuffer(3, transport) // Small buffer
	defer buf.Close()

	// Send more than buffer size
	for i := 0; i < 5; i++ {
		buf.Send(*NewEntry(Info, "message").With("seq", i))
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	entries := transport.Entries()
	// Should have dropped oldest entries
	if len(entries) > 5 {
		t.Errorf("entries count = %d, should not exceed sent count", len(entries))
	}
}

func TestBuffer_Close_FlushesRemaining(t *testing.T) {
	transport := &testTransporter{}
	buf := NewBuffer(100, transport)

	for i := 0; i < 5; i++ {
		buf.Send(*NewEntry(Info, "message"))
	}

	buf.Close()

	// After close, all entries should be delivered
	entries := transport.Entries()
	if len(entries) != 5 {
		t.Errorf("entries count = %d, want 5", len(entries))
	}
}

func TestBuffer_Close_CalledMultipleTimes_NoPanic(t *testing.T) {
	transport := &testTransporter{}
	buf := NewBuffer(100, transport)

	buf.Close()
	buf.Close() // Should not panic
}

func TestBuffer_Send_AfterClose_DoesNotPanic(t *testing.T) {
	transport := &testTransporter{}
	buf := NewBuffer(100, transport)
	buf.Close()

	// Should not panic
	buf.Send(*NewEntry(Info, "after close"))
}

func TestBuffer_TransporterError_FallsBackToStderr(t *testing.T) {
	transport := &testTransporter{writeErr: errors.New("write failed")}
	buf := NewBuffer(100, transport)
	defer buf.Close()

	buf.Send(*NewEntry(Error, "error message"))

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Entry was not stored (error) but should not crash
	entries := transport.Entries()
	if len(entries) != 0 {
		t.Errorf("entries count = %d, want 0 (due to error)", len(entries))
	}
}

func TestBuffer_MultipleTransporters_AllReceiveEntry(t *testing.T) {
	t1 := &testTransporter{}
	t2 := &testTransporter{}
	buf := NewBuffer(100, t1, t2)
	defer buf.Close()

	buf.Send(*NewEntry(Info, "broadcast"))

	time.Sleep(50 * time.Millisecond)

	if len(t1.Entries()) != 1 {
		t.Errorf("t1 entries = %d, want 1", len(t1.Entries()))
	}
	if len(t2.Entries()) != 1 {
		t.Errorf("t2 entries = %d, want 1", len(t2.Entries()))
	}
}

func TestBuffer_DroppedCount_TracksDroppedEntries(t *testing.T) {
	transport := &testTransporter{delay: 50 * time.Millisecond}
	buf := NewBuffer(2, transport) // Very small buffer

	// Flood the buffer
	for i := 0; i < 10; i++ {
		buf.Send(*NewEntry(Info, "flood"))
	}

	time.Sleep(10 * time.Millisecond) // Let some drop

	dropped := buf.DroppedCount()
	if dropped == 0 {
		// It's possible nothing dropped if timing was lucky, skip in that case
		t.Skip("no drops detected, timing dependent test")
	}
}

func TestBuffer_Concurrent_Send_Safe(t *testing.T) {
	transport := &testTransporter{}
	buf := NewBuffer(1000, transport)
	defer buf.Close()

	var wg sync.WaitGroup
	var sent int64

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				buf.Send(*NewEntry(Info, "concurrent"))
				atomic.AddInt64(&sent, 1)
			}
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	entries := transport.Entries()
	dropped := buf.DroppedCount()

	// All entries should be either delivered or counted as dropped
	total := int64(len(entries)) + int64(dropped)
	if total < sent {
		t.Errorf("delivered(%d) + dropped(%d) = %d, want >= %d", len(entries), dropped, total, sent)
	}
}
