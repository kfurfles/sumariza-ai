package usecases

import (
	"context"

	"sumariza-ai/internal/domain"
	"sumariza-ai/pkg/log"
)

// TweetCache defines the interface for caching tweets.
type TweetCache interface {
	Get(username, tweetID string) (*domain.Tweet, bool)
	Set(username, tweetID string, tweet *domain.Tweet)
}

// GetTweetUseCase handles retrieving tweets with cache-first strategy.
type GetTweetUseCase struct {
	cache   TweetCache
	scraper *ScrapeTweetUseCase
}

// NewGetTweetUseCase creates a new GetTweetUseCase.
func NewGetTweetUseCase(cache TweetCache, scraper *ScrapeTweetUseCase) *GetTweetUseCase {
	return &GetTweetUseCase{
		cache:   cache,
		scraper: scraper,
	}
}

// Execute retrieves a tweet, checking cache first before scraping.
func (uc *GetTweetUseCase) Execute(ctx context.Context, tweetID, username string) (*domain.Tweet, error) {
	// Check cache first (key is normalized: /{username}/status/{id})
	if tweet, found := uc.cache.Get(username, tweetID); found {
		log.GlobalDebugCtx(ctx, "cache hit", "username", username, "tweet_id", tweetID)
		return tweet, nil
	}

	log.GlobalDebugCtx(ctx, "cache miss, scraping", "username", username, "tweet_id", tweetID)

	// Cache miss: scrape
	tweet, err := uc.scraper.Execute(ctx, tweetID, username)
	if err != nil {
		return nil, err
	}

	// Store in cache with normalized key
	uc.cache.Set(username, tweetID, tweet)

	return tweet, nil
}
