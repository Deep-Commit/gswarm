# GSwarm Testing Strategy

This document outlines the comprehensive testing strategy for the GSwarm supervisor application, covering unit tests, integration tests, and end-to-end testing.

## Overview

GSwarm uses a layered testing approach to ensure reliability and maintainability:

- **Unit Tests**: Fast, isolated tests for individual functions and packages
- **Integration Tests**: End-to-end tests with mocked external dependencies
- **End-to-End Tests**: Full system tests (optional, for smoke testing)

## Project Structure

```
gswarm/
├── internal/
│   ├── bootstrap/     # Environment setup functions
│   │   ├── bootstrap.go
│   │   └── bootstrap_test.go
│   ├── config/        # Configuration management
│   │   ├── config.go
│   │   └── config_test.go
│   ├── train/         # Training process management
│   │   ├── train.go
│   │   └── train_test.go
│   └── prompt/        # User interaction functions
│       ├── prompt.go
│       └── prompt_test.go
├── cmd/gswarm/
│   ├── main.go
│   └── main_test.go   # Integration tests
├── testdata/          # Test configuration files
│   ├── quick.yaml
│   └── noop_train.py
├── scripts/
│   └── run-tests.sh   # Test runner script
└── .github/workflows/
    └── test.yml       # CI/CD configuration
```

## Running Tests

### Quick Start

```bash
# Run all tests with coverage
make test

# Run unit tests only
make test-unit

# Run integration tests only
make test-integration

# Run tests in short mode (skip integration tests)
make test-short

# Run benchmarks
make test-bench
```

### Using the Test Runner Script

```bash
# Run all tests
./scripts/run-tests.sh

# Run tests in short mode
./scripts/run-tests.sh --short

# Run tests with benchmarks
./scripts/run-tests.sh --bench
```

### Direct Go Commands

```bash
# Run all tests with race detection
go test -race ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run specific package tests
go test -v ./internal/config/...
go test -v ./internal/bootstrap/...
```

## Test Categories

### 1. Unit Tests

Unit tests focus on individual functions and packages in isolation.

#### Configuration Tests (`internal/config/`)

- **Table-driven tests** for `ValidateConfiguration()`
- **Flag override tests** for `GetConfiguration()`
- **Path generation tests** for `GetConfigPath()`

```go
func TestValidateConfiguration(t *testing.T) {
    cases := []struct {
        name    string
        cfg     Configuration
        wantErr bool
    }{
        {"valid gsm8k", Configuration{ParamB:"0.5", Game:"gsm8k"}, false},
        {"invalid param", Configuration{ParamB:"3", Game:"gsm8k"}, true},
    }
    // ... test implementation
}
```

#### Bootstrap Tests (`internal/bootstrap/`)

- **Mocked command execution** tests
- **Environment setup** validation
- **Error handling** scenarios

```go
func TestEnsureNodeAndNpm_AlreadyInstalled(t *testing.T) {
    mock := &mockCommandRunner{success: true}
    // ... test implementation
}
```

#### Training Tests (`internal/train/`)

- **Error detection** tests for identity conflicts
- **Process cleanup** validation
- **Requirements installation** tests

#### Prompt Tests (`internal/prompt/`)

- **User input validation** tests
- **Default value handling**
- **Input retry logic** tests

### 2. Integration Tests

Integration tests verify the interaction between packages and the overall application flow.

#### Main Application Tests (`cmd/gswarm/`)

- **End-to-end flow** with mocked dependencies
- **Configuration validation** in context
- **Error handling** scenarios
- **Process management** tests

### 3. Mocking Strategy

#### Command Execution Mocking

All external command execution is mocked using package-level variables:

```go
// In bootstrap package
var CommandRunner = exec.Command

// In tests
CommandRunner = func(name string, args ...string) *exec.Cmd {
    return exec.Command("echo", "success")
}
```

#### Prompt Function Mocking

User interaction functions are mocked for testing:

```go
// In config package
var testPromptHFToken = prompt.PromptHFToken

// In tests
testPromptHFToken = func() string { return "test-token" }
```

## Test Data

### Configuration Files

- `testdata/quick.yaml`: Minimal test configuration
- `testdata/noop_train.py`: No-op training script for integration tests

### Test Utilities

- **Mock command runners** for external process simulation
- **Temporary directories** for isolated testing
- **Pipe-based stdin simulation** for user input testing

## Coverage Requirements

- **Minimum coverage**: 80% (enforced in CI)
- **Coverage reports**: Generated as HTML files
- **Coverage upload**: Automatic upload to Codecov

## CI/CD Integration

### GitHub Actions

The `.github/workflows/test.yml` file defines:

- **Matrix testing** across Go versions (1.21, 1.22)
- **Separate jobs** for full tests and quick tests
- **Coverage reporting** to Codecov
- **Linting** with golangci-lint

### Local Development

```bash
# Pre-commit checks
make fmt
make lint
make test-short

# Full test suite
make test
```

## Best Practices

### Writing Tests

1. **Use table-driven tests** for multiple scenarios
2. **Mock external dependencies** consistently
3. **Test error conditions** as well as success cases
4. **Use descriptive test names** that explain the scenario
5. **Clean up resources** in test teardown

### Test Organization

1. **Group related tests** in the same test function
2. **Use subtests** for complex test scenarios
3. **Keep tests focused** on a single responsibility
4. **Use helper functions** for common test setup

### Mocking Guidelines

1. **Mock at package boundaries** (exec.Command, user input)
2. **Use package-level variables** for testability
3. **Restore original functions** in test teardown
4. **Verify mock interactions** when relevant

## Troubleshooting

### Common Issues

1. **Flag conflicts**: Reset `flag.CommandLine` in tests
2. **Race conditions**: Use `-race` flag for detection
3. **Stdin simulation**: Use pipes for user input testing
4. **Temporary files**: Use `t.TempDir()` for cleanup

### Debugging Tests

```bash
# Run tests with verbose output
go test -v ./internal/config/

# Run specific test
go test -run TestValidateConfiguration ./internal/config/

# Run tests with race detection
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Performance Considerations

- **Unit tests**: Should run in < 1 second
- **Integration tests**: Should run in < 10 seconds
- **Full test suite**: Should run in < 30 seconds
- **Use `testing.Short()`** for long-running tests

## Future Enhancements

1. **Property-based testing** for configuration validation
2. **Fuzzing** for input validation
3. **Performance benchmarks** for critical paths
4. **Load testing** for process management
5. **Chaos testing** for error scenarios

## Contributing

When adding new features:

1. **Write tests first** (TDD approach)
2. **Ensure coverage** meets minimum requirements
3. **Update this documentation** if needed
4. **Run full test suite** before submitting PR

For questions about testing, see the [Contributing Guide](CONTRIBUTING.md) or open an issue. 