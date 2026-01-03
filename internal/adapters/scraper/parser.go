package scraper

import (
	"context"
	"regexp"
	"strings"
	"time"

	"sumariza-ai/internal/domain"
	"sumariza-ai/pkg/log"

	"github.com/chromedp/chromedp"
)

// TwitterScraper scrapes tweets from Twitter using Chromedp.
type TwitterScraper struct {
	pool      *BrowserPool
	selectors *SelectorConfig
}

// NewTwitterScraper creates a new Twitter scraper.
func NewTwitterScraper(pool *BrowserPool, selectors *SelectorConfig) *TwitterScraper {
	return &TwitterScraper{
		pool:      pool,
		selectors: selectors,
	}
}

// Scrape fetches and parses a tweet from Twitter.
func (s *TwitterScraper) Scrape(ctx context.Context, tweetID string) (*domain.Tweet, error) {
	// Use /i/status/{id} format for scraping (doesn't require username)
	url := "https://twitter.com/i/status/" + tweetID

	log.GlobalDebug("scrape starting", "tweet_id", tweetID, "url", url)
	startTime := time.Now()

	var html string

	// Execute scraping with exclusive tab access (backpressure)
	// Using WithTabCtx to properly propagate context cancellation/timeout
	err := s.pool.WithTabCtx(ctx, func(tabCtx context.Context) error {
		// Step 1: Navigate to the URL
		log.GlobalDebug("scrape step: navigating", "tweet_id", tweetID)
		navStart := time.Now()
		if err := chromedp.Run(tabCtx, chromedp.Navigate(url)); err != nil {
			log.GlobalError("scrape navigation failed",
				"tweet_id", tweetID,
				"error", err,
				"duration_ms", time.Since(navStart).Milliseconds())
			return err
		}
		log.GlobalDebug("scrape step: navigation complete",
			"tweet_id", tweetID,
			"duration_ms", time.Since(navStart).Milliseconds())

		// Check context before continuing
		if tabCtx.Err() != nil {
			log.GlobalWarn("scrape context canceled after navigation",
				"tweet_id", tweetID,
				"error", tabCtx.Err())
			return tabCtx.Err()
		}

		// Step 2: Wait for tweet container
		log.GlobalDebug("scrape step: waiting for container", "tweet_id", tweetID)
		containerStart := time.Now()
		containerSelector := s.selectors.GetTweetContainer()
		if err := chromedp.Run(tabCtx, chromedp.WaitVisible(containerSelector, chromedp.ByQuery)); err != nil {
			log.GlobalError("scrape wait container failed",
				"tweet_id", tweetID,
				"selector", containerSelector,
				"error", err,
				"duration_ms", time.Since(containerStart).Milliseconds())
			return err
		}
		log.GlobalDebug("scrape step: container visible",
			"tweet_id", tweetID,
			"duration_ms", time.Since(containerStart).Milliseconds())

		// Check context before continuing
		if tabCtx.Err() != nil {
			log.GlobalWarn("scrape context canceled after container wait",
				"tweet_id", tweetID,
				"error", tabCtx.Err())
			return tabCtx.Err()
		}

		// Step 3: Wait for tweet text
		log.GlobalDebug("scrape step: waiting for text", "tweet_id", tweetID)
		textStart := time.Now()
		textSelector := s.selectors.GetTweetText()
		if err := chromedp.Run(tabCtx, chromedp.WaitVisible(textSelector, chromedp.ByQuery)); err != nil {
			log.GlobalError("scrape wait text failed",
				"tweet_id", tweetID,
				"selector", textSelector,
				"error", err,
				"duration_ms", time.Since(textStart).Milliseconds())
			return err
		}
		log.GlobalDebug("scrape step: text visible",
			"tweet_id", tweetID,
			"duration_ms", time.Since(textStart).Milliseconds())

		// Check context before continuing
		if tabCtx.Err() != nil {
			log.GlobalWarn("scrape context canceled after text wait",
				"tweet_id", tweetID,
				"error", tabCtx.Err())
			return tabCtx.Err()
		}

		// Step 4: Extract HTML
		log.GlobalDebug("scrape step: extracting html", "tweet_id", tweetID)
		htmlStart := time.Now()
		if err := chromedp.Run(tabCtx, chromedp.OuterHTML("html", &html)); err != nil {
			log.GlobalError("scrape html extraction failed",
				"tweet_id", tweetID,
				"error", err,
				"duration_ms", time.Since(htmlStart).Milliseconds())
			return err
		}
		log.GlobalDebug("scrape step: html extracted",
			"tweet_id", tweetID,
			"html_length", len(html),
			"duration_ms", time.Since(htmlStart).Milliseconds())

		return nil
	})

	if err != nil {
		log.GlobalError("scrape failed",
			"tweet_id", tweetID,
			"error", err,
			"total_duration_ms", time.Since(startTime).Milliseconds())
		return nil, domain.ErrScrapingFailed
	}

	log.GlobalDebug("scrape complete, parsing html",
		"tweet_id", tweetID,
		"total_duration_ms", time.Since(startTime).Milliseconds())

	tweet, partial := s.parseHTML(html, tweetID)

	// Text is essential - fail if not found
	if tweet.Content.Text == "" {
		log.GlobalError("scrape text not found in html",
			"tweet_id", tweetID,
			"html_length", len(html))
		return nil, domain.ErrTextNotFound
	}

	tweet.Partial = partial

	if partial {
		log.GlobalDebug("partial data retrieved", "tweet_id", tweetID)
	}

	log.GlobalInfo("scrape success",
		"tweet_id", tweetID,
		"partial", partial,
		"total_duration_ms", time.Since(startTime).Milliseconds())

	return tweet, nil
}

