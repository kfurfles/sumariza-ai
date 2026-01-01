package cache

import (
	"fmt"
	"sync"
	"time"

	"sumariza-ai/internal/domain"
)

// MemoryCache is an in-memory cache with TTL support.
type MemoryCache struct {
	tweets sync.Map
	ttl    time.Duration
}

// cacheEntry holds a cached tweet with expiration metadata.
type cacheEntry struct {
	tweet     *domain.Tweet
	expiresAt time.Time
	scrapedAt time.Time
}

// NewMemoryCache creates a new in-memory cache with the specified TTL.
func NewMemoryCache(ttl time.Duration) *MemoryCache {
	cache := &MemoryCache{ttl: ttl}
	go cache.cleanup()
	return cache
}

// NormalizedKey returns the cache key for a tweet: /{username}/status/{id}
func NormalizedKey(username, tweetID string) string {
	return fmt.Sprintf("/%s/status/%s", username, tweetID)
}

// Get retrieves a tweet from the cache.
// Returns the tweet and true if found and not expired, otherwise nil and false.
func (c *MemoryCache) Get(username, tweetID string) (*domain.Tweet, bool) {
	key := NormalizedKey(username, tweetID)
	value, ok := c.tweets.Load(key)
	if !ok {
		return nil, false
	}

	entry := value.(*cacheEntry)
	if time.Now().After(entry.expiresAt) {
		c.tweets.Delete(key)
		return nil, false
	}

	return entry.tweet, true
}

// Set stores a tweet in the cache with the configured TTL.
func (c *MemoryCache) Set(username, tweetID string, tweet *domain.Tweet) {
	key := NormalizedKey(username, tweetID)
	now := time.Now()
	c.tweets.Store(key, &cacheEntry{
		tweet:     tweet,
		expiresAt: now.Add(c.ttl),
		scrapedAt: now,
	})
}

// cleanup periodically removes expired entries from the cache.
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		now := time.Now()
		c.tweets.Range(func(key, value interface{}) bool {
			entry := value.(*cacheEntry)
			if now.After(entry.expiresAt) {
				c.tweets.Delete(key)
			}
			return true
		})
	}
}
