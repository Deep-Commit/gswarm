# GSwarm Makefile

# Version information
VERSION := $(shell cat VERSION 2>/dev/null || echo "1.0.0")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)"

# Binary name
BINARY_NAME := gswarm

# Build directory
BUILD_DIR := build

# Go files
GO_FILES := $(shell find . -name "*.go" -type f)

.PHONY: all build clean install test test-unit test-integration test-coverage test-bench fmt lint lint-vet lint-staticcheck lint-full version help token-cleanup

# Default target
all: build

# Build the application
build:
	@echo "Building GSwarm version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/gswarm
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all platforms
build-all: clean
	@echo "Building GSwarm for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/gswarm
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/gswarm
	
	# macOS
	@echo "Building for macOS..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/gswarm
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/gswarm
	
	# Windows
	@echo "Building for Windows..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/gswarm
	
	@echo "Build complete for all platforms!"

# Install the application
install: build
	@echo "Installing GSwarm..."
	@rm -f $(shell go env GOPATH)/bin/$(BINARY_NAME)
	@ln -sf $(shell pwd)/$(BUILD_DIR)/$(BINARY_NAME) $(shell go env GOPATH)/bin/$(BINARY_NAME)
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

# Token management targets
token-cleanup:
	@echo "Cleaning up expired tokens..."
	@rm -f ~/.gswarm/tokens.json
	@rm -f /tmp/gswarm.pid
	@echo "Token cleanup complete!"

# Show version information
version:
	@echo "GSwarm version: $(VERSION)"
	@echo "Build date: $(BUILD_DATE)"
	@echo "Git commit: $(GIT_COMMIT)"

# Show help
help:
	@echo "GSwarm Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  build-all    - Build for all platforms (Linux, macOS, Windows)"
	@echo "  install      - Install the application"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run all tests with coverage"
	@echo "  test-unit    - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-bench   - Run benchmarks"
	@echo "  test-short   - Run tests in short mode (skip integration)"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run comprehensive linting (vet + staticcheck + golangci-lint)"
	@echo "  lint-vet     - Run go vet (basic Go toolchain checks)"
	@echo "  lint-staticcheck - Run Staticcheck (advanced static analysis)"
	@echo "  lint-full    - Run full linting suite with extended timeout"
	@echo "  token-cleanup - Clean up expired tokens"
	@echo "  version      - Show version information"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Linting Strategy:"
	@echo "  - go vet: Basic Go toolchain checks (built-in)"
	@echo "  - Staticcheck: Advanced static analysis with 150+ checks"
	@echo "  - golangci-lint: Comprehensive linting with multiple linters"
	@echo "  - Recommended workflow: make lint-vet && make lint-staticcheck"
	@echo ""
	@echo "Testing Strategy:"
	@echo "  - Unit tests: Fast, isolated tests for individual functions"
	@echo "  - Integration tests: End-to-end tests with mocked dependencies"
	@echo "  - Coverage: Enforced minimum coverage with HTML reports"
	@echo "  - Race detection: All tests run with -race flag"
	@echo ""
	@echo "Token Management:"
	@echo "  - Enhanced supervisor with automatic token expiration detection"
	@echo "  - External monitoring script for continuous operation"
	@echo "  - Token manager for advanced token handling"
	@echo ""
	@echo "Version: $(VERSION)"
	@echo "Build date: $(BUILD_DATE)"
	@echo "Git commit: $(GIT_COMMIT)" 