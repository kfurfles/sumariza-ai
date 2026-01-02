TEST_RESULTS_DIR := test-results
COVERAGE_DIR := coverage

.PHONY: dev test build docker run clean templ css deps

# Development
dev: templ css
	@go run cmd/server/main.go

# Install dependencies
deps:
	@go mod download
	@go install github.com/a-h/templ/cmd/templ@latest
	npm i

deps-server:
	CGO_ENABLED=0
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

test-ci:
	@mkdir -p $(TEST_RESULTS_DIR) $(COVERAGE_DIR)
	@echo "Running unit tests with coverage..."
	@gotestsum --junitfile $(TEST_RESULTS_DIR)/junit-unit.xml \
		--format testname \
		-- \
		-coverprofile=$(COVERAGE_DIR)/coverage-unit.out \
		-covermode=atomic \
		-timeout 2m \
		./...
	@go tool cover -html=$(COVERAGE_DIR)/coverage-unit.out -o $(COVERAGE_DIR)/coverage-unit.html
	@echo "Unit tests complete. Coverage report: $(COVERAGE_DIR)/coverage-unit.html"

# Run tests with coverage
test-cover:
	@go test ./... -v -cover

# Build binary
build: templ css
	@go build -o bin/sumariza ./cmd/server

build-server: deps-server templ css
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/sumariza-linux ./cmd/server

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

