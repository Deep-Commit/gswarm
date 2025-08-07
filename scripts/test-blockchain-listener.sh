#!/bin/bash

# Test script for Blockchain Listener Service
# This script helps test the blockchain listener functionality

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_ROOT/build"
BINARY_NAME="blockchain-listener"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if binary exists
check_binary() {
    if [ ! -f "$BUILD_DIR/$BINARY_NAME" ]; then
        log_error "Binary not found: $BUILD_DIR/$BINARY_NAME"
        log_info "Building blockchain listener..."
        cd "$PROJECT_ROOT"
        make build-blockchain-listener
        if [ ! -f "$BUILD_DIR/$BINARY_NAME" ]; then
            log_error "Failed to build binary"
            exit 1
        fi
    fi
    log_success "Binary found: $BUILD_DIR/$BINARY_NAME"
}

# Test version command
test_version() {
    log_info "Testing version command..."
    if "$BUILD_DIR/$BINARY_NAME" --version; then
        log_success "Version command works"
    else
        log_error "Version command failed"
        return 1
    fi
}

# Test help command
test_help() {
    log_info "Testing help command..."
    if "$BUILD_DIR/$BINARY_NAME" --help; then
        log_success "Help command works"
    else
        log_error "Help command failed"
        return 1
    fi
}

# Test configuration validation
test_config_validation() {
    log_info "Testing configuration validation..."
    
    # Test with missing RPC URL
    if "$BUILD_DIR/$BINARY_NAME" --contract-address "0x1234567890123456789012345678901234567890" --postgres-dsn "postgres://test" 2>&1 | grep -q "RPC URL is required"; then
        log_success "RPC URL validation works"
    else
        log_error "RPC URL validation failed"
        return 1
    fi
    
    # Test with missing contract address
    if "$BUILD_DIR/$BINARY_NAME" --rpc-url "wss://test" --postgres-dsn "postgres://test" 2>&1 | grep -q "Contract address is required"; then
        log_success "Contract address validation works"
    else
        log_error "Contract address validation failed"
        return 1
    fi
    
    # Test with missing PostgreSQL DSN
    if "$BUILD_DIR/$BINARY_NAME" --rpc-url "wss://test" --contract-address "0x1234567890123456789012345678901234567890" 2>&1 | grep -q "PostgreSQL DSN is required"; then
        log_success "PostgreSQL DSN validation works"
    else
        log_error "PostgreSQL DSN validation failed"
        return 1
    fi
}

# Test invalid contract address
test_invalid_contract() {
    log_info "Testing invalid contract address..."
    if "$BUILD_DIR/$BINARY_NAME" --rpc-url "wss://test" --contract-address "invalid-address" --postgres-dsn "postgres://test" 2>&1 | grep -q "invalid contract address"; then
        log_success "Invalid contract address validation works"
    else
        log_error "Invalid contract address validation failed"
        return 1
    fi
}

# Test environment variables
test_env_vars() {
    log_info "Testing environment variable support..."
    
    # Test RPC_URL environment variable
    export RPC_URL="wss://test-rpc"
    if "$BUILD_DIR/$BINARY_NAME" --contract-address "0x1234567890123456789012345678901234567890" --postgres-dsn "postgres://test" 2>&1 | grep -q "RPC URL is required"; then
        log_error "RPC_URL environment variable not working"
        return 1
    fi
    unset RPC_URL
    
    # Test CONTRACT_ADDRESS environment variable
    export CONTRACT_ADDRESS="0x1234567890123456789012345678901234567890"
    if "$BUILD_DIR/$BINARY_NAME" --rpc-url "wss://test" --postgres-dsn "postgres://test" 2>&1 | grep -q "Contract address is required"; then
        log_error "CONTRACT_ADDRESS environment variable not working"
        return 1
    fi
    unset CONTRACT_ADDRESS
    
    # Test POSTGRES_DSN environment variable
    export POSTGRES_DSN="postgres://test"
    if "$BUILD_DIR/$BINARY_NAME" --rpc-url "wss://test" --contract-address "0x1234567890123456789012345678901234567890" 2>&1 | grep -q "PostgreSQL DSN is required"; then
        log_error "POSTGRES_DSN environment variable not working"
        return 1
    fi
    unset POSTGRES_DSN
    
    log_success "Environment variable support works"
}

# Main test function
run_tests() {
    log_info "Starting blockchain listener tests..."
    
    check_binary
    test_version
    test_help
    test_config_validation
    test_invalid_contract
    test_env_vars
    
    log_success "All tests passed!"
}

# Show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  --version      Test version command only"
    echo "  --help         Test help command only"
    echo "  --config       Test configuration validation only"
    echo "  --env          Test environment variables only"
    echo ""
    echo "Examples:"
    echo "  $0              Run all tests"
    echo "  $0 --version    Test version command"
    echo "  $0 --config     Test configuration validation"
}

# Parse command line arguments
case "${1:-}" in
    -h|--help)
        show_usage
        exit 0
        ;;
    --version)
        check_binary
        test_version
        exit 0
        ;;
    --help)
        check_binary
        test_help
        exit 0
        ;;
    --config)
        check_binary
        test_config_validation
        test_invalid_contract
        exit 0
        ;;
    --env)
        check_binary
        test_env_vars
        exit 0
        ;;
    "")
        run_tests
        exit 0
        ;;
    *)
        log_error "Unknown option: $1"
        show_usage
        exit 1
        ;;
esac 