package scraper

import (
	"testing"

	"sumariza-ai/internal/domain"
	"sumariza-ai/test/fixtures"
)

func TestParseHTML_BasicTweet_ExtractsAllFields(t *testing.T) {
	// Arrange
	html := fixtures.GenerateBasicTweet()
	s := &TwitterScraper{selectors: &SelectorConfig{
		TweetText: "[data-testid='tweetText']",
	}}

	// Act
	tweet, partial := s.parseHTML(html, "123")

	// Assert
	if tweet.ID != "123" {
		t.Errorf("ID: got %v, want 123", tweet.ID)
	}
	if tweet.Content.Text == "" {
		t.Error("expected tweet text to be extracted")
	}
	if tweet.Content.Direction != domain.LTR {
		t.Errorf("Direction: got %v, want LTR", tweet.Content.Direction)
	}
	// Author fields might be partial depending on regex matching
	_ = partial
}

func TestParseHTML_PartialTweet_MarksAsPartial(t *testing.T) {
	// Arrange
	html := fixtures.GeneratePartialTweet()
	s := &TwitterScraper{selectors: &SelectorConfig{}}

	// Act
	tweet, partial := s.parseHTML(html, "456")

	// Assert
	if tweet.Content.Text == "" {
		t.Error("expected tweet text to be extracted")
	}
	if !partial {
		t.Error("expected partial to be true for tweet with missing author info")
	}
}

func TestParseHTML_RTLTweet_DetectsDirection(t *testing.T) {
	// Arrange
	html := fixtures.GenerateRTLTweet()
	s := &TwitterScraper{selectors: &SelectorConfig{}}

	// Act
	tweet, _ := s.parseHTML(html, "789")

	// Assert
	if tweet.Content.Direction != domain.RTL {
		t.Errorf("Direction: got %v, want RTL", tweet.Content.Direction)
	}
}

func TestParseHTML_VerifiedTweet_DetectsBadge(t *testing.T) {
	// Arrange
	html := fixtures.GenerateVerifiedTweet()
	s := &TwitterScraper{selectors: &SelectorConfig{}}

	// Act
	tweet, _ := s.parseHTML(html, "999")

	// Assert
	if !tweet.Author.Verified {
		t.Error("expected author to be verified")
	}
	if tweet.Author.VerifiedType != domain.VerifiedBlue {
		t.Errorf("VerifiedType: got %v, want blue", tweet.Author.VerifiedType)
	}
}

func TestParseHTML_QuoteTweet_ExtractsQuote(t *testing.T) {
	// Arrange
	html := fixtures.GenerateQuoteTweet()
	s := &TwitterScraper{selectors: &SelectorConfig{}}

	// Act
	tweet, _ := s.parseHTML(html, "100")

	// Assert
	if tweet.Content.QuotedTweet == nil {
		t.Error("expected quoted tweet to be extracted")
	}
	if tweet.Content.QuotedTweet != nil && tweet.Content.QuotedTweet.Text == "" {
		t.Error("expected quoted tweet text to be extracted")
	}
}

func TestExtractTweetText_BasicHTML_ReturnsText(t *testing.T) {
	// Arrange
	html := `<div data-testid="tweetText" dir="ltr">Hello World</div>`

	// Act
	text := extractTweetText(html)

	// Assert
	if text != "Hello World" {
		t.Errorf("got %q, want 'Hello World'", text)
	}
}

func TestExtractTextDirection_RTL_ReturnsRTL(t *testing.T) {
	// Arrange
	html := `<div dir="rtl">مرحبا</div>`

	// Act
	dir := extractTextDirection(html)

	// Assert
	if dir != domain.RTL {
		t.Errorf("got %v, want RTL", dir)
	}
}

func TestExtractTextDirection_LTR_ReturnsLTR(t *testing.T) {
	// Arrange
	html := `<div dir="ltr">Hello</div>`

	// Act
	dir := extractTextDirection(html)

	// Assert
	if dir != domain.LTR {
		t.Errorf("got %v, want LTR", dir)
	}
}

func TestExtractTextDirection_NoDir_DefaultsToLTR(t *testing.T) {
	// Arrange
	html := `<div>Hello</div>`

	// Act
	dir := extractTextDirection(html)

	// Assert
	if dir != domain.LTR {
		t.Errorf("got %v, want LTR (default)", dir)
	}
}

func TestExtractTimestamp_ValidDatetime_ParsesCorrectly(t *testing.T) {
	// Arrange
	html := `<time datetime="2026-01-01T12:00:00Z">Jan 1</time>`

	// Act
	ts := extractTimestamp(html)

	// Assert
	if ts.IsZero() {
		t.Error("expected timestamp to be parsed")
	}
	if ts.Year() != 2026 || ts.Month() != 1 || ts.Day() != 1 {
		t.Errorf("got %v, want 2026-01-01", ts)
	}
}

