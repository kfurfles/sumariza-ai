package web_test

import (
	"testing"

	"sumariza-ai/internal/adapters/web"
	"sumariza-ai/internal/domain"
)

func TestParseTweetURL_ValidTwitterURL_ReturnsUsernameAndID(t *testing.T) {
	// Arrange
	url := "https://twitter.com/elonmusk/status/1234567890123456789"
	expectedUsername := "elonmusk"
	expectedID := "1234567890123456789"

	// Act
	username, id, err := web.ParseTweetURL(url)

	// Assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if username != expectedUsername {
		t.Errorf("username: got %v, want %v", username, expectedUsername)
	}
	if id != expectedID {
		t.Errorf("id: got %v, want %v", id, expectedID)
	}
}

func TestParseTweetURL_ValidXURL_ReturnsUsernameAndID(t *testing.T) {
	// Arrange
	url := "https://x.com/acgfbr/status/2006396789411172607"
	expectedUsername := "acgfbr"
	expectedID := "2006396789411172607"

	// Act
	username, id, err := web.ParseTweetURL(url)

	// Assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if username != expectedUsername {
		t.Errorf("username: got %v, want %v", username, expectedUsername)
	}
	if id != expectedID {
		t.Errorf("id: got %v, want %v", id, expectedID)
	}
}

func TestParseTweetURL_ValidMobileURL_ReturnsUsernameAndID(t *testing.T) {
	// Arrange
	url := "https://mobile.twitter.com/jack/status/20"
	expectedUsername := "jack"
	expectedID := "20"

	// Act
	username, id, err := web.ParseTweetURL(url)

	// Assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if username != expectedUsername {
		t.Errorf("username: got %v, want %v", username, expectedUsername)
	}
	if id != expectedID {
		t.Errorf("id: got %v, want %v", id, expectedID)
	}
}

func TestParseTweetURL_URLWithQueryParams_IgnoresParams(t *testing.T) {
	// Arrange
	testCases := []struct {
		name             string
		url              string
		expectedUsername string
		expectedID       string
	}{
		{
			name:             "with s param",
			url:              "https://twitter.com/user/status/123456?s=20",
			expectedUsername: "user",
			expectedID:       "123456",
		},
		{
			name:             "with multiple params",
			url:              "https://x.com/test/status/789?t=xyz&s=20",
			expectedUsername: "test",
			expectedID:       "789",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			username, id, err := web.ParseTweetURL(tc.url)

			// Assert
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if username != tc.expectedUsername {
				t.Errorf("username: got %v, want %v", username, tc.expectedUsername)
			}
			if id != tc.expectedID {
				t.Errorf("id: got %v, want %v", id, tc.expectedID)
			}
		})
	}
}

func TestParseTweetURL_InvalidURL_ReturnsError(t *testing.T) {
	// Arrange
	testCases := []struct {
		name string
		url  string
	}{
		{name: "google url", url: "https://google.com"},
		{name: "twitter without status", url: "https://twitter.com/user"},
		{name: "twitter profile only", url: "https://twitter.com/user/"},
		{name: "not a url", url: "not a url"},
		{name: "empty string", url: ""},
		{name: "twitter without id", url: "https://twitter.com/user/status/"},
		{name: "non-numeric id", url: "https://twitter.com/user/status/abc"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			_, _, err := web.ParseTweetURL(tc.url)

			// Assert
			if err != domain.ErrInvalidURL {
				t.Errorf("URL %q: expected ErrInvalidURL, got %v", tc.url, err)
			}
		})
	}
}

func TestParseTweetURL_HTTPWithoutS_ReturnsUsernameAndID(t *testing.T) {
	// Arrange
	url := "http://twitter.com/user/status/123"
	expectedUsername := "user"
	expectedID := "123"

	// Act
	username, id, err := web.ParseTweetURL(url)

	// Assert
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if username != expectedUsername {
		t.Errorf("username: got %v, want %v", username, expectedUsername)
	}
	if id != expectedID {
		t.Errorf("id: got %v, want %v", id, expectedID)
	}
}

