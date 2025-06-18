#!/bin/bash

# GSwarm Version Bump Script
# Usage: ./scripts/bump-version.sh [major|minor|patch]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "Not in a git repository. Please run this script from the project root."
    exit 1
fi

# Check if there are uncommitted changes
if ! git diff-index --quiet HEAD --; then
    print_warning "You have uncommitted changes. Please commit or stash them before bumping version."
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Get current version
CURRENT_VERSION=$(cat VERSION 2>/dev/null || echo "0.0.0")
print_status "Current version: $CURRENT_VERSION"

# Parse version components
IFS='.' read -ra VERSION_PARTS <<< "$CURRENT_VERSION"
MAJOR=${VERSION_PARTS[0]}
MINOR=${VERSION_PARTS[1]}
PATCH=${VERSION_PARTS[2]}

# Determine bump type
BUMP_TYPE=${1:-patch}

case $BUMP_TYPE in
    major)
        NEW_MAJOR=$((MAJOR + 1))
        NEW_MINOR=0
        NEW_PATCH=0
        print_status "Bumping major version: $CURRENT_VERSION -> $NEW_MAJOR.$NEW_MINOR.$NEW_PATCH"
        ;;
    minor)
        NEW_MAJOR=$MAJOR
        NEW_MINOR=$((MINOR + 1))
        NEW_PATCH=0
        print_status "Bumping minor version: $CURRENT_VERSION -> $NEW_MAJOR.$NEW_MINOR.$NEW_PATCH"
        ;;
    patch)
        NEW_MAJOR=$MAJOR
        NEW_MINOR=$MINOR
        NEW_PATCH=$((PATCH + 1))
        print_status "Bumping patch version: $CURRENT_VERSION -> $NEW_MAJOR.$NEW_MINOR.$NEW_PATCH"
        ;;
    *)
        print_error "Invalid bump type: $BUMP_TYPE"
        print_error "Usage: $0 [major|minor|patch]"
        exit 1
        ;;
esac

NEW_VERSION="$NEW_MAJOR.$NEW_MINOR.$NEW_PATCH"

# Update VERSION file
echo "$NEW_VERSION" > VERSION
print_status "Updated VERSION file: $NEW_VERSION"

# Update main.go version variable
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS version
    sed -i '' "s/Version   = \"[^\"]*\"/Version   = \"$NEW_VERSION\"/" cmd/gswarm/main.go
else
    # Linux version
    sed -i "s/Version   = \"[^\"]*\"/Version   = \"$NEW_VERSION\"/" cmd/gswarm/main.go
fi
print_status "Updated main.go version variable"

# Build the application to test
print_status "Building application with new version..."
make clean > /dev/null 2>&1 || true
make build > /dev/null 2>&1

# Test version output
VERSION_OUTPUT=$(./build/gswarm -version 2>/dev/null || echo "Build failed")
if echo "$VERSION_OUTPUT" | grep -q "$NEW_VERSION"; then
    print_status "Version bump successful! New version: $NEW_VERSION"
else
    print_error "Version bump failed. Output: $VERSION_OUTPUT"
    exit 1
fi

# Create git tag
TAG_NAME="v$NEW_VERSION"
print_status "Creating git tag: $TAG_NAME"

# Add changes
git add VERSION cmd/gswarm/main.go
git commit -m "Bump version to $NEW_VERSION" > /dev/null 2>&1 || print_warning "No changes to commit"

# Create tag
git tag -a "$TAG_NAME" -m "Release version $NEW_VERSION" > /dev/null 2>&1 || print_warning "Tag already exists"

print_status "Version bump complete!"
print_status "New version: $NEW_VERSION"
print_status "Tag: $TAG_NAME"
print_status ""
print_status "Next steps:"
print_status "1. Review changes: git log --oneline -5"
print_status "2. Push changes: git push && git push --tags"
print_status "3. Create release on GitHub" 