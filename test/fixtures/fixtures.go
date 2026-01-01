// Package fixtures provides HTML test fixtures for testing the parser.
package fixtures

// GenerateBasicTweet creates HTML fixture for a simple text tweet.
func GenerateBasicTweet() string {
	return `
<!DOCTYPE html>
<html>
<head><title>Tweet</title></head>
<body>
<article data-testid="tweet">
    <div data-testid="User-Name">
        <span>John Doe</span>
        <a href="/johndoe/status/123">@johndoe</a>
    </div>
    <a href="/johndoe/status/123">
        <img data-testid="Tweet-User-Avatar" src="https://example.com/avatar.jpg"/>
    </a>
    <div data-testid="tweetText" dir="ltr">
        This is a test tweet content.
    </div>
    <time datetime="2026-01-01T12:00:00Z">12:00 PM · Jan 1, 2026</time>
</article>
</body>
</html>
`
}

// GeneratePartialTweet creates HTML fixture with missing optional fields.
func GeneratePartialTweet() string {
	return `
<!DOCTYPE html>
<html>
<head><title>Tweet</title></head>
<body>
<article data-testid="tweet">
    <div data-testid="tweetText" dir="ltr">
        This is a test tweet content with missing author info.
    </div>
</article>
</body>
</html>
`
}

// GenerateRTLTweet creates HTML fixture with Arabic text (RTL).
func GenerateRTLTweet() string {
	return `
<!DOCTYPE html>
<html>
<head><title>Tweet</title></head>
<body>
<article data-testid="tweet">
    <div data-testid="User-Name">
        <span>Ahmed</span>
        <a href="/ahmed/status/456">@ahmed</a>
    </div>
    <div data-testid="tweetText" dir="rtl">
        مرحبا بالعالم
    </div>
    <time datetime="2026-01-01T12:00:00Z">12:00 PM · Jan 1, 2026</time>
</article>
</body>
</html>
`
}

// GenerateVerifiedTweet creates HTML fixture with a verified user.
func GenerateVerifiedTweet() string {
	return `
<!DOCTYPE html>
<html>
<head><title>Tweet</title></head>
<body>
<article data-testid="tweet">
    <div data-testid="User-Name">
        <span>Verified User</span>
        <a href="/verified/status/789">@verified</a>
        <svg data-testid="icon-verified"></svg>
    </div>
    <div data-testid="tweetText" dir="ltr">
        This is from a verified account.
    </div>
    <time datetime="2026-01-01T14:30:00Z">2:30 PM · Jan 1, 2026</time>
</article>
</body>
</html>
`
}

// GenerateQuoteTweet creates HTML fixture with a quoted tweet.
func GenerateQuoteTweet() string {
	return `
<!DOCTYPE html>
<html>
<head><title>Tweet</title></head>
<body>
<article data-testid="tweet">
    <div data-testid="User-Name">
        <span>Quoter</span>
        <a href="/quoter/status/100">@quoter</a>
    </div>
    <div data-testid="tweetText" dir="ltr">
        Check out this tweet!
    </div>
    <div data-testid="quoteTweet">
        <div data-testid="User-Name">
            <span>Original Author</span>
        </div>
        <div data-testid="tweetText" dir="ltr">Original tweet content here</div>
    </div>
    <time datetime="2026-01-01T16:00:00Z">4:00 PM · Jan 1, 2026</time>
</article>
</body>
</html>
`
}

// GenerateEmptyTweet creates HTML fixture with no tweet text (error case).
func GenerateEmptyTweet() string {
	return `
<!DOCTYPE html>
<html>
<head><title>Tweet Not Found</title></head>
<body>
<article data-testid="tweet">
    <div>This tweet is unavailable.</div>
</article>
</body>
</html>
`
}