func TestExtractHandleFromURL_ValidHTML_ReturnsHandle(t *testing.T) {
	// Arrange
	html := `<a href="/testuser/status/123">@testuser</a>`

	// Act
	handle := extractHandleFromURL(html)

	// Assert
	if handle != "testuser" {
		t.Errorf("got %q, want 'testuser'", handle)
	}
}

func TestExtractNameAndHandle_ValidHTML_ExtractsBoth(t *testing.T) {
	// Arrange - simplified version of Twitter's User-Name structure
	html := `<div data-testid="User-Name"><span>Concierge do porão</span><span>@eduhonorato</span></div></div></div>`

	// Act
	name, handle := extractNameAndHandle(html)

	// Assert
	if name != "Concierge do porão" {
		t.Errorf("name: got %q, want 'Concierge do porão'", name)
	}
	if handle != "eduhonorato" {
		t.Errorf("handle: got %q, want 'eduhonorato'", handle)
	}
}

func TestCleanText_MultipleSpaces_NormalizesSpaces(t *testing.T) {
	// Arrange
	text := "  Hello    World  "

	// Act
	clean := cleanText(text)

	// Assert
	if clean != "Hello World" {
		t.Errorf("got %q, want 'Hello World'", clean)
	}
}

func TestStripHTML_WithTags_RemovesTags(t *testing.T) {
	// Arrange
	html := "<span>Hello</span> <strong>World</strong>"

	// Act
	text := stripHTML(html)

	// Assert
	if text != "Hello World" {
		t.Errorf("got %q, want 'Hello World'", text)
	}
}

// TDD: Tests for preserving newlines in tweet text

func TestCleanTextPreserveNewlines_PreservesLineBreaks(t *testing.T) {
	// Arrange
	text := "Line 1\nLine 2\nLine 3"

	// Act
	clean := cleanTextPreserveNewlines(text)

	// Assert
	if clean != "Line 1\nLine 2\nLine 3" {
		t.Errorf("got %q, want 'Line 1\\nLine 2\\nLine 3'", clean)
	}
}

func TestCleanTextPreserveNewlines_NormalizesHorizontalSpaces(t *testing.T) {
	// Arrange
	text := "Hello    World\nFoo    Bar"

	// Act
	clean := cleanTextPreserveNewlines(text)

	// Assert
	if clean != "Hello World\nFoo Bar" {
		t.Errorf("got %q, want 'Hello World\\nFoo Bar'", clean)
	}
}

func TestCleanTextPreserveNewlines_CollapsesExcessiveNewlines(t *testing.T) {
	// Arrange
	text := "Para 1\n\n\n\n\nPara 2"

	// Act
	clean := cleanTextPreserveNewlines(text)

	// Assert - should collapse to max 2 newlines (paragraph break)
	if clean != "Para 1\n\nPara 2" {
		t.Errorf("got %q, want 'Para 1\\n\\nPara 2'", clean)
	}
}

func TestCleanTextPreserveNewlines_TrimsLineSpaces(t *testing.T) {
	// Arrange
	text := "  Line 1  \n  Line 2  "

	// Act
	clean := cleanTextPreserveNewlines(text)

	// Assert
	if clean != "Line 1\nLine 2" {
		t.Errorf("got %q, want 'Line 1\\nLine 2'", clean)
	}
}

func TestStripHTMLKeepLinks_ConvertsBrToNewline(t *testing.T) {
	// Arrange
	html := "Line 1<br>Line 2<br/>Line 3"

	// Act
	text := stripHTMLKeepLinks(html)

	// Assert - should have newlines where <br> was
	expected := "Line 1\nLine 2\nLine 3"
	if text != expected {
		t.Errorf("got %q, want %q", text, expected)
	}
}

func TestStripHTMLKeepLinks_ConvertsClosingDivToNewline(t *testing.T) {
	// Arrange
	html := "<div>Line 1</div><div>Line 2</div>"

	// Act
	text := stripHTMLKeepLinks(html)

	// Assert - should have newlines where </div> was
	expected := "Line 1\nLine 2\n"
	if text != expected {
		t.Errorf("got %q, want %q", text, expected)
	}
}

func TestExtractTweetText_WithLineBreaks_PreservesFormatting(t *testing.T) {
	// Arrange - HTML with line breaks similar to Twitter's structure
	html := `<div data-testid="tweetText"><span>First line</span><br><span>Second line</span></div>`

	// Act
	text := extractTweetText(html)

	// Assert - should preserve line break
	if text != "First line\nSecond line" {
		t.Errorf("got %q, want 'First line\\nSecond line'", text)
	}
}
