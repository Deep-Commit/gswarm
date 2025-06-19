#!/bin/bash

# Pre-commit hook for GSwarm
# This script runs linting checks before allowing commits

set -e

echo "🔍 Running pre-commit linting checks..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "success")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "error")
            echo -e "${RED}❌ $message${NC}"
            ;;
        "warning")
            echo -e "${YELLOW}⚠️  $message${NC}"
            ;;
    esac
}

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_status "error" "Not in a git repository"
    exit 1
fi

# Get list of staged Go files
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [ -z "$STAGED_GO_FILES" ]; then
    print_status "success" "No Go files staged, skipping linting"
    exit 0
fi

echo "📁 Staged Go files:"
echo "$STAGED_GO_FILES" | sed 's/^/  - /'

# Run go vet (fast, basic checks)
echo ""
echo "🔍 Running go vet..."
if make lint-vet > /dev/null 2>&1; then
    print_status "success" "go vet passed"
else
    print_status "error" "go vet failed"
    echo "Run 'make lint-vet' to see details"
    exit 1
fi

# Run Staticcheck (comprehensive analysis)
echo ""
echo "🔍 Running Staticcheck..."
if make lint-staticcheck > /dev/null 2>&1; then
    print_status "success" "Staticcheck passed"
else
    print_status "warning" "Staticcheck found issues"
    echo "Run 'make lint-staticcheck' to see details"
    echo "You can still commit, but consider fixing these issues"
fi

# Optional: Run full linting (slower, but comprehensive)
if [ "$1" = "--full" ]; then
    echo ""
    echo "🔍 Running full linting suite..."
    if make lint > /dev/null 2>&1; then
        print_status "success" "Full linting passed"
    else
        print_status "warning" "Full linting found issues"
        echo "Run 'make lint' to see details"
    fi
fi

print_status "success" "Pre-commit checks completed"
echo ""
echo "💡 Tips:"
echo "  - Run 'make lint-vet' for quick feedback"
echo "  - Run 'make lint-staticcheck' for comprehensive analysis"
echo "  - Run 'make lint' for full linting suite"
echo "  - Use '--full' flag for complete pre-commit check" 