package usecases

import (
	"context"

	"sumariza-ai/internal/domain"
	"sumariza-ai/pkg/log"
)

// TweetScraper defines the interface for scraping tweets.
type TweetScraper interface {
	Scrape(ctx context.Context, tweetID string) (*domain.Tweet, error)
}

// ScrapeTweetUseCase handles the scraping of a single tweet.
type ScrapeTweetUseCase struct {
	scraper TweetScraper
}

// NewScrapeTweetUseCase creates a new ScrapeTweetUseCase.
func NewScrapeTweetUseCase(scraper TweetScraper) *ScrapeTweetUseCase {
	return &ScrapeTweetUseCase{scraper: scraper}
}

// Execute scrapes a tweet and sets the username and URL.
func (uc *ScrapeTweetUseCase) Execute(ctx context.Context, tweetID, username string) (*domain.Tweet, error) {
	tweet, err := uc.scraper.Scrape(ctx, tweetID)
	if err != nil {
		return nil, err
	}

	// Set username from input URL
	tweet.Username = username
	tweet.URL = "https://x.com/" + username + "/status/" + tweetID

	// Log if partial data (for debugging)
	if tweet.Partial {
		log.GlobalWarn("partial data retrieved", "tweet_id", tweetID)
	}

	return tweet, nil
}
