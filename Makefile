# GSwarm Makefile

# Version information
VERSION := $(shell cat VERSION 2>/dev/null || echo "1.0.0")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)"

# Binary names
TELEGRAM_BINARY := gswarm
DISCORD_BINARY := discordd
SERVER_BINARY := gswarm-server

# Build directory
BUILD_DIR := build

# Go files
GO_FILES := $(shell find . -name "*.go" -type f)

.PHONY: all build build-telegram build-discord build-server clean install test test-unit test-integration test-coverage test-bench fmt lint lint-vet lint-staticcheck lint-full version help

# Default target
all: build

# Build both applications
build: build-telegram build-discord

# Build the Telegram bot
build-telegram:
	@echo "Building GSwarm Telegram Bot version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(TELEGRAM_BINARY) ./cmd/gswarm
	@echo "Telegram bot build complete: $(BUILD_DIR)/$(TELEGRAM_BINARY)"

# Build the Discord bot
build-discord:
	@echo "Building GSwarm Discord Bot version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(DISCORD_BINARY) ./cmd/discordd
	@echo "Discord bot build complete: $(BUILD_DIR)/$(DISCORD_BINARY)"

# Build for all platforms
build-all: clean
	@echo "Building GSwarm for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(TELEGRAM_BINARY)-linux-amd64 ./cmd/gswarm
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(DISCORD_BINARY)-linux-amd64 ./cmd/discordd
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(TELEGRAM_BINARY)-linux-arm64 ./cmd/gswarm
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(DISCORD_BINARY)-linux-arm64 ./cmd/discordd
	
	# macOS
	@echo "Building for macOS..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(TELEGRAM_BINARY)-darwin-amd64 ./cmd/gswarm
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(DISCORD_BINARY)-darwin-amd64 ./cmd/discordd
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(TELEGRAM_BINARY)-darwin-arm64 ./cmd/gswarm
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(DISCORD_BINARY)-darwin-arm64 ./cmd/discordd
	
	# Windows
	@echo "Building for Windows..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(TELEGRAM_BINARY)-windows-amd64.exe ./cmd/gswarm
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(DISCORD_BINARY)-windows-amd64.exe ./cmd/discordd
	
	@echo "Build complete for all platforms!"

# Install the applications
install: build
	@echo "Installing GSwarm..."
	@rm -f $(shell go env GOPATH)/bin/$(TELEGRAM_BINARY)
	@rm -f $(shell go env GOPATH)/bin/$(DISCORD_BINARY)
	@rm -f $(shell go env GOPATH)/bin/$(SERVER_BINARY)
	@ln -sf $(shell pwd)/$(BUILD_DIR)/$(TELEGRAM_BINARY) $(shell go env GOPATH)/bin/$(TELEGRAM_BINARY)
	@ln -sf $(shell pwd)/$(BUILD_DIR)/$(DISCORD_BINARY) $(shell go env GOPATH)/bin/$(DISCORD_BINARY)
	@echo "Installation complete!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete!"

# Run all tests
test:
	@echo "Running all tests..."
	@./scripts/run-tests.sh

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@go test -race -v ./internal/...

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	@go test -v ./cmd/gswarm/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@./scripts/run-tests.sh

# Run benchmarks
test-bench:
	@echo "Running benchmarks..."
	@./scripts/run-tests.sh --bench

# Run tests in short mode (skip integration tests)
test-short:
	@echo "Running tests in short mode..."
	@./scripts/run-tests.sh --short

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code with golangci-lint (comprehensive)
lint: lint-vet lint-staticcheck
	@echo "Running comprehensive linting with golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Run go vet (basic Go toolchain checks)
lint-vet:
	@echo "🔍 Running go vet (basic Go toolchain checks)..."
	@go vet ./...
	@echo "✅ go vet completed successfully"

# Run Staticcheck (advanced static analysis)
lint-staticcheck:
	@echo "🔍 Running Staticcheck (advanced static analysis)..."
	@./scripts/staticcheck.sh

