# Linting Strategy for GSwarm

This document outlines our comprehensive linting strategy, with a focus on **Staticcheck** as the most impactful tool for Go code quality.

## Overview

Our linting strategy follows a layered approach:

1. **go vet** - Basic Go toolchain checks (built-in)
2. **Staticcheck** - Advanced static analysis with 150+ checks
3. **golangci-lint** - Comprehensive linting with multiple linters

## Staticcheck: The Most Impactful Tool

Staticcheck is our primary static analysis tool because it:

- **Subsumes go vet** - Includes all go vet checks plus much more
- **Catches dead code** - Identifies unused variables, functions, and imports
- **Finds performance bugs** - Detects inefficient patterns and unnecessary allocations
- **Catches correctness bugs** - Identifies logic errors, race conditions, and API misuse
- **Suggests simplifications** - Recommends cleaner, more idiomatic code
- **Covers a huge surface area** - 150+ different checks in one tool

### Staticcheck Check Categories

Staticcheck organizes its checks into categories:

- **S** - Style checks (naming, formatting, etc.)
- **SA** - Static analysis (correctness, performance, etc.)
- **ST** - Style checks (subset of S)
- **SA1000** - Specific checks (e.g., SA1000 for time.Sleep usage)

## Quick Start

### Install Tools

```bash
# Install Staticcheck
go install honnef.co/go/tools/cmd/staticcheck@latest

# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Run Linting

```bash
# Run go vet (basic checks)
make lint-vet

# Run Staticcheck (advanced analysis)
make lint-staticcheck

# Run comprehensive linting (recommended)
make lint

# Run full linting suite with extended timeout
make lint-full
```

## Individual Tool Usage

### go vet

Basic Go toolchain checks:

```bash
go vet ./...
```

**What it catches:**
- Bad Printf verbs
- Unreachable code
- Incorrect struct tags
- Basic API misuse

### Staticcheck

Advanced static analysis:

```bash
# Run all checks
staticcheck -checks=all ./...

# Run style and correctness checks only
staticcheck -checks=SA ./...

# Run style checks only
staticcheck -checks=ST ./...

# Run all checks except performance
staticcheck -checks=S ./...

# Run specific check
staticcheck -checks=SA1000 ./...
```

**What it catches (examples):**
- Unused variables and functions
- Inefficient string concatenation
- Unnecessary type conversions
- Potential nil pointer dereferences
- Race conditions
- API misuse patterns
- Dead code elimination opportunities

### golangci-lint

Comprehensive linting with multiple linters:

```bash
golangci-lint run
```

**What it provides:**
- Runs multiple linters in parallel
- Configurable via `.golangci.yml`
- Includes Staticcheck, errcheck, revive, and many others
- Customizable rules and exclusions

## Configuration

### Staticcheck Configuration

Staticcheck is configured in `.golangci.yml`:

```yaml
linters-settings:
  staticcheck:
    go: "1.21"
    checks: ["all"]
```

### golangci-lint Configuration

The `.golangci.yml` file configures:

- **Enabled linters** - Which linters to run
- **Linter settings** - Specific configuration for each linter
- **Exclusions** - Rules to ignore for specific files/patterns
- **Performance settings** - Timeouts and parallel execution

## Common Issues and Solutions

### Error Handling

**Problem:** Unchecked error returns
```go
input, _ := reader.ReadString('\n')  // ❌ Error ignored
```

**Solution:** Always check errors
```go
input, err := reader.ReadString('\n')
if err != nil {
    return fmt.Errorf("failed to read input: %w", err)
}
```

### Unused Parameters

**Problem:** Unused function parameters
```go
func testFunction(prompt string, defaultValue string) bool {  // ❌ 'prompt' unused
    return false
}
```

**Solution:** Use underscore for unused parameters
```go
func testFunction(_ string, defaultValue string) bool {  // ✅ Clear intent
    return false
}
```

### Error Wrapping

**Problem:** Non-wrapping error formatting
```go
return fmt.Errorf("failed to install: %v", err)  // ❌ Loses error context
```

**Solution:** Use error wrapping
```go
return fmt.Errorf("failed to install: %w", err)  // ✅ Preserves error chain
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Lint

on: [push, pull_request]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    
    - name: Install Staticcheck
      run: go install honnef.co/go/tools/cmd/staticcheck@latest
    
    - name: Run go vet
      run: go vet ./...
    
    - name: Run Staticcheck
      run: staticcheck -checks=all ./...
    
    - name: Run golangci-lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest
```

## Best Practices

### 1. Run Linting Early and Often

```bash
# In your development workflow
make lint-vet      # Quick feedback
make lint-staticcheck  # Comprehensive analysis
```

### 2. Fix Issues Incrementally

- Start with `go vet` issues (usually easy to fix)
- Address Staticcheck issues systematically
- Use `golangci-lint` for comprehensive coverage

### 3. Configure Exclusions Carefully

Only exclude rules when:
- The code is intentionally written that way
- It's a false positive that can't be avoided
- The rule doesn't apply to your use case

### 4. Use Linting in Pre-commit Hooks

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Running pre-commit linting..."
make lint-vet
make lint-staticcheck

if [ $? -ne 0 ]; then
    echo "Linting failed. Please fix issues before committing."
    exit 1
fi
```

## Performance Considerations

### Staticcheck Performance

- Staticcheck is fast and can analyze large codebases quickly
- Use `-checks=SA` for faster feedback during development
- Use `-checks=all` for comprehensive analysis in CI

### golangci-lint Performance

- Configure appropriate timeouts in `.golangci.yml`
- Use parallel execution for faster results
- Consider running only essential linters in pre-commit hooks

## Troubleshooting

### Common Issues

1. **Staticcheck not found**
   ```bash
   go install honnef.co/go/tools/cmd/staticcheck@latest
   ```

2. **golangci-lint timeout**
   - Increase timeout in `.golangci.yml`
   - Run with `--timeout=10m` flag

3. **False positives**
   - Use `//nolint` comments sparingly
   - Configure exclusions in `.golangci.yml`

### Getting Help

- [Staticcheck Documentation](https://staticcheck.io/docs/)
- [golangci-lint Documentation](https://golangci-lint.run/)
- [Go vet Documentation](https://pkg.go.dev/cmd/vet)

## Conclusion

Staticcheck is indeed the single most impactful tool for Go code quality. Combined with `go vet` and `golangci-lint`, it provides comprehensive coverage of potential issues while maintaining excellent performance.

**Recommended workflow:**
1. Always run `go vet ./...` as part of your build
2. Use Staticcheck for advanced analysis and catching subtle bugs
3. Use golangci-lint for comprehensive linting with multiple tools
4. Fix issues incrementally to maintain code quality

This setup gives you the highest return on investment in terms of reliability and code quality. 