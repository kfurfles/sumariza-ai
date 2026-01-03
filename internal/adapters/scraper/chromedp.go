package scraper

import (
	"context"
	"os"
	"sync"

	"sumariza-ai/pkg/log"

	"github.com/chromedp/chromedp"
)

// BrowserPool manages a single Chrome process and enforces
// serialized tab usage (1 tab at a time).
// Safe for 4GB VPS environments.
type BrowserPool struct {
	allocCtx context.Context
	ctx      context.Context
	cancel   context.CancelFunc
	opts     []chromedp.ExecAllocatorOption

	mu     sync.Mutex
	tabSem chan struct{}
}

// NewBrowserPool creates a browser pool with exactly one Chrome instance
// and one tab allowed at a time.
func NewBrowserPool(options []chromedp.ExecAllocatorOption) (*BrowserPool, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// Core
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),

		// Memory / CPU reduction
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
	)

	opts = append(opts, options...)

	// Explicit Chrome/Chromium path (systemd-safe)
	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		log.GlobalInfo("browser pool using custom chrome path", "path", chromePath)
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	bp := &BrowserPool{
		opts:   opts,
		tabSem: make(chan struct{}, 1), // HARD LIMIT: 1 tab
	}

	if err := bp.start(); err != nil {
		return nil, err
	}

	return bp, nil
}

// start initializes or restarts the Chrome process.
func (bp *BrowserPool) start() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.cancel != nil {
		bp.cancel()
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), bp.opts...)
	ctx, _ := chromedp.NewContext(allocCtx)

	// Force Chrome startup
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		return err
	}

	bp.allocCtx = allocCtx
	bp.ctx = ctx
	bp.cancel = cancel

	log.GlobalInfo("browser pool chrome started")
	return nil
}

// WithTab executes a function with exclusive access to a browser tab.
// This guarantees:
//   - one Chrome process
//   - one tab at a time
func (bp *BrowserPool) WithTab(fn func(ctx context.Context) error) error {
	// Acquire tab slot (blocks until available)
	bp.tabSem <- struct{}{}
	defer func() { <-bp.tabSem }()

	// Acquire a healthy tab (handles restart if needed)
	tabCtx, tabCancel, err := bp.acquireTab()
	if err != nil {
		return err
	}
	defer tabCancel()

	return fn(tabCtx)
}

// acquireTab creates a new browser tab and performs a health check.
// If the browser is unhealthy, it restarts Chrome and creates a new tab.
// Returns exactly one valid tab context with its cancel function.
func (bp *BrowserPool) acquireTab() (context.Context, context.CancelFunc, error) {
	bp.mu.Lock()
	tabCtx, tabCancel := chromedp.NewContext(bp.ctx)
	bp.mu.Unlock()

	// Health check - verify the tab is functional
	if err := chromedp.Run(tabCtx); err != nil {
		// Cancel the failed tab before restart
		tabCancel()

		log.GlobalWarn("browser pool tab failed, restarting chrome", "error", err)

		if restartErr := bp.start(); restartErr != nil {
			return nil, nil, restartErr
		}

		// Create a new tab after restart
		bp.mu.Lock()
		tabCtx, tabCancel = chromedp.NewContext(bp.ctx)
		bp.mu.Unlock()
	}

	return tabCtx, tabCancel, nil
}

// Close shuts down the browser completely.
func (bp *BrowserPool) Close() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.cancel != nil {
		bp.cancel()
		log.GlobalInfo("browser pool chrome stopped")
	}
}
