package web

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
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

