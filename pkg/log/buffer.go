package log

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
)

// Buffer provides asynchronous log delivery with a ring buffer.
// When the buffer is full, oldest entries are dropped.
type Buffer struct {
	entries      chan Entry
	transporters []Transporter
	dropped      int64
	closed       int32
	done         chan struct{}
	wg           sync.WaitGroup
}

// NewBuffer creates a new async buffer with the given capacity.
// Logs are sent to all provided transporters.
func NewBuffer(capacity int, transporters ...Transporter) *Buffer {
	b := &Buffer{
		entries:      make(chan Entry, capacity),
		transporters: transporters,
		done:         make(chan struct{}),
	}

	b.wg.Add(1)
	go b.worker()

	return b
}

// Send queues an entry for async delivery.
// If the buffer is full, the oldest entry is dropped.
// Safe to call from multiple goroutines.
func (b *Buffer) Send(entry Entry) {
	if atomic.LoadInt32(&b.closed) == 1 {
		return
	}

	select {
	case b.entries <- entry:
		// Successfully queued
	default:
		// Buffer full, drop oldest by receiving and discarding
		select {
		case <-b.entries:
			atomic.AddInt64(&b.dropped, 1)
		default:
			// Someone else already drained it
		}
		// Try again
		select {
		case b.entries <- entry:
		default:
			atomic.AddInt64(&b.dropped, 1)
		}
	}
}

// DroppedCount returns the number of entries dropped due to buffer overflow.
func (b *Buffer) DroppedCount() int64 {
	return atomic.LoadInt64(&b.dropped)
}

// Close stops the worker and flushes remaining entries.
// Safe to call multiple times.
func (b *Buffer) Close() {
	if !atomic.CompareAndSwapInt32(&b.closed, 0, 1) {
		return
	}

	close(b.done)
	b.wg.Wait()

	// Flush remaining entries
	for {
		select {
		case entry := <-b.entries:
			b.deliver(entry)
		default:
			return
		}
	}
}

// worker processes entries from the buffer.
func (b *Buffer) worker() {
	defer b.wg.Done()

	for {
		select {
		case entry := <-b.entries:
			b.deliver(entry)
		case <-b.done:
			return
		}
	}
}

// deliver sends an entry to all transporters.
// On error, falls back to stderr.
func (b *Buffer) deliver(entry Entry) {
	for _, t := range b.transporters {
		if err := t.Write(entry); err != nil {
			// Fallback to stderr
			fmt.Fprintf(os.Stderr, "log transporter %q failed: %v\n", t.Name(), err)
		}
	}
}