// parseHTML extracts tweet data from the HTML.
func (s *TwitterScraper) parseHTML(html, tweetID string) (*domain.Tweet, bool) {
	partial := false

	tweet := &domain.Tweet{
		ID: tweetID,
	}

	// Parse author info
	tweet.Author, partial = s.parseAuthor(html)

	// Parse content
	tweet.Content = s.parseContent(html)

	return tweet, partial
}

// parseAuthor extracts author information from the HTML.
func (s *TwitterScraper) parseAuthor(html string) (domain.Author, bool) {
	partial := false
	author := domain.Author{}

	// Extract author name and handle from User-Name testid
	// The structure contains "DisplayName @handle" - we split by @
	name, handle := extractNameAndHandle(html)
	if name != "" {
		author.Name = name
	} else {
		partial = true
	}

	if handle != "" {
		author.Handle = handle
	} else {
		// Fallback to URL-based handle extraction
		handleMatch := extractHandleFromURL(html)
		if handleMatch != "" {
			author.Handle = handleMatch
		} else {
			partial = true
		}
	}

	// Extract avatar
	avatarMatch := extractAvatar(html)
	if avatarMatch != "" {
		author.AvatarURL = avatarMatch
	} else {
		partial = true
	}

	// Check for verified badge
	if strings.Contains(html, `data-testid="icon-verified"`) {
		author.Verified = true
		author.VerifiedType = detectVerifiedType(html)
	}

	return author, partial
}

// parseContent extracts the tweet content from the HTML.
func (s *TwitterScraper) parseContent(html string) domain.Content {
	content := domain.Content{
		Direction: domain.LTR,
	}

	// Extract tweet text (already cleaned with newlines preserved)
	textMatch := extractTweetText(html)
	if textMatch != "" {
		content.Text = textMatch
	}

	// Extract text direction
	content.Direction = extractTextDirection(html)

	// Extract timestamp
	content.CreatedAt = extractTimestamp(html)

	// Extract quoted tweet (1 level only)
	content.QuotedTweet = extractQuotedTweet(html)

	return content
}

