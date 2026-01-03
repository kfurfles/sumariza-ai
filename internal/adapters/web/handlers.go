package web

import (
	"context"
	"time"

	"sumariza-ai/internal/domain"
	"sumariza-ai/internal/usecases"
	"sumariza-ai/pkg/log"
	"sumariza-ai/templates/components"
	"sumariza-ai/templates/pages"
	"sumariza-ai/templates/partials"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

// Handlers contains the HTTP handlers for the web application.
type Handlers struct {
	getTweet *usecases.GetTweetUseCase
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(getTweet *usecases.GetTweetUseCase) *Handlers {
	return &Handlers{
		getTweet: getTweet,
	}
}

// render is a helper to render templ components.
func render(c *fiber.Ctx, component templ.Component) error {
	c.Set("Content-Type", "text/html")
	return adaptor.HTTPHandler(templ.Handler(component))(c)
}

// Home renders the landing page with URL input.
func (h *Handlers) Home(c *fiber.Ctx) error {
	return render(c, pages.Home())
}

// ViewTweet renders a tweet by username and ID (mirrors Twitter URL structure).
// Shows skeleton immediately, HTMX loads content.
func (h *Handlers) ViewTweet(c *fiber.Ctx) error {
	username := c.Params("username")
	tweetID := c.Params("id")
	return render(c, pages.TweetViewWithSkeleton(username, tweetID))
}

// FetchTweet handles HTMX request to fetch and render a tweet from form input.
func (h *Handlers) FetchTweet(c *fiber.Ctx) error {
	url := c.FormValue("url")

	username, tweetID, err := ParseTweetURL(url)
	if err != nil {
		log.GlobalErrorCtx(c.UserContext(), "invalid tweet URL", "url", url, "error", err)
		return h.renderError(c, domain.ErrInvalidURL)
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), 30*time.Second)
	defer cancel()

	tweet, err := h.getTweet.Execute(ctx, tweetID, username)
	if err != nil {
		log.GlobalErrorCtx(ctx, "fetch tweet failed", "username", username, "tweet_id", tweetID, "error", err)
		return h.renderError(c, err)
	}

	// Set HX-Push-Url header for shareable URL (mirrors Twitter structure)
	c.Set("HX-Push-Url", "/"+username+"/status/"+tweetID)

	return render(c, partials.TweetContent(tweet))
}

// APIGetTweet handles the HTMX request to fetch actual tweet content.
// Used for direct URL access (domain swap).
func (h *Handlers) APIGetTweet(c *fiber.Ctx) error {
	username := c.Params("username")
	tweetID := c.Params("id")

	ctx, cancel := context.WithTimeout(c.UserContext(), 30*time.Second)
	defer cancel()

	tweet, err := h.getTweet.Execute(ctx, tweetID, username)
	if err != nil {
		log.GlobalErrorCtx(ctx, "api get tweet failed", "username", username, "tweet_id", tweetID, "error", err)
		return render(c, components.ErrorMessage(h.friendlyError(err)))
	}

	return render(c, components.TweetCard(tweet))
}

// renderError renders a full-page error.
func (h *Handlers) renderError(c *fiber.Ctx, err error) error {
	c.Status(fiber.StatusNotFound)
	return render(c, pages.Error(h.friendlyError(err)))
}

// friendlyError returns a neutral, non-blaming error message.
func (h *Handlers) friendlyError(err error) string {
	switch err {
	case domain.ErrTweetNotFound:
		return "This tweet couldn't be found. It might be private or no longer available."
	case domain.ErrTweetPrivate:
		return "This tweet isn't available. It might be from a private account."
	case domain.ErrInvalidURL:
		return "That doesn't look like a tweet URL. Try pasting a link from twitter.com or x.com"
	case domain.ErrRateLimited:
		return "Too many requests. Please wait a moment and try again."
	case domain.ErrTextNotFound:
		return "This tweet couldn't be loaded. It might not be publicly available."
	default:
		return "Unable to load this tweet right now. Please try again in a moment."
	}
}
