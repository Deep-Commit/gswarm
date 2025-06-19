#!/bin/bash

# Staticcheck runner script
# This script runs Staticcheck independently for focused analysis

set -e

echo "🔍 Running Staticcheck analysis..."

# Check if staticcheck is installed
if ! command -v staticcheck >/dev/null 2>&1; then
    echo "❌ Staticcheck not found. Installing..."
    go install honnef.co/go/tools/cmd/staticcheck@latest
fi

# Run staticcheck with all checks enabled
echo "📊 Running Staticcheck with all checks..."
staticcheck -checks=all ./...

echo "✅ Staticcheck analysis complete!"

# Optional: Run with specific checks for different purposes
echo ""
echo "🔧 Additional Staticcheck commands you can run:"
echo "  staticcheck -checks=SA ./...     # Style and correctness checks"
echo "  staticcheck -checks=ST ./...     # Style checks only"
echo "  staticcheck -checks=S ./...      # All checks except performance"
echo "  staticcheck -checks=SA1000 ./... # Specific check (e.g., SA1000 for time.Sleep)" 