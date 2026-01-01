package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"sumariza-ai/internal/adapters/cache"
	"sumariza-ai/internal/adapters/scraper"
	"sumariza-ai/internal/adapters/web"
	"sumariza-ai/internal/usecases"
)

func main() {
	// Load selector configuration
	selectors, err := scraper.LoadSelectors("config/selectors.yaml")
	if err != nil {
		log.Fatalf("Failed to load selectors: %v", err)
	}

	// Initialize browser pool (single persistent browser)
	browserPool, err := scraper.NewBrowserPool()
	if err != nil {
		log.Fatalf("Failed to initialize browser: %v", err)
	}
	defer browserPool.Close()

	// Get cache TTL from environment (default 5 minutes)
	cacheTTL := getCacheTTL()

	// Initialize adapters
	tweetScraper := scraper.NewTwitterScraper(browserPool, selectors)
	tweetCache := cache.NewMemoryCache(cacheTTL)

	// Initialize use cases
	scrapeUC := usecases.NewScrapeTweetUseCase(tweetScraper)
	getTweetUC := usecases.NewGetTweetUseCase(tweetCache, scrapeUC)

	// Initialize web handlers
	handlers := web.NewHandlers(getTweetUC)
	rateLimiter := web.NewRateLimiter(10, time.Minute) // 10 scrapes/min

	// Setup Fiber
	app := fiber.New(fiber.Config{
		AppName: "Sumariza AI",
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// Setup routes
	web.SetupRoutes(app, handlers, rateLimiter)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Starting Sumariza AI on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

// getCacheTTL returns the cache TTL from environment variable or default.
func getCacheTTL() time.Duration {
	ttlMinutes := os.Getenv("CACHE_TTL_MINUTES")
	if ttlMinutes == "" {
		return 5 * time.Minute
	}

	minutes, err := strconv.Atoi(ttlMinutes)
	if err != nil {
		log.Printf("Invalid CACHE_TTL_MINUTES value, using default 5 minutes")
		return 5 * time.Minute
	}

	return time.Duration(minutes) * time.Minute
}

