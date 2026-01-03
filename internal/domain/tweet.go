// Package domain contains the core business entities and rules.
package domain

import "time"

// Tweet represents a single Twitter/X post.
type Tweet struct {
	ID       string
	URL      string // Original Twitter URL
	Username string // Extracted from input URL
	Author   Author
	Content  Content
	Partial  bool // True if some optional data is missing
}

// Author represents the tweet author's information.
type Author struct {
	Name         string
	Handle       string
	AvatarURL    string
	Verified     bool
	VerifiedType VerifiedType
}

// VerifiedType represents the type of verification badge.
type VerifiedType string

const (
	VerifiedNone VerifiedType = "none"
	VerifiedBlue VerifiedType = "blue"
	VerifiedGold VerifiedType = "gold"
	VerifiedGray VerifiedType = "gray"
)

// Content represents the tweet's content.
type Content struct {
	Text        string
	CreatedAt   time.Time
	QuotedTweet *QuotedTweet  // Limited to 1 level only
	Direction   TextDirection // LTR or RTL - extracted from Twitter's dir attribute
}

// QuotedTweet represents a quoted tweet within the main tweet.
// Any quotes inside this quoted tweet are ignored (no recursive parsing).
type QuotedTweet struct {
	ID     string
	URL    string
	Author Author
	Text   string
}

// TextDirection represents the text direction (LTR or RTL).
type TextDirection string

const (
	LTR TextDirection = "ltr"
	RTL TextDirection = "rtl"
)
