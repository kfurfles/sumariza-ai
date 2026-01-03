//go:build integration

package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ChromeContainer wraps a testcontainers Chrome instance
type ChromeContainer struct {
	testcontainers.Container
	wsURL string
}

// setupChromeContainer starts a Chrome container with CDP exposed
func setupChromeContainer(ctx context.Context) (*ChromeContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "chromedp/headless-shell:latest",
		ExposedPorts: []string{"9222/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("DevTools listening").WithStartupTimeout(60*time.Second),
			wait.ForHTTP("/json/version").WithPort("9222/tcp").WithStartupTimeout(60*time.Second),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	port, err := container.MappedPort(ctx, "9222")
	if err != nil {
		return nil, fmt.Errorf("failed to get port: %w", err)
	}

	// Get the actual WebSocket URL from Chrome's /json/version endpoint
	versionURL := fmt.Sprintf("http://%s:%s/json/version", host, port.Port())
	wsURL, err := getWebSocketURL(versionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get WebSocket URL: %w", err)
	}

	// Replace internal container hostname with actual host
	wsURL = replaceHost(wsURL, host, port.Port())

	return &ChromeContainer{
		Container: container,
		wsURL:     wsURL,
	}, nil
}

// getWebSocketURL fetches the DevTools WebSocket URL from Chrome
func getWebSocketURL(versionURL string) (string, error) {
	resp, err := http.Get(versionURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.WebSocketDebuggerURL, nil
}

// replaceHost replaces the container internal host with the external mapped host
func replaceHost(wsURL, host, port string) string {
	// The Chrome returns ws://127.0.0.1:9222/devtools/browser/<uuid>
	// We need to replace the host:port with the mapped external ones
	// Find the path part starting from /devtools
	idx := 0
	for i := len("ws://"); i < len(wsURL); i++ {
		if wsURL[i] == '/' {
			idx = i
			break
		}
	}
	if idx > 0 {
		return fmt.Sprintf("ws://%s:%s%s", host, port, wsURL[idx:])
	}
	return wsURL
}

// TestBrowserPool is a testable version that connects to a remote Chrome
type TestBrowserPoolIntegration struct {
	allocCtx context.Context
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
	tabSem   chan struct{}
}

// NewTestBrowserPoolIntegration creates a browser pool connected to remote Chrome
func NewTestBrowserPoolIntegration(wsURL string) (*TestBrowserPoolIntegration, error) {
	allocCtx, cancel := chromedp.NewRemoteAllocator(context.Background(), wsURL)
	ctx, _ := chromedp.NewContext(allocCtx)

	// Verify connection
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to chrome: %w", err)
	}

	return &TestBrowserPoolIntegration{
		allocCtx: allocCtx,
		ctx:      ctx,
		cancel:   cancel,
		tabSem:   make(chan struct{}, 1), // HARD LIMIT: 1 tab
	}, nil
}

// WithTab executes with exclusive tab access (same logic as production BrowserPool)
func (bp *TestBrowserPoolIntegration) WithTab(fn func(ctx context.Context) error) error {
	bp.tabSem <- struct{}{}
	defer func() { <-bp.tabSem }()

	bp.mu.Lock()
	tabCtx, tabCancel := chromedp.NewContext(bp.ctx)
	bp.mu.Unlock()
	defer tabCancel()

	return fn(tabCtx)
}

// Close shuts down the browser pool
func (bp *TestBrowserPoolIntegration) Close() {
	if bp.cancel != nil {
		bp.cancel()
	}
}

// --- Integration Tests ---

func TestIntegration_BrowserPool_WithTab_NavigatesSuccessfully(t *testing.T) {
	ctx := context.Background()

	// Start Chrome container
	chrome, err := setupChromeContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to setup Chrome container: %v", err)
	}
	defer chrome.Terminate(ctx)

	// Create browser pool connected to container
	pool, err := NewTestBrowserPoolIntegration(chrome.wsURL)
	if err != nil {
		t.Fatalf("Failed to create browser pool: %v", err)
	}
	defer pool.Close()

	// Test navigation
	var title string
	err = pool.WithTab(func(tabCtx context.Context) error {
		return chromedp.Run(tabCtx,
			chromedp.Navigate("https://example.com"),
			chromedp.Title(&title),
		)
	})

	if err != nil {
		t.Errorf("Navigation failed: %v", err)
	}
	if title == "" {
		t.Error("Expected page title, got empty string")
	}
	t.Logf("Successfully navigated, page title: %s", title)
}

