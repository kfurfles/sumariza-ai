package domain

import "errors"

var (
	// ErrTweetNotFound is returned when the tweet does not exist or was deleted.
	ErrTweetNotFound = errors.New("tweet not found or deleted")

	// ErrTweetPrivate is returned when the tweet is from a private account.
	ErrTweetPrivate = errors.New("tweet is from a private account")

	// ErrInvalidURL is returned when the URL format is invalid.
	ErrInvalidURL = errors.New("invalid tweet URL format")

	// ErrScrapingFailed is returned when the scraping operation fails.
	ErrScrapingFailed = errors.New("failed to scrape tweet")

	// ErrRateLimited is returned when rate limit is exceeded.
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrTextNotFound is returned when essential tweet text is not found.
	// This covers sensitive/age-gated content scenarios.
	ErrTextNotFound = errors.New("essential tweet text not found")
)

