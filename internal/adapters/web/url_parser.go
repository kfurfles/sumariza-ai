package web

import (
	"regexp"

	"sumariza-ai/internal/domain"
)

// tweetURLRegex matches Twitter/X URLs and extracts username and tweet ID.
// Accepts twitter.com, x.com, and mobile.twitter.com.
// Query parameters are preserved in the URL but ignored during parsing.
var tweetURLRegex = regexp.MustCompile(
	`^https?://(twitter\.com|x\.com|mobile\.twitter\.com)/(\w+)/status/(\d+)`,
)

// ParseTweetURL extracts the username and tweet ID from a Twitter/X URL.
// Returns domain.ErrInvalidURL if the URL format is invalid.
func ParseTweetURL(url string) (username string, tweetID string, err error) {
	matches := tweetURLRegex.FindStringSubmatch(url)
	if matches == nil || len(matches) < 4 {
		return "", "", domain.ErrInvalidURL
	}
	return matches[2], matches[3], nil
}

