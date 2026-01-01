package usecases_test

import (
	"context"
	"errors"
	"testing"

	"sumariza-ai/internal/domain"
	"sumariza-ai/internal/usecases"
)

// MockScraper is a mock implementation of TweetScraper.
type MockScraper struct {
	tweet *domain.Tweet
	err   error
}

func (m *MockScraper) Scrape(ctx context.Context, tweetID string) (*domain.Tweet, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tweet, nil
}

// MockCache is a mock implementation of TweetCache.
type MockCache struct {
	tweets map[string]*domain.Tweet
}

func NewMockCache() *MockCache {
	return &MockCache{tweets: make(map[string]*domain.Tweet)}
}

func (m *MockCache) Get(username, tweetID string) (*domain.Tweet, bool) {
	key := "/" + username + "/status/" + tweetID
	tweet, found := m.tweets[key]
	return tweet, found
}

func (m *MockCache) Set(username, tweetID string, tweet *domain.Tweet) {
	key := "/" + username + "/status/" + tweetID
	m.tweets[key] = tweet
}

// ScrapeTweetUseCase tests

func TestScrapeTweetUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockScraper := &MockScraper{
		tweet: &domain.Tweet{
			ID:      "123",
			Content: domain.Content{Text: "Hello world"},
		},
	}
	uc := usecases.NewScrapeTweetUseCase(mockScraper)

	// Act
	tweet, err := uc.Execute(context.Background(), "123", "testuser")

	// Assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tweet.Username != "testuser" {
		t.Errorf("Username: got %v, want testuser", tweet.Username)
	}
	if tweet.URL != "https://x.com/testuser/status/123" {
		t.Errorf("URL: got %v, want https://x.com/testuser/status/123", tweet.URL)
	}
}

func TestScrapeTweetUseCase_Execute_ScraperError(t *testing.T) {
	// Arrange
	expectedErr := errors.New("scraping failed")
	mockScraper := &MockScraper{err: expectedErr}
	uc := usecases.NewScrapeTweetUseCase(mockScraper)

	// Act
	_, err := uc.Execute(context.Background(), "123", "testuser")

	// Assert
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

// GetTweetUseCase tests

func TestGetTweetUseCase_Execute_CacheHit(t *testing.T) {
	// Arrange
	cachedTweet := &domain.Tweet{
		ID:       "123",
		Username: "cached",
		Content:  domain.Content{Text: "Cached tweet"},
	}
	cache := NewMockCache()
	cache.Set("testuser", "123", cachedTweet)

	mockScraper := &MockScraper{
		tweet: &domain.Tweet{ID: "123", Content: domain.Content{Text: "Fresh tweet"}},
	}
	scrapeUC := usecases.NewScrapeTweetUseCase(mockScraper)
	uc := usecases.NewGetTweetUseCase(cache, scrapeUC)

	// Act
	tweet, err := uc.Execute(context.Background(), "123", "testuser")

	// Assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tweet.Content.Text != "Cached tweet" {
		t.Errorf("expected cached tweet, got %v", tweet.Content.Text)
	}
}

func TestGetTweetUseCase_Execute_CacheMiss(t *testing.T) {
	// Arrange
	cache := NewMockCache()
	mockScraper := &MockScraper{
		tweet: &domain.Tweet{
			ID:      "456",
			Content: domain.Content{Text: "Fresh tweet"},
		},
	}
	scrapeUC := usecases.NewScrapeTweetUseCase(mockScraper)
	uc := usecases.NewGetTweetUseCase(cache, scrapeUC)

	// Act
	tweet, err := uc.Execute(context.Background(), "456", "newuser")

	// Assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tweet.Content.Text != "Fresh tweet" {
		t.Errorf("expected fresh tweet, got %v", tweet.Content.Text)
	}
	if tweet.Username != "newuser" {
		t.Errorf("Username: got %v, want newuser", tweet.Username)
	}
}

func TestGetTweetUseCase_Execute_CacheMiss_StoresInCache(t *testing.T) {
	// Arrange
	cache := NewMockCache()
	mockScraper := &MockScraper{
		tweet: &domain.Tweet{
			ID:      "789",
			Content: domain.Content{Text: "New tweet"},
		},
	}
	scrapeUC := usecases.NewScrapeTweetUseCase(mockScraper)
	uc := usecases.NewGetTweetUseCase(cache, scrapeUC)

	// Act
	_, _ = uc.Execute(context.Background(), "789", "user")

	// Verify cache was populated
	cachedTweet, found := cache.Get("user", "789")

	// Assert
	if !found {
		t.Error("expected tweet to be cached after scrape")
	}
	if cachedTweet.Content.Text != "New tweet" {
		t.Errorf("cached tweet text: got %v, want 'New tweet'", cachedTweet.Content.Text)
	}
}

func TestGetTweetUseCase_Execute_ScraperError(t *testing.T) {
	// Arrange
	cache := NewMockCache()
	mockScraper := &MockScraper{err: domain.ErrScrapingFailed}
	scrapeUC := usecases.NewScrapeTweetUseCase(mockScraper)
	uc := usecases.NewGetTweetUseCase(cache, scrapeUC)

	// Act
	_, err := uc.Execute(context.Background(), "999", "user")

	// Assert
	if err != domain.ErrScrapingFailed {
		t.Errorf("expected ErrScrapingFailed, got %v", err)
	}
}
