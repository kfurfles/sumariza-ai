package main

import (
	"os"
	"strconv"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/joho/godotenv"

	"sumariza-ai/internal/adapters/cache"
	"sumariza-ai/internal/adapters/scraper"
	"sumariza-ai/internal/adapters/web"
	"sumariza-ai/internal/usecases"
	"sumariza-ai/pkg/log"
	"sumariza-ai/pkg/log/transporters"
)

func main() {
	// Initialize logger
	appLogger := log.New(log.Info, transporters.NewStdout())
	log.SetDefault(appLogger)
	defer appLogger.Close()

	// Load .env file if it exists (development only, ignored in production)
	_ = godotenv.Load()

	// Load selector configuration
	selectors, err := scraper.LoadSelectors("config/selectors.yaml")
	if err != nil {
		log.GlobalFatal("failed to load selectors", "error", err)
		os.Exit(1)
	}

	// Initialize browser pool (single persistent browser)
	var options []chromedp.ExecAllocatorOption
	if !getIsLocalEnv() {
		options = append(options, chromedp.Flag("single-process", true))
	}
	browserPool, err := scraper.NewBrowserPool(options)
	if err != nil {
		log.GlobalFatal("failed to initialize browser", "error", err)
		os.Exit(1)
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

	// Middleware (order matters!)
	app.Use(recover.New())                        // 1. Panic recovery
	app.Use(requestid.New(web.RequestIDConfig())) // 2. Generate/extract request ID (Fiber managed)
	app.Use(web.RequestIDToContextMiddleware())   // 3. Bridge request ID to pkg/log context
	app.Use(web.RequestLoggerMiddleware())        // 4. Structured JSON request logging

	// Setup routes
	web.SetupRoutes(app, handlers, rateLimiter)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.GlobalInfo("starting server", "port", port)
	if err := app.Listen(":" + port); err != nil {
		log.GlobalFatal("server failed", "error", err)
		os.Exit(1)
	}
}

// getCacheTTL returns the cache TTL from environment variable or default.
func getCacheTTL() time.Duration {
	ttlMinutes := os.Getenv("CACHE_TTL_MINUTES")
	if ttlMinutes == "" {
		return 5 * time.Minute
	}

	minutes, err := strconv.Atoi(ttlMinutes)
	if err != nil {
		log.GlobalWarn("invalid CACHE_TTL_MINUTES, using default", "value", ttlMinutes)
		return 5 * time.Minute
	}

	return time.Duration(minutes) * time.Minute
}

func getIsLocalEnv() bool {
	value := os.Getenv("IS_LOCAL")
	if value == "1" {
		log.GlobalInfo("environment detected", "env", "LOCAL")
		return true
	}
	log.GlobalInfo("environment detected", "env", "PRODUCTION")
	return false
}
