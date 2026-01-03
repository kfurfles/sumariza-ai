TEST_RESULTS_DIR := test-results
COVERAGE_DIR := coverage

# Load deploy config if exists
# Create .env.deploy with:
#   SSH_HOST=your-server.com
#   SSH_USER=root
#   SSH_PORT=22
#   DEPLOY_PATH=/var/www/sumariza-ai
#   APP_PORT=3000
#   CACHE_TTL_MINUTES=5
#   CHROME_PATH=/usr/bin/chromium-browser
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
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/sumariza ./cmd/server

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
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/sumariza ./cmd/server
	@echo "Built: bin/sumariza"

# Deploy application (build + upload + restart)
# Mirrors the GitHub Actions deploy workflow - uploads to /tmp then moves with sudo
deploy: build-linux
	@echo "Deploying to $(SSH_HOST)..."
	@echo "Preparing staging area..."
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST) "rm -rf /tmp/sumariza-deploy && mkdir -p /tmp/sumariza-deploy"
	@echo "Uploading files..."
	@scp -P $(SSH_PORT) bin/sumariza $(SSH_USER)@$(SSH_HOST):/tmp/sumariza-deploy/
	@scp -P $(SSH_PORT) -r config $(SSH_USER)@$(SSH_HOST):/tmp/sumariza-deploy/
	@scp -P $(SSH_PORT) -r static $(SSH_USER)@$(SSH_HOST):/tmp/sumariza-deploy/
	@scp -P $(SSH_PORT) deploy/sumariza-ai.service $(SSH_USER)@$(SSH_HOST):/tmp/sumariza-deploy/
	@echo "Installing files and restarting service..."
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST) "\
		sudo mkdir -p $(DEPLOY_PATH)/bin && \
		sudo mv /tmp/sumariza-deploy/sumariza $(DEPLOY_PATH)/bin/sumariza && \
		sudo chmod +x $(DEPLOY_PATH)/bin/sumariza && \
		sudo rm -rf $(DEPLOY_PATH)/config && \
		sudo mv /tmp/sumariza-deploy/config $(DEPLOY_PATH)/config && \
		sudo rm -rf $(DEPLOY_PATH)/static && \
		sudo mv /tmp/sumariza-deploy/static $(DEPLOY_PATH)/static && \
		sudo cp /tmp/sumariza-deploy/sumariza-ai.service /etc/systemd/system/sumariza-ai.service && \
		sudo chown -R www-data:www-data $(DEPLOY_PATH) && \
		sudo systemctl daemon-reload && \
		sudo systemctl enable sumariza-ai && \
		sudo systemctl restart sumariza-ai && \
		rm -rf /tmp/sumariza-deploy && \
		sleep 3 && \
		sudo systemctl is-active sumariza-ai"
	@echo "Deploy complete!"

# Initial server setup (creates .env and base structure)
deploy-setup:
	@echo "Running initial setup on $(SSH_HOST)..."
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST) "\
		mkdir -p $(DEPLOY_PATH)/bin && \
		id -u www-data &>/dev/null || sudo useradd -r -s /bin/false www-data && \
		sudo chown -R www-data:www-data $(DEPLOY_PATH)"
	@echo "Setup complete!"
	@echo ""
	@echo "IMPORTANT: Create .env file on server:"
	@echo "  ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST)"
	@echo "  sudo nano $(DEPLOY_PATH)/.env"
	@echo ""
	@echo "Add these variables:"
	@echo "  PORT=3000"
	@echo "  CACHE_TTL_MINUTES=5"
	@echo "  CHROME_PATH=/usr/bin/chromium-browser"
	@echo ""
	@echo "Then run: make deploy"

# Update .env file on server
deploy-env:
ifndef APP_PORT
	$(error APP_PORT is not set. Usage: make deploy-env APP_PORT=3000 CACHE_TTL_MINUTES=5 CHROME_PATH=/usr/bin/chromium-browser)
endif
	@echo "Updating .env on $(SSH_HOST)..."
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST) "\
		echo 'PORT=$(APP_PORT)' | sudo tee $(DEPLOY_PATH)/.env > /dev/null && \
		echo 'CACHE_TTL_MINUTES=$(CACHE_TTL_MINUTES)' | sudo tee -a $(DEPLOY_PATH)/.env > /dev/null && \
		echo 'CHROME_PATH=$(CHROME_PATH)' | sudo tee -a $(DEPLOY_PATH)/.env > /dev/null && \
		sudo chmod 600 $(DEPLOY_PATH)/.env && \
		sudo chown www-data:www-data $(DEPLOY_PATH)/.env"
	@echo ".env updated!"

# View application logs
deploy-logs:
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST) "sudo journalctl -u sumariza-ai -f"

# SSH into server
deploy-ssh:
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST)

# Check service status
deploy-status:
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST) "sudo systemctl status sumariza-ai"

# Quick restart without redeploy
deploy-restart:
	@ssh -p $(SSH_PORT) $(SSH_USER)@$(SSH_HOST) "sudo systemctl restart sumariza-ai && sleep 2 && sudo systemctl is-active sumariza-ai"