# Run full linting suite (vet + staticcheck + golangci-lint)
lint-full: lint-vet lint-staticcheck
	@echo "🔍 Running full linting suite..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running golangci-lint with all linters..."; \
		golangci-lint run --timeout=10m; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Show version information
version:
	@echo "GSwarm version: $(VERSION)"
	@echo "Build date: $(BUILD_DATE)"
	@echo "Git commit: $(GIT_COMMIT)"

# Run the Discord bot (connects to external API)
run-discord:
	@echo "Starting GSwarm Discord Bot (external API)..."
	@echo "Make sure to set these environment variables:"
	@echo "  DISCORD_BOT_TOKEN=your_discord_bot_token"
	@echo "  GSWARM_API_SECRET=your_api_secret"
	@echo "  DISCORD_GUILD_ID=your_guild_id"
	@echo "  DISCORD_ROLE_ID=your_role_id (optional)"
	@echo ""
	@if [ -z "$$DISCORD_BOT_TOKEN" ]; then \
		echo "❌ DISCORD_BOT_TOKEN not set"; \
		exit 1; \
	fi
	@if [ -z "$$GSWARM_API_SECRET" ]; then \
		echo "❌ GSWARM_API_SECRET not set"; \
		exit 1; \
	fi
	@if [ -z "$$DISCORD_GUILD_ID" ]; then \
		echo "❌ DISCORD_GUILD_ID not set"; \
		exit 1; \
	fi
	@echo "✅ All required environment variables set, starting Discord bot..."
	@./build/discordd

# Test the account linking system
test-account-linking:
	@echo "Testing account linking system..."
	@./scripts/test-account-linking.sh

# Show help
help:
	@echo "GSwarm Makefile - Telegram Monitoring Service & Account Linking"
	@echo ""
	@echo "Available targets:"
	@echo "  build        - Build Discord and Telegram bots"
	@echo "  build-telegram - Build the Telegram monitoring service"
	@echo "  build-discord - Build the Discord bot"
	@echo "  build-all    - Build for all platforms (Linux, macOS, Windows)"
	@echo "  install      - Install Discord and Telegram bots"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run all tests with coverage"
	@echo "  test-unit    - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-bench   - Run benchmarks"
	@echo "  test-short   - Run tests in short mode (skip integration)"
	@echo "  test-account-linking - Test the account linking system"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run comprehensive linting (vet + staticcheck + golangci-lint)"
	@echo "  lint-vet     - Run go vet (basic Go toolchain checks)"
	@echo "  lint-staticcheck - Run Staticcheck (advanced static analysis)"
	@echo "  lint-full    - Run full linting suite with extended timeout"
	@echo "  run-discord  - Run the Discord bot (connects to external API)"
	@echo "  version      - Show version information"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Account Linking System:"
	@echo "  - Secure cross-platform Discord-Telegram account linking"
	@echo "  - External API integration (https://gswarm.dev/api)"
	@echo "  - API key protected code issuance (Discord bot only)"
	@echo "  - Public code verification endpoint (Telegram bot)"
	@echo "  - Single-use, time-limited linking codes"
	@echo "  - Automatic duplicate prevention"
	@echo ""
	@echo "Discord Bot Commands:"
	@echo "  /link-telegram     - Generate code to link Discord-Telegram accounts"
	@echo ""
	@echo "Quick Start (Discord Bot):"
	@echo "  1. Set environment variables:"
	@echo "     export DISCORD_BOT_TOKEN=your_token"
	@echo "     export GSWARM_API_SECRET=your_secret"
	@echo "     export DISCORD_GUILD_ID=your_guild_id"
	@echo "  2. Run: make run-discord"
	@echo ""
	@echo "Telegram Monitoring Service:"
	@echo "  - Real-time blockchain monitoring for Gensyn AI"
	@echo "  - Vote and reward tracking with change detection"
	@echo "  - Peer ID monitoring and balance updates"
	@echo "  - Secure local configuration management"
	@echo ""
	@echo "Version: $(VERSION)"
	@echo "Build date: $(BUILD_DATE)"
	@echo "Git commit: $(GIT_COMMIT)" 