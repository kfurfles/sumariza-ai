.PHONY: dev test build docker run clean templ css deps

# Development
dev: templ css
	@go run cmd/server/main.go

# Install dependencies
deps:
	@go mod download
	@go install github.com/a-h/templ/cmd/templ@latest

# Generate Templ templates
templ:
	@templ generate

# Build Tailwind CSS
css:
	@npx tailwindcss -i ./static/css/input.css -o ./static/css/output.css --minify

# Watch CSS changes
css-watch:
	@npx tailwindcss -i ./static/css/input.css -o ./static/css/output.css --watch

# Run tests
test:
	@go test ./... -v

# Run tests with coverage
test-cover:
	@go test ./... -v -cover

# Build binary
build: templ css
	@go build -o bin/sumariza ./cmd/server

# Docker build
docker:
	@docker build -t sumariza-ai .

# Run with Docker
run: docker
	@docker run -p 3000:3000 sumariza-ai

# Clean build artifacts
clean:
	@rm -rf bin/
	@rm -f static/css/output.css
	@find . -name "*_templ.go" -delete

