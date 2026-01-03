package web

import (
	"bytes"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"sumariza-ai/pkg/log"
	"sumariza-ai/pkg/log/transporters"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func setupTestApp() *fiber.App {
	app := fiber.New()
	app.Use(requestid.New(RequestIDConfig()))
	app.Use(RequestIDToContextMiddleware())
	return app
}

func TestRequestIDToContext_ExtractsIDFromFiber(t *testing.T) {
	app := setupTestApp()

	var capturedRequestID string
	app.Get("/test", func(c *fiber.Ctx) error {
		capturedRequestID = log.RequestIDFromContext(c.UserContext())
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if capturedRequestID == "" {
		t.Error("request_id should be extracted from Fiber's requestid middleware")
	}

	// Should also be in response header (set by Fiber's requestid)
	headerID := resp.Header.Get("X-Request-ID")
	if headerID == "" {
		t.Error("X-Request-ID header should be set in response")
	}

	if headerID != capturedRequestID {
		t.Errorf("response header = %q, context = %q, should match", headerID, capturedRequestID)
	}
}

func TestRequestIDToContext_UsesProvidedID(t *testing.T) {
	app := setupTestApp()

	var capturedRequestID string
	app.Get("/test", func(c *fiber.Ctx) error {
		capturedRequestID = log.RequestIDFromContext(c.UserContext())
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "custom-trace-id-123")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()

	if capturedRequestID != "custom-trace-id-123" {
		t.Errorf("request_id = %q, want %q", capturedRequestID, "custom-trace-id-123")
	}
}

func TestRequestLoggerMiddleware_LogsRequest(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := log.New(log.Info, transporters.NewStdoutWithWriter(&buf))
	log.SetDefault(logger)
	defer logger.Close()

	app := fiber.New()
	app.Use(requestid.New(RequestIDConfig()))
	app.Use(RequestIDToContextMiddleware())
	app.Use(RequestLoggerMiddleware())

	app.Get("/test-path", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test-path", nil)
	req.Header.Set("X-Request-ID", "test-req-123")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// Wait for async log
	logger.Close()

	output := buf.String()

	// Verify log contains expected fields
	if !strings.Contains(output, "request completed") {
		t.Errorf("log should contain 'request completed', got: %s", output)
	}
	if !strings.Contains(output, "test-req-123") {
		t.Errorf("log should contain request_id 'test-req-123', got: %s", output)
	}
	if !strings.Contains(output, "/test-path") {
		t.Errorf("log should contain path '/test-path', got: %s", output)
	}
	if !strings.Contains(output, `"status":200`) || !strings.Contains(output, `"status": 200`) {
		// JSON might have spaces or not
		if !strings.Contains(output, "200") {
			t.Errorf("log should contain status 200, got: %s", output)
		}
	}
}

func TestRequestLoggerMiddleware_LogsErrorStatus(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(log.Info, transporters.NewStdoutWithWriter(&buf))
	log.SetDefault(logger)
	defer logger.Close()

	app := fiber.New()
	app.Use(requestid.New(RequestIDConfig()))
	app.Use(RequestIDToContextMiddleware())
	app.Use(RequestLoggerMiddleware())

	app.Get("/not-found", func(c *fiber.Ctx) error {
		return c.Status(404).SendString("not found")
	})

	req := httptest.NewRequest("GET", "/not-found", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	logger.Close()

	output := buf.String()

	// 4xx should be logged as WARN
	if !strings.Contains(output, "WARN") {
		t.Errorf("4xx status should be logged as WARN, got: %s", output)
	}
}

func TestRequestLoggerMiddleware_Logs500AsError(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(log.Info, transporters.NewStdoutWithWriter(&buf))
	log.SetDefault(logger)
	defer logger.Close()

	app := fiber.New()
	app.Use(requestid.New(RequestIDConfig()))
	app.Use(RequestIDToContextMiddleware())
	app.Use(RequestLoggerMiddleware())

	app.Get("/error", func(c *fiber.Ctx) error {
		return c.Status(500).SendString("internal error")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	logger.Close()

	output := buf.String()

	// 5xx should be logged as ERROR
	if !strings.Contains(output, "ERROR") {
		t.Errorf("5xx status should be logged as ERROR, got: %s", output)
	}
}
