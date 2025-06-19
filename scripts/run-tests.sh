#!/bin/bash

# Test runner script for GSwarm
# This script runs all tests with proper flags and coverage reporting

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Running GSwarm tests...${NC}"

# Function to print test results
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2 passed${NC}"
    else
        echo -e "${RED}✗ $2 failed${NC}"
        exit 1
    fi
}

# Run unit tests with race detection
echo -e "${YELLOW}Running unit tests...${NC}"
go test -race -v ./internal/...
print_result $? "Unit tests"

# Run integration tests (if not in short mode)
if [ "$1" != "--short" ]; then
    echo -e "${YELLOW}Running integration tests...${NC}"
    go test -v ./cmd/gswarm/...
    print_result $? "Integration tests"
else
    echo -e "${YELLOW}Skipping integration tests (short mode)${NC}"
fi

# Run tests with coverage
echo -e "${YELLOW}Running tests with coverage...${NC}"
go test -race -coverprofile=coverage.out ./...
print_result $? "Coverage tests"

# Generate coverage report
if command -v go tool cover >/dev/null 2>&1; then
    echo -e "${YELLOW}Generating coverage report...${NC}"
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}Coverage report generated: coverage.html${NC}"
    
    # Show coverage summary
    echo -e "${YELLOW}Coverage summary:${NC}"
    go tool cover -func=coverage.out | tail -1
else
    echo -e "${YELLOW}go tool cover not available, skipping coverage report${NC}"
fi

# Run benchmarks (if requested)
if [ "$1" = "--bench" ]; then
    echo -e "${YELLOW}Running benchmarks...${NC}"
    go test -bench=. -benchmem ./...
    print_result $? "Benchmarks"
fi

# Clean up coverage file
rm -f coverage.out

echo -e "${GREEN}All tests completed successfully!${NC}" 