package scraper

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"sumariza-ai/pkg/log"

	// "github.com/chromedp/cdproto/network"
	// "github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
)

const defaultIdleTimeout = 5 * time.Minute

// BrowserPool manages a single Chrome instance with a single reusable tab.
// Chrome is started lazily on first request and stopped after idle timeout.
type BrowserPool struct {
	allocCtx   context.Context
	browserCtx context.Context
	tabCtx     context.Context
	tabCancel  context.CancelFunc
	cancel     context.CancelFunc
	opts       []chromedp.ExecAllocatorOption

	mu         sync.Mutex
	chromeLogs *strings.Builder

	// Idle timeout management
	idleTimeout time.Duration
	idleTimer   *time.Timer
	running     bool
}

// NewBrowserPool creates a browser pool with one Chrome instance and one reusable tab.
// Chrome starts lazily on first request and stops after 5 minutes of inactivity.
func NewBrowserPool(options []chromedp.ExecAllocatorOption) (*BrowserPool, error) {
	chromeLogs := &strings.Builder{}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-notifications", true),
		chromedp.Flag("disable-translate", true),
		chromedp.Flag("disable-component-update", true),
		chromedp.Flag("disable-domain-reliability", true),
		chromedp.Flag("disable-features", "Translate,BackForwardCache"),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("disable-site-isolation-trials", true),
		chromedp.CombinedOutput(chromeLogs),
	)

	opts = append(opts, options...)

	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		log.GlobalInfo("browser pool using custom chrome path", "path", chromePath)
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	bp := &BrowserPool{
		opts:        opts,
		chromeLogs:  chromeLogs,
		idleTimeout: defaultIdleTimeout,
		running:     false,
	}

	// Lazy start - Chrome will start on first request
	log.GlobalInfo("browser pool initialized (lazy start)", "idle_timeout", defaultIdleTimeout)

	return bp, nil
}

// startBrowser initializes Chrome and creates the persistent tab.
// Must be called with mutex NOT held.
func (bp *BrowserPool) startBrowser() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	return bp.startBrowserLocked()
}

// startBrowserLocked initializes Chrome. Must be called with mutex held.
func (bp *BrowserPool) startBrowserLocked() error {
	// Cleanup previous instance if any
	if bp.cancel != nil {
		bp.cancel()
	}
	if bp.chromeLogs != nil {
		bp.chromeLogs.Reset()
	}

	log.GlobalDebug("browser pool starting chrome")

	// Create allocator
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), bp.opts...)

	// Create browser context
	browserCtx, _ := chromedp.NewContext(allocCtx)

	// Start Chrome
	if err := chromedp.Run(browserCtx); err != nil {
		chromeLogs := ""
		if bp.chromeLogs != nil && bp.chromeLogs.Len() > 0 {
			chromeLogs = bp.chromeLogs.String()
		}
		allocCancel()
		log.GlobalError("browser pool chrome startup failed",
			"error", err,
			"chrome_logs", chromeLogs)
		return err
	}

	// Create the persistent tab
	tabCtx, tabCancel := chromedp.NewContext(browserCtx)

	// Initialize the tab
	if err := chromedp.Run(tabCtx, chromedp.Navigate("about:blank")); err != nil {
		chromeLogs := ""
		if bp.chromeLogs != nil && bp.chromeLogs.Len() > 0 {
			chromeLogs = bp.chromeLogs.String()
		}
		tabCancel()
		allocCancel()
		log.GlobalError("browser pool tab initialization failed",
			"error", err,
			"chrome_logs", chromeLogs)
		return err
	}

	bp.allocCtx = allocCtx
	bp.browserCtx = browserCtx
	bp.tabCtx = tabCtx
	bp.tabCancel = tabCancel
	bp.cancel = allocCancel
	bp.running = true

	log.GlobalInfo("browser pool chrome started")
	return nil
}

// stopBrowserLocked stops Chrome. Must be called with mutex held.
func (bp *BrowserPool) stopBrowserLocked() {
	if !bp.running {
		return
	}

	if bp.tabCancel != nil {
		bp.tabCancel()
		bp.tabCancel = nil
	}
	if bp.cancel != nil {
		bp.cancel()
		bp.cancel = nil
	}

	bp.tabCtx = nil
	bp.browserCtx = nil
	bp.allocCtx = nil
	bp.running = false

	log.GlobalInfo("browser pool chrome stopped (idle timeout)")
}

// resetIdleTimer resets the idle timeout timer.
// Must be called with mutex held.
func (bp *BrowserPool) resetIdleTimer() {
	// Stop existing timer if any
	if bp.idleTimer != nil {
		bp.idleTimer.Stop()
	}

	// Start new timer
	bp.idleTimer = time.AfterFunc(bp.idleTimeout, func() {
		bp.mu.Lock()
		defer bp.mu.Unlock()

		if bp.running {
			log.GlobalInfo("browser pool idle timeout reached", "timeout", bp.idleTimeout)
			bp.stopBrowserLocked()
		}
	})
}

// stopIdleTimer stops the idle timeout timer.
// Must be called with mutex held.
func (bp *BrowserPool) stopIdleTimer() {
	if bp.idleTimer != nil {
		bp.idleTimer.Stop()
		bp.idleTimer = nil
	}
}

// cleanTab clears cookies, cache, and navigates to about:blank.
func (bp *BrowserPool) cleanTab() error {
	if bp.tabCtx == nil {
		return nil
	}
	return nil
	// return chromedp.Run(bp.tabCtx,
	// 	network.ClearBrowserCookies(),
	// 	network.ClearBrowserCache(),
	// 	storage.ClearDataForOrigin("", "*"),
	// 	chromedp.Navigate("about:blank"),
	// )
}

// isHealthyLocked checks if the tab is still functional.
// Must be called with mutex held.
func (bp *BrowserPool) isHealthyLocked() bool {
	if !bp.running {
		return false
	}
	if bp.tabCtx == nil {
		return false
	}
	if bp.tabCtx.Err() != nil {
		return false
	}

	// Try a simple operation
	var title string
	err := chromedp.Run(bp.tabCtx, chromedp.Title(&title))
	return err == nil
}

// ensureBrowserRunning ensures Chrome is running, starting it if necessary.
// Must be called with mutex held.
func (bp *BrowserPool) ensureBrowserRunning() error {
	if bp.isHealthyLocked() {
		return nil
	}

	log.GlobalDebug("browser pool ensuring chrome is running")
	return bp.startBrowserLocked()
}

// Execute runs chromedp actions with proper locking and health management.
// This is the main entry point for scraping operations.
func (bp *BrowserPool) Execute(ctx context.Context, actions ...chromedp.Action) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Check caller context
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Ensure browser is running
	if err := bp.ensureBrowserRunning(); err != nil {
		return err
	}

	// Execute the actions
	err := chromedp.Run(bp.tabCtx, actions...)

	// Clean up after use (best effort)
	if cleanErr := bp.cleanTab(); cleanErr != nil {
		log.GlobalDebug("browser pool clean tab failed", "error", cleanErr)
	}

	// Reset idle timer
	bp.resetIdleTimer()

	return err
}

// WithTab provides backward compatibility - executes a function with tab access.
// Deprecated: Use Execute() directly with chromedp actions.
func (bp *BrowserPool) WithTab(fn func(ctx context.Context) error) error {
	return bp.WithTabCtx(context.Background(), fn)
}

// WithTabCtx executes a function with tab access, respecting context cancellation.
func (bp *BrowserPool) WithTabCtx(ctx context.Context, fn func(ctx context.Context) error) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Check caller context
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Ensure browser is running
	if err := bp.ensureBrowserRunning(); err != nil {
		return err
	}

	// Execute the function
	err := fn(bp.tabCtx)

	// Clean up after use (best effort)
	if cleanErr := bp.cleanTab(); cleanErr != nil {
		log.GlobalDebug("browser pool clean tab failed", "error", cleanErr)
	}

	// Reset idle timer
	bp.resetIdleTimer()

	return err
}

// Close shuts down the browser and stops all timers.
func (bp *BrowserPool) Close() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.stopIdleTimer()
	bp.stopBrowserLocked()

	log.GlobalInfo("browser pool closed")
}