// extractTweetText extracts the main tweet text from HTML, preserving links with full URLs.
func extractTweetText(html string) string {
	// Find the tweetText container - Twitter uses div with nested spans
	// The content may be in a div that contains multiple spans with the actual text
	re := regexp.MustCompile(`data-testid="tweetText"[^>]*>([\s\S]*?)</div>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		// Try without closing div (might be deeply nested)
		re2 := regexp.MustCompile(`data-testid="tweetText"[^>]*>([\s\S]*?)<div`)
		matches = re2.FindStringSubmatch(html)
		if len(matches) < 2 {
			return ""
		}
	}

	content := matches[1]

	// Replace links with their full href URLs
	// Twitter uses <a href="FULL_URL">truncated_text</a>
	// We want to preserve the full URL from href
	content = preserveLinks(content)

	// Convert closing </span> to preserve line structure
	// Twitter puts newlines inside <span> tags
	content = regexp.MustCompile(`</span>`).ReplaceAllString(content, "")

	// Remove remaining HTML tags (spans, etc.) but keep the processed links
	// This also converts <br>, </div>, </p> to newlines
	content = stripHTMLKeepLinks(content)

	// Clean text while preserving newlines for formatting
	return cleanTextPreserveNewlines(content)
}

// preserveLinks replaces Twitter's truncated link text with the full URL from href.
func preserveLinks(html string) string {
	// Simple regex to match <a> tags - captures href and the entire link content
	// Using [\s\S]*? for content to handle nested tags
	linkRe := regexp.MustCompile(`<a[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)

	return linkRe.ReplaceAllStringFunc(html, func(match string) string {
		submatches := linkRe.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			href := submatches[1]
			// Skip Twitter internal links (hashtags, mentions, etc.)
			if strings.HasPrefix(href, "/") ||
				strings.Contains(href, "twitter.com/hashtag") ||
				strings.Contains(href, "twitter.com/search") ||
				strings.Contains(href, "x.com/hashtag") ||
				strings.Contains(href, "x.com/search") {
				// For hashtags/mentions, just return the visible text
				if len(submatches) >= 3 {
					return " " + stripHTML(submatches[2]) + " "
				}
				return ""
			}
			// For external links (including t.co redirects), use the full URL from href
			// Mark it with special delimiters so we can convert back to link later
			return " [[LINK:" + href + "]] "
		}
		return match
	})
}

// stripHTMLKeepLinks removes HTML tags but preserves our link markers, emojis, and converts line breaks.
func stripHTMLKeepLinks(html string) string {
	// Convert line break elements to newlines BEFORE removing tags
	html = regexp.MustCompile(`<br\s*/?\s*>`).ReplaceAllString(html, "\n")
	html = regexp.MustCompile(`</div>`).ReplaceAllString(html, "\n")
	html = regexp.MustCompile(`</p>`).ReplaceAllString(html, "\n")

	// Preserve emojis from <img alt="emoji"> tags (Twitter renders emojis as images)
	// Replace <img alt="X" ...> with the alt text (emoji)
	emojiRe := regexp.MustCompile(`<img[^>]*alt="([^"]*)"[^>]*>`)
	html = emojiRe.ReplaceAllString(html, "$1")

	// Remove remaining HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, "")
	return text
}

