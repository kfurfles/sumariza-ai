package web

import (
	"sync"
	"time"

	"sumariza-ai/pkg/log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// RateLimiter tracks scrape requests per IP.
type RateLimiter struct {
	scrapes map[string][]time.Time
	mu      sync.RWMutex
	limit   int
	window  time.Duration
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		scrapes: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}
	go rl.cleanup()
	return rl
}

// RecordScrape records a scrape request for the given IP.
func (rl *RateLimiter) RecordScrape(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.scrapes[ip] = append(rl.scrapes[ip], now)
}

// CanScrape checks if the IP is allowed to make another scrape.
func (rl *RateLimiter) CanScrape(ip string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	timestamps := rl.scrapes[ip]

	// Count recent scrapes
	var recent int
	for _, t := range timestamps {
		if t.After(cutoff) {
			recent++
		}
	}

	return recent < rl.limit
}

// Middleware returns a Fiber middleware for rate limiting.
func (rl *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Rate limit is checked at scrape time, not here
		// This middleware can be used for other purposes
		return c.Next()
	}
}

// cleanup periodically removes old entries from the rate limiter.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.window)
		for ip, timestamps := range rl.scrapes {
			var recent []time.Time
			for _, t := range timestamps {
				if t.After(cutoff) {
					recent = append(recent, t)
				}
			}
			if len(recent) == 0 {
				delete(rl.scrapes, ip)
			} else {
				rl.scrapes[ip] = recent
			}
		}
		rl.mu.Unlock()
	}
}

// RequestIDConfig returns the configuration for Fiber's requestid middleware.
// Uses X-Request-ID header, generates UUID if not present.
func RequestIDConfig() requestid.Config {
	return requestid.Config{
		Header:     "X-Request-ID",
		ContextKey: "requestid",
	}
}

// RequestIDToContextMiddleware bridges Fiber's requestid to pkg/log context.
// Must be used AFTER requestid.New() middleware.
func RequestIDToContextMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get request ID from Fiber's requestid middleware
		reqID := c.Locals("requestid")
		if reqID != nil {
			if id, ok := reqID.(string); ok {
				ctx := log.WithRequestID(c.UserContext(), id)
				c.SetUserContext(ctx)
			}
		}
		return c.Next()
	}
}

// RequestLoggerMiddleware logs HTTP requests in structured JSON format.
// Replaces Fiber's default logger middleware.
// Must be used AFTER RequestIDToContextMiddleware.
func RequestLoggerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		status := c.Response().StatusCode()

		// Determine log level based on status
		ctx := c.UserContext()
		fields := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"ip", c.IP(),
			"user_agent", c.Get("User-Agent"),
		}

		// Add error if present
		if err != nil {
			fields = append(fields, "error", err.Error())
		}

		// Log based on status code
		switch {
		case status >= 500:
			log.GlobalErrorCtx(ctx, "request completed", fields...)
		case status >= 400:
			log.GlobalWarnCtx(ctx, "request completed", fields...)
		default:
			log.GlobalInfoCtx(ctx, "request completed", fields...)
		}

		return err
	}
}
