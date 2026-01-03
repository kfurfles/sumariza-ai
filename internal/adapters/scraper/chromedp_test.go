package scraper

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestBrowserPool is a testable version of BrowserPool that doesn't require Chrome.
// It simulates the same semaphore behavior for testing backpressure.
type TestBrowserPool struct {
	tabSem chan struct{}
}

// NewTestBrowserPool creates a test browser pool with configurable concurrency.
func NewTestBrowserPool(maxTabs int) *TestBrowserPool {
	return &TestBrowserPool{
		tabSem: make(chan struct{}, maxTabs),
	}
}

// WithTab simulates the same semaphore logic as BrowserPool.WithTab
func (bp *TestBrowserPool) WithTab(fn func(ctx context.Context) error) error {
	bp.tabSem <- struct{}{}
	defer func() { <-bp.tabSem }()

	return fn(context.Background())
}

// WithTabCtx is like WithTab but respects context cancellation while waiting for semaphore.
// The context is also propagated to the callback function.
func (bp *TestBrowserPool) WithTabCtx(ctx context.Context, fn func(ctx context.Context) error) error {
	// Wait for semaphore while respecting context cancellation
	select {
	case bp.tabSem <- struct{}{}:
		// Acquired semaphore
	case <-ctx.Done():
		// Context canceled while waiting
		return ctx.Err()
	}
	defer func() { <-bp.tabSem }()

	// Propagate the context to the callback
	return fn(ctx)
}

// --- Tests for backpressure behavior ---

func TestWithTab_Backpressure_OnlyOneAtATime(t *testing.T) {
	// Arrange
	pool := NewTestBrowserPool(1) // Only 1 tab allowed

	var concurrentCount int32
	var maxConcurrent int32
	var wg sync.WaitGroup

	// Act - Launch 5 concurrent requests
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = pool.WithTab(func(ctx context.Context) error {
				// Increment concurrent counter
				current := atomic.AddInt32(&concurrentCount, 1)

				// Track max concurrent
				for {
					max := atomic.LoadInt32(&maxConcurrent)
					if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
						break
					}
				}

				// Simulate work
				time.Sleep(10 * time.Millisecond)

				// Decrement concurrent counter
				atomic.AddInt32(&concurrentCount, -1)
				return nil
			})
		}()
	}

	wg.Wait()

	// Assert - max concurrent should never exceed 1
	if maxConcurrent != 1 {
		t.Errorf("maxConcurrent: got %d, want 1 (backpressure violated)", maxConcurrent)
	}
}

func TestWithTab_SemaphoreReleased_OnSuccess(t *testing.T) {
	// Arrange
	pool := NewTestBrowserPool(1)
	executed := false

	// Act - First call
	err := pool.WithTab(func(ctx context.Context) error {
		executed = true
		return nil
	})

	// Assert - First call succeeded
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !executed {
		t.Error("function was not executed")
	}

	// Act - Second call should not block (semaphore was released)
	done := make(chan bool, 1)
	go func() {
		_ = pool.WithTab(func(ctx context.Context) error {
			return nil
		})
		done <- true
	}()

	// Assert - Second call completes within timeout
	select {
	case <-done:
		// Success - semaphore was released
	case <-time.After(100 * time.Millisecond):
		t.Error("second call blocked - semaphore was not released after success")
	}
}

func TestWithTab_SemaphoreReleased_OnError(t *testing.T) {
	// Arrange
	pool := NewTestBrowserPool(1)
	expectedErr := errors.New("intentional error")

	// Act - First call returns error
	err := pool.WithTab(func(ctx context.Context) error {
		return expectedErr
	})

	// Assert - Error was returned
	if err != expectedErr {
		t.Errorf("error: got %v, want %v", err, expectedErr)
	}

	// Act - Second call should not block (semaphore was released despite error)
	done := make(chan bool, 1)
	go func() {
		_ = pool.WithTab(func(ctx context.Context) error {
			return nil
		})
		done <- true
	}()

	// Assert - Second call completes within timeout
	select {
	case <-done:
		// Success - semaphore was released
	case <-time.After(100 * time.Millisecond):
		t.Error("second call blocked - semaphore was not released after error")
	}
}

func TestWithTab_SemaphoreReleased_OnPanic(t *testing.T) {
	// Arrange
	pool := NewTestBrowserPool(1)

	// Act - First call panics (wrapped in recovery)
	func() {
		defer func() {
			recover() // Catch the panic
		}()
		_ = pool.WithTab(func(ctx context.Context) error {
			panic("intentional panic")
		})
	}()

	// Act - Second call should not block (semaphore was released despite panic)
	done := make(chan bool, 1)
	go func() {
		_ = pool.WithTab(func(ctx context.Context) error {
			return nil
		})
		done <- true
	}()

	// Assert - Second call completes within timeout
	select {
	case <-done:
		// Success - semaphore was released
	case <-time.After(100 * time.Millisecond):
		t.Error("second call blocked - semaphore was not released after panic")
	}
}

