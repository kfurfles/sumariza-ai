TEST_RESULTS_DIR := test-results
COVERAGE_DIR := coverage

# Load deploy config if exists
-include .env.deploy
export

# Default values
SSH_HOST ?= localhost
SSH_USER ?= root
SSH_PORT ?= 22
DEPLOY_PATH ?= /var/www/sumariza-ai
DOMAIN ?= 

.PHONY: dev test build docker run clean templ css deps \
        build-linux deploy deploy-setup deploy-logs deploy-ssh deploy-status

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

# =============================================================================
# DEPLOY TARGETS
# =============================================================================

# Build binary for Linux (cross-compilation)
build-linux: templ css
	@echo "Building for Linux amd64..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/sumariza-linux ./cmd/server
	@echo "Built: bin/sumariza-linux"

# Deploy application (build + upload + restart)
deploy: build-linux
	@echo "Deploying to $(SSH_HOST)..."
	@scp -P $(SSH_PORT) bin/sumariza-linux $(SSH_USER)@$(SSH_HOST):$(DEPLOY_PATH)/bin/sumariza
	@scp -P $(SSH_PORT) -r config/ $(SSH_USER)@$(SSH_HOST):$(DEPLOY_PATH)/
	@scp -P $(SSH_PORT) -r static/ $(SSH_USER)@$(SSH_HOST):$(DEPLOY_PATH)/
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST) "sudo systemctl restart sumariza-ai"
	@echo "Deploy complete!"

# Initial server setup
deploy-setup:
	@echo "Running initial setup on $(SSH_HOST)..."
	@scp -P $(SSH_PORT) -r deploy/ $(SSH_USER)@$(SSH_HOST):/tmp/sumariza-deploy/
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST) "cd /tmp/sumariza-deploy && DOMAIN=$(DOMAIN) DEPLOY_PATH=$(DEPLOY_PATH) bash setup.sh"
	@echo "Setup complete! Now run: make deploy"

# View application logs
deploy-logs:
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST) "journalctl -u sumariza-ai -f"

# SSH into server
deploy-ssh:
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST)

# Check service status
deploy-status:
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST) "systemctl status sumariza-ai"