func TestIntegration_BrowserPool_Backpressure_OnlyOneTabAtATime(t *testing.T) {
	ctx := context.Background()

	// Start Chrome container
	chrome, err := setupChromeContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to setup Chrome container: %v", err)
	}
	defer chrome.Terminate(ctx)

	// Create browser pool
	pool, err := NewTestBrowserPoolIntegration(chrome.wsURL)
	if err != nil {
		t.Fatalf("Failed to create browser pool: %v", err)
	}
	defer pool.Close()

	var concurrentCount int32
	var maxConcurrent int32
	var wg sync.WaitGroup

	// Launch 3 concurrent requests (real browser operations)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = pool.WithTab(func(tabCtx context.Context) error {
				// Increment concurrent counter
				current := atomic.AddInt32(&concurrentCount, 1)

				// Track max concurrent
				for {
					max := atomic.LoadInt32(&maxConcurrent)
					if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
						break
					}
				}

				// Real browser work - navigate to a page
				var title string
				err := chromedp.Run(tabCtx,
					chromedp.Navigate("https://example.com"),
					chromedp.Title(&title),
				)

				t.Logf("Tab %d: title=%s, concurrent=%d", idx, title, current)

				// Decrement concurrent counter
				atomic.AddInt32(&concurrentCount, -1)
				return err
			})
		}(i)
	}

	wg.Wait()

	// Assert - max concurrent should never exceed 1
	if maxConcurrent != 1 {
		t.Errorf("maxConcurrent: got %d, want 1 (backpressure violated!)", maxConcurrent)
	} else {
		t.Logf("Backpressure working correctly: maxConcurrent=%d", maxConcurrent)
	}
}

func TestIntegration_BrowserPool_MultipleTabs_AllComplete(t *testing.T) {
	ctx := context.Background()

	// Start Chrome container
	chrome, err := setupChromeContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to setup Chrome container: %v", err)
	}
	defer chrome.Terminate(ctx)

	// Create browser pool
	pool, err := NewTestBrowserPoolIntegration(chrome.wsURL)
	if err != nil {
		t.Fatalf("Failed to create browser pool: %v", err)
	}
	defer pool.Close()

	var completed int32
	var wg sync.WaitGroup
	numRequests := 5

	// Launch multiple concurrent requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := pool.WithTab(func(tabCtx context.Context) error {
				var title string
				return chromedp.Run(tabCtx,
					chromedp.Navigate("https://example.com"),
					chromedp.Title(&title),
				)
			})
			if err == nil {
				atomic.AddInt32(&completed, 1)
			} else {
				t.Logf("Request %d failed: %v", idx, err)
			}
		}(i)
	}

	wg.Wait()

	// All requests should complete
	if completed != int32(numRequests) {
		t.Errorf("completed: got %d, want %d", completed, numRequests)
	} else {
		t.Logf("All %d requests completed successfully", numRequests)
	}
}

func TestIntegration_BrowserPool_SemaphoreReleased_OnError(t *testing.T) {
	ctx := context.Background()

	// Start Chrome container
	chrome, err := setupChromeContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to setup Chrome container: %v", err)
	}
	defer chrome.Terminate(ctx)

	// Create browser pool
	pool, err := NewTestBrowserPoolIntegration(chrome.wsURL)
	if err != nil {
		t.Fatalf("Failed to create browser pool: %v", err)
	}
	defer pool.Close()

	// First request - intentionally fails (invalid URL)
	err = pool.WithTab(func(tabCtx context.Context) error {
		return chromedp.Run(tabCtx,
			chromedp.Navigate("http://invalid.url.that.does.not.exist.local"),
			chromedp.WaitVisible("body", chromedp.ByQuery),
		)
	})
	// Error is expected
	t.Logf("First request error (expected): %v", err)

	// Second request should NOT block (semaphore must be released)
	done := make(chan bool, 1)
	go func() {
		_ = pool.WithTab(func(tabCtx context.Context) error {
			var title string
			return chromedp.Run(tabCtx,
				chromedp.Navigate("https://example.com"),
				chromedp.Title(&title),
			)
		})
		done <- true
	}()

	// Should complete within timeout
	select {
	case <-done:
		t.Log("Semaphore released correctly after error")
	case <-time.After(30 * time.Second):
		t.Error("Second request blocked - semaphore was NOT released after error")
	}
}

func TestIntegration_BrowserPool_ExtractHTML(t *testing.T) {
	ctx := context.Background()

	// Start Chrome container
	chrome, err := setupChromeContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to setup Chrome container: %v", err)
	}
	defer chrome.Terminate(ctx)

	// Create browser pool
	pool, err := NewTestBrowserPoolIntegration(chrome.wsURL)
	if err != nil {
		t.Fatalf("Failed to create browser pool: %v", err)
	}
	defer pool.Close()

	var html string
	err = pool.WithTab(func(tabCtx context.Context) error {
		return chromedp.Run(tabCtx,
			chromedp.Navigate("https://example.com"),
			chromedp.WaitVisible("body", chromedp.ByQuery),
			chromedp.OuterHTML("html", &html),
		)
	})

	if err != nil {
		t.Errorf("Failed to extract HTML: %v", err)
	}

	if html == "" {
		t.Error("Expected HTML content, got empty string")
	}

	if len(html) < 100 {
		t.Errorf("HTML content too short: %d bytes", len(html))
	}

	t.Logf("Successfully extracted %d bytes of HTML", len(html))
}
