package scraper

import (
	"context"
	"log"
	"sync"

	"github.com/chromedp/chromedp"
)

// BrowserPool manages a single persistent browser with multiple tabs.
// Includes basic restart capability if browser crashes.
type BrowserPool struct {
	allocCtx context.Context
	ctx      context.Context
	cancel   context.CancelFunc
	opts     []chromedp.ExecAllocatorOption
	mu       sync.Mutex
}

// NewBrowserPool creates a new browser pool with a headless Chrome instance.
func NewBrowserPool() (*BrowserPool, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	bp := &BrowserPool{opts: opts}
	if err := bp.start(); err != nil {
		return nil, err
	}

	return bp, nil
}

// start initializes or restarts the browser.
func (bp *BrowserPool) start() error {
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), bp.opts...)
	ctx, _ := chromedp.NewContext(allocCtx)

	// Start browser
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		return err
	}

	bp.allocCtx = allocCtx
	bp.ctx = ctx
	bp.cancel = cancel
	return nil
}

// NewTab creates a new browser tab and returns its context.
// If the browser has crashed, it will attempt to restart.
func (bp *BrowserPool) NewTab() (context.Context, context.CancelFunc) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	tabCtx, tabCancel := chromedp.NewContext(bp.ctx)

	// Basic health check - if browser is dead, restart it
	if err := chromedp.Run(tabCtx); err != nil {
		log.Printf("Browser pool: tab creation failed, attempting restart: %v", err)
		tabCancel()

		if restartErr := bp.start(); restartErr != nil {
			log.Printf("Browser pool: restart failed: %v", restartErr)
			return tabCtx, tabCancel // Return failed context
		}

		// Try again after restart
		return chromedp.NewContext(bp.ctx)
	}

	return tabCtx, tabCancel
}

// Close shuts down the browser pool.
func (bp *BrowserPool) Close() {
	if bp.cancel != nil {
		bp.cancel()
	}
}