func TestWithTab_SequentialExecution_MaintainsOrder(t *testing.T) {
	// Arrange
	pool := NewTestBrowserPool(1)
	var order []int
	var mu sync.Mutex

	// Act - Launch requests sequentially
	for i := 0; i < 3; i++ {
		idx := i
		_ = pool.WithTab(func(ctx context.Context) error {
			mu.Lock()
			order = append(order, idx)
			mu.Unlock()
			return nil
		})
	}

	// Assert - Order should be preserved
	if len(order) != 3 {
		t.Errorf("expected 3 executions, got %d", len(order))
	}
	for i := 0; i < 3; i++ {
		if order[i] != i {
			t.Errorf("order[%d]: got %d, want %d", i, order[i], i)
		}
	}
}

func TestWithTab_ConcurrentRequests_AllComplete(t *testing.T) {
	// Arrange
	pool := NewTestBrowserPool(1)
	var completed int32
	var wg sync.WaitGroup
	numRequests := 10

	// Act - Launch many concurrent requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = pool.WithTab(func(ctx context.Context) error {
				atomic.AddInt32(&completed, 1)
				return nil
			})
		}()
	}

	wg.Wait()

	// Assert - All requests should complete
	if completed != int32(numRequests) {
		t.Errorf("completed: got %d, want %d", completed, numRequests)
	}
}

// --- Test that BrowserPool struct has correct semaphore initialization ---

func TestNewBrowserPool_SemaphoreCapacity(t *testing.T) {
	// This test documents the expected behavior:
	// BrowserPool should have a semaphore with capacity 1
	//
	// We can't test the actual BrowserPool without Chrome,
	// but we verify our test pool mimics the same capacity.

	// Arrange
	pool := NewTestBrowserPool(1)

	// Act - Try to acquire twice without releasing
	acquired := 0
	pool.tabSem <- struct{}{} // First acquire
	acquired++

	// Second acquire should block
	select {
	case pool.tabSem <- struct{}{}:
		acquired++
		t.Error("semaphore should block on second acquire (capacity should be 1)")
	case <-time.After(50 * time.Millisecond):
		// Expected - second acquire blocked
	}

	// Assert
	if acquired != 1 {
		t.Errorf("should only acquire once, got %d", acquired)
	}
}

// --- Tests for context cancellation ---

func TestWithTab_ContextCanceled_WhileWaitingForSemaphore(t *testing.T) {
	// Arrange
	pool := NewTestBrowserPool(1)

	// Hold the semaphore to force second call to wait
	pool.tabSem <- struct{}{}

	// Create a context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Act - Try to acquire while semaphore is held
	errChan := make(chan error, 1)
	go func() {
		errChan <- pool.WithTabCtx(ctx, func(tabCtx context.Context) error {
			return nil
		})
	}()

	// Cancel the context after a short delay
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Assert - Should return context.Canceled error
	select {
	case err := <-errChan:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("WithTabCtx should have returned after context cancellation")
	}

	// Cleanup - release semaphore
	<-pool.tabSem
}

func TestWithTab_ContextDeadlineExceeded_WhileWaitingForSemaphore(t *testing.T) {
	// Arrange
	pool := NewTestBrowserPool(1)

	// Hold the semaphore to force second call to wait
	pool.tabSem <- struct{}{}

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	// Act - Try to acquire while semaphore is held
	err := pool.WithTabCtx(ctx, func(tabCtx context.Context) error {
		return nil
	})

	// Assert - Should return context.DeadlineExceeded
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}

	// Cleanup - release semaphore
	<-pool.tabSem
}

func TestWithTab_ContextCanceled_DuringExecution(t *testing.T) {
	// Arrange
	pool := NewTestBrowserPool(1)
	ctx, cancel := context.WithCancel(context.Background())

	// Act - Start execution, cancel during work
	errChan := make(chan error, 1)
	go func() {
		errChan <- pool.WithTabCtx(ctx, func(tabCtx context.Context) error {
			// Simulate long work that checks context
			select {
			case <-tabCtx.Done():
				return tabCtx.Err()
			case <-time.After(1 * time.Second):
				return nil
			}
		})
	}()

	// Cancel after work starts
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Assert - Should return context.Canceled
	select {
	case err := <-errChan:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("WithTabCtx should have returned after context cancellation")
	}
}

func TestWithTab_ContextPropagated_ToFunction(t *testing.T) {
	// Arrange
	pool := NewTestBrowserPool(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var receivedCtx context.Context

	// Act
	err := pool.WithTabCtx(ctx, func(tabCtx context.Context) error {
		receivedCtx = tabCtx
		return nil
	})

	// Assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// The received context should have the same deadline as the parent
	deadline, ok := ctx.Deadline()
	receivedDeadline, receivedOk := receivedCtx.Deadline()

	if ok != receivedOk {
		t.Errorf("deadline presence mismatch: parent=%v, received=%v", ok, receivedOk)
	}
	if ok && !deadline.Equal(receivedDeadline) {
		t.Errorf("deadline mismatch: parent=%v, received=%v", deadline, receivedDeadline)
	}
}
