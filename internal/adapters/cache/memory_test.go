package cache_test

import (
	"testing"
	"time"

	"sumariza-ai/internal/adapters/cache"
	"sumariza-ai/internal/domain"
)

func TestNormalizedKey_ReturnsCorrectFormat(t *testing.T) {
	// Arrange
	username := "elonmusk"
	tweetID := "1234567890"
	expected := "/elonmusk/status/1234567890"

	// Act
	key := cache.NormalizedKey(username, tweetID)

	// Assert
	if key != expected {
		t.Errorf("got %v, want %v", key, expected)
	}
}

func TestMemoryCache_SetAndGet_ReturnsTweet(t *testing.T) {
	// Arrange
	c := cache.NewMemoryCache(5 * time.Minute)
	tweet := &domain.Tweet{
		ID:       "123",
		Username: "testuser",
		Content: domain.Content{
			Text: "Hello world",
		},
	}

	// Act
	c.Set("testuser", "123", tweet)
	result, found := c.Get("testuser", "123")

	// Assert
	if !found {
		t.Error("expected tweet to be found")
	}
	if result.ID != tweet.ID {
		t.Errorf("ID: got %v, want %v", result.ID, tweet.ID)
	}
	if result.Content.Text != tweet.Content.Text {
		t.Errorf("Text: got %v, want %v", result.Content.Text, tweet.Content.Text)
	}
}

func TestMemoryCache_GetNonExistent_ReturnsNotFound(t *testing.T) {
	// Arrange
	c := cache.NewMemoryCache(5 * time.Minute)

	// Act
	_, found := c.Get("nonexistent", "999")

	// Assert
	if found {
		t.Error("expected tweet to not be found")
	}
}

func TestMemoryCache_ExpiredEntry_ReturnsNotFound(t *testing.T) {
	// Arrange
	c := cache.NewMemoryCache(10 * time.Millisecond)
	tweet := &domain.Tweet{
		ID:       "123",
		Username: "testuser",
	}

	// Act
	c.Set("testuser", "123", tweet)
	time.Sleep(20 * time.Millisecond) // Wait for expiration
	_, found := c.Get("testuser", "123")

	// Assert
	if found {
		t.Error("expected expired tweet to not be found")
	}
}

func TestMemoryCache_DifferentUsers_SameTweetID_AreSeparate(t *testing.T) {
	// Arrange
	c := cache.NewMemoryCache(5 * time.Minute)
	tweet1 := &domain.Tweet{
		ID:       "123",
		Username: "user1",
		Content:  domain.Content{Text: "Tweet from user1"},
	}
	tweet2 := &domain.Tweet{
		ID:       "123",
		Username: "user2",
		Content:  domain.Content{Text: "Tweet from user2"},
	}

	// Act
	c.Set("user1", "123", tweet1)
	c.Set("user2", "123", tweet2)
	result1, found1 := c.Get("user1", "123")
	result2, found2 := c.Get("user2", "123")

	// Assert
	if !found1 || !found2 {
		t.Error("expected both tweets to be found")
	}
	if result1.Content.Text != "Tweet from user1" {
		t.Errorf("user1 tweet: got %v, want 'Tweet from user1'", result1.Content.Text)
	}
	if result2.Content.Text != "Tweet from user2" {
		t.Errorf("user2 tweet: got %v, want 'Tweet from user2'", result2.Content.Text)
	}
}

func TestMemoryCache_OverwriteExisting_UpdatesTweet(t *testing.T) {
	// Arrange
	c := cache.NewMemoryCache(5 * time.Minute)
	original := &domain.Tweet{
		ID:      "123",
		Content: domain.Content{Text: "Original"},
	}
	updated := &domain.Tweet{
		ID:      "123",
		Content: domain.Content{Text: "Updated"},
	}

	// Act
	c.Set("user", "123", original)
	c.Set("user", "123", updated)
	result, found := c.Get("user", "123")

	// Assert
	if !found {
		t.Error("expected tweet to be found")
	}
	if result.Content.Text != "Updated" {
		t.Errorf("got %v, want 'Updated'", result.Content.Text)
	}
}
