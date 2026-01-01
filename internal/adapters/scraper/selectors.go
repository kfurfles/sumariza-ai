package scraper

import (
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// SelectorConfig holds the CSS selectors for scraping Twitter.
type SelectorConfig struct {
	TweetContainer string
	TweetText      string
	Timestamp      string
	AuthorName     string
	AuthorHandle   string
	AuthorAvatar   string
	VerifiedBadge  string
	QuoteContainer string
	QuoteText      string

	mu          sync.RWMutex
	lastModTime time.Time
	filePath    string
}

// rawConfig represents the YAML structure.
type rawConfig struct {
	Tweet struct {
		Container string `yaml:"container"`
		Text      string `yaml:"text"`
		Timestamp string `yaml:"timestamp"`
	} `yaml:"tweet"`
	Author struct {
		Name     string `yaml:"name"`
		Handle   string `yaml:"handle"`
		Avatar   string `yaml:"avatar"`
		Verified string `yaml:"verified_badge"`
	} `yaml:"author"`
	Quote struct {
		Container string `yaml:"container"`
		Text      string `yaml:"text"`
	} `yaml:"quote"`
}

// LoadSelectors loads selector configuration from a YAML file.
// It starts a background goroutine for hot-reloading.
func LoadSelectors(filePath string) (*SelectorConfig, error) {
	config := &SelectorConfig{filePath: filePath}
	if err := config.reload(); err != nil {
		return nil, err
	}

	// Start hot-reload watcher
	go config.watch()

	return config, nil
}

// reload reads the configuration from the file.
func (c *SelectorConfig) reload() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return err
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.TweetContainer = raw.Tweet.Container
	c.TweetText = raw.Tweet.Text
	c.Timestamp = raw.Tweet.Timestamp
	c.AuthorName = raw.Author.Name
	c.AuthorHandle = raw.Author.Handle
	c.AuthorAvatar = raw.Author.Avatar
	c.VerifiedBadge = raw.Author.Verified
	c.QuoteContainer = raw.Quote.Container
	c.QuoteText = raw.Quote.Text

	return nil
}

// watch monitors the configuration file for changes and reloads it.
func (c *SelectorConfig) watch() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		info, err := os.Stat(c.filePath)
		if err != nil {
			continue
		}
		if info.ModTime().After(c.lastModTime) {
			_ = c.reload()
			c.lastModTime = info.ModTime()
		}
	}
}

// GetTweetContainer returns the tweet container selector (thread-safe).
func (c *SelectorConfig) GetTweetContainer() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.TweetContainer
}

// GetTweetText returns the tweet text selector (thread-safe).
func (c *SelectorConfig) GetTweetText() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.TweetText
}
