package web

import (
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures the application routes.
func SetupRoutes(app *fiber.App, handlers *Handlers, rateLimiter *RateLimiter) {
	// Static assets
	app.Static("/static", "./static")

	// Home page
	app.Get("/", handlers.Home)

	// Tweet view - mirrors Twitter URL structure
	// Example: /acgfbr/status/2006396789411172607
	app.Get("/:username/status/:id", handlers.ViewTweet)

	// HTMX endpoint for fetching tweets from form input
	app.Post("/fetch", handlers.FetchTweet)

	// API endpoint for HTMX to fetch tweet content (direct URL access)
	app.Get("/api/tweet/:username/:id", handlers.APIGetTweet)
}