// extractNameAndHandle extracts display name and handle from User-Name testid.
// Twitter structure: [data-testid="User-Name"] contains "DisplayName @handle"
func extractNameAndHandle(html string) (name, handle string) {
	// Find the User-Name container
	re := regexp.MustCompile(`data-testid="User-Name"[^>]*>([\s\S]*?)</div></div></div>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return "", ""
	}

	// Strip HTML and get plain text
	content := stripHTML(matches[1])
	content = cleanText(content)

	// Split by @ to separate name from handle
	// Format: "Display Name @handle" or "Display Name@handle"
	parts := strings.Split(content, "@")
	if len(parts) >= 2 {
		name = cleanText(parts[0])
		// Handle might have extra text after it, take first word
		handlePart := cleanText(parts[1])
		handleWords := strings.Fields(handlePart)
		if len(handleWords) > 0 {
			handle = handleWords[0]
		}
	} else if len(parts) == 1 {
		// No @ found, the whole thing might be the name
		name = cleanText(parts[0])
	}

	return name, handle
}

// extractHandleFromURL extracts the @handle from status URL in HTML.
func extractHandleFromURL(html string) string {
	re := regexp.MustCompile(`href="/([a-zA-Z0-9_]+)/status/`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractAvatar extracts the avatar URL from HTML.
func extractAvatar(html string) string {
	re := regexp.MustCompile(`data-testid="Tweet-User-Avatar"[^>]*>.*?<img[^>]*src="([^"]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractTextDirection reads the dir attribute from Twitter's HTML.
func extractTextDirection(html string) domain.TextDirection {
	if strings.Contains(html, `dir="rtl"`) {
		return domain.RTL
	}
	return domain.LTR
}

// extractTimestamp extracts the tweet timestamp from HTML.
func extractTimestamp(html string) time.Time {
	re := regexp.MustCompile(`<time[^>]*datetime="([^"]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		t, err := time.Parse(time.RFC3339, matches[1])
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

// extractQuotedTweet extracts a quoted tweet (1 level only).
func extractQuotedTweet(html string) *domain.QuotedTweet {
	if !strings.Contains(html, `data-testid="quoteTweet"`) {
		return nil
	}

	// Extract quoted tweet section
	re := regexp.MustCompile(`data-testid="quoteTweet"[\s\S]*?data-testid="tweetText"[^>]*>([^<]+)`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return nil
	}

	return &domain.QuotedTweet{
		Text: cleanText(matches[1]),
	}
}

// detectVerifiedType determines the type of verification badge.
func detectVerifiedType(html string) domain.VerifiedType {
	// Check for gold badge (organizations)
	if strings.Contains(html, "gold") || strings.Contains(html, "Gold") {
		return domain.VerifiedGold
	}
	// Check for gray badge (government)
	if strings.Contains(html, "gray") || strings.Contains(html, "Gray") {
		return domain.VerifiedGray
	}
	// Default to blue
	return domain.VerifiedBlue
}

// extractBySelector is a simple helper to extract content by a pattern.
func extractBySelector(html, selector string) string {
	// This is a simplified extraction - real implementation would use proper HTML parsing
	if selector == "" {
		return ""
	}
	return ""
}

// cleanText removes extra whitespace and trims the text.
func cleanText(text string) string {
	// Remove multiple spaces
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// cleanTextPreserveNewlines normalizes horizontal whitespace but preserves line breaks.
func cleanTextPreserveNewlines(text string) string {
	// Normalize horizontal whitespace only (spaces, tabs) - not newlines
	re := regexp.MustCompile(`[^\S\n]+`)
	text = re.ReplaceAllString(text, " ")

	// Collapse multiple newlines to max 2 (paragraph separation)
	re2 := regexp.MustCompile(`\n{3,}`)
	text = re2.ReplaceAllString(text, "\n\n")

	// Trim spaces from each line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// stripHTML removes HTML tags from a string, preserving emoji alt text.
func stripHTML(html string) string {
	// Preserve emojis from <img alt="emoji"> tags (Twitter renders emojis as images)
	emojiRe := regexp.MustCompile(`<img[^>]*alt="([^"]*)"[^>]*>`)
	html = emojiRe.ReplaceAllString(html, "$1")

	// Remove remaining HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, "")
	return cleanText(text)
}

// extractHasVideo checks if the tweet contains a video.
// Twitter uses data-testid="videoPlayer" or "videoComponent" for videos.
func extractHasVideo(html string) bool {
	return strings.Contains(html, `data-testid="videoPlayer"`) ||
		strings.Contains(html, `data-testid="videoComponent"`)
}

// extractImages extracts image URLs from the tweet.
// Twitter uses data-testid="tweetPhoto" for images.
func extractImages(html string) []string {
	if !strings.Contains(html, `data-testid="tweetPhoto"`) {
		return nil
	}

	// Find image URLs within tweetPhoto containers
	// Twitter uses <img src="..."> inside these containers
	re := regexp.MustCompile(`data-testid="tweetPhoto"[^>]*>[\s\S]*?<img[^>]*src="([^"]+)"`)
	matches := re.FindAllStringSubmatch(html, -1)

	var images []string
	for _, match := range matches {
		if len(match) > 1 {
			images = append(images, match[1])
		}
	}

	return images
}
