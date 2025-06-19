# Linting Tools Comparison

This document compares the three main linting tools used in the GSwarm project.

## Tool Comparison

| Tool | Purpose | Speed | Coverage | Configuration |
|------|---------|-------|----------|---------------|
| **go vet** | Basic Go toolchain checks | ⚡ Fast | 🔴 Limited | Built-in |
| **Staticcheck** | Advanced static analysis | 🚀 Fast | 🟢 Comprehensive | Simple |
| **golangci-lint** | Multi-linter orchestration | 🐌 Slower | 🟡 Very Comprehensive | Complex |

## go vet

**What it is:** Built-in Go tool that performs basic correctness checks.

**Speed:** Very fast (built into Go toolchain)

**What it catches:**
- Bad Printf verbs
- Unreachable code
- Incorrect struct tags
- Basic API misuse
- Suspicious assignments
- Suspicious function calls

**Example output:**
```bash
$ go vet ./...
# No output = no issues found
```

**When to use:**
- Quick feedback during development
- Pre-commit hooks
- CI/CD pipelines (always run this)

## Staticcheck

**What it is:** Advanced static analysis tool with 150+ checks.

**Speed:** Fast (optimized for Go)

**What it catches:**
- Everything go vet catches, plus:
- Unused variables and functions
- Inefficient string concatenation
- Unnecessary type conversions
- Potential nil pointer dereferences
- Race conditions
- API misuse patterns
- Dead code elimination opportunities
- Performance issues
- Style violations

**Example output:**
```bash
$ staticcheck -checks=all ./...
cmd/gswarm/main.go:555:6: func getKeys is unused (U1000)
internal/prompt/prompt.go:15:9: Error return value of `reader.ReadString` is not checked (errcheck)
```

**When to use:**
- Comprehensive code review
- Performance analysis
- Code quality audits
- Finding subtle bugs

## golangci-lint

**What it is:** Orchestrator that runs multiple linters in parallel.

**Speed:** Slower (runs many tools)

**What it catches:**
- Everything Staticcheck catches, plus:
- Additional linters (errcheck, revive, stylecheck, etc.)
- Customizable rule sets
- Parallel execution of multiple tools

**Example output:**
```bash
$ golangci-lint run
internal/prompt/prompt.go:15:9: Error return value of `reader.ReadString` is not checked (errcheck)
internal/prompt/prompt.go:11:6: exported: func name will be used as prompt.PromptUser by other packages, and that stutters (revive)
```

**When to use:**
- Comprehensive linting
- Custom rule configuration
- Team-wide code standards enforcement

## Recommended Workflow

### Development Workflow

1. **Quick feedback:** `make lint-vet`
   ```bash
   # Fast, basic checks
   go vet ./...
   ```

2. **Comprehensive analysis:** `make lint-staticcheck`
   ```bash
   # Advanced static analysis
   staticcheck -checks=all ./...
   ```

3. **Full linting:** `make lint`
   ```bash
   # Complete linting suite
   golangci-lint run
   ```

### CI/CD Pipeline

```yaml
# Always run these in order
- name: Run go vet
  run: go vet ./...

- name: Run Staticcheck
  run: staticcheck -checks=all ./...

- name: Run golangci-lint
  run: golangci-lint run
```

### Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Running pre-commit checks..."

# Quick checks first
make lint-vet
if [ $? -ne 0 ]; then
    echo "❌ go vet failed"
    exit 1
fi

# Comprehensive analysis
make lint-staticcheck
if [ $? -ne 0 ]; then
    echo "❌ Staticcheck failed"
    exit 1
fi

echo "✅ Pre-commit checks passed"
```

## Performance Comparison

| Tool | Time (small project) | Time (large project) | Memory Usage |
|------|---------------------|---------------------|--------------|
| go vet | < 1s | 2-5s | Low |
| Staticcheck | 1-3s | 5-15s | Low |
| golangci-lint | 5-10s | 30-60s | Medium |

## Configuration Complexity

| Tool | Configuration | Learning Curve |
|------|---------------|----------------|
| go vet | None needed | None |
| Staticcheck | Simple flags | Low |
| golangci-lint | Complex YAML | High |

## Key Takeaways

1. **Always run go vet** - It's built-in, fast, and catches basic issues
2. **Staticcheck is the most impactful** - 150+ checks, fast, comprehensive
3. **golangci-lint for teams** - When you need multiple linters and custom rules
4. **Start simple** - Use go vet + Staticcheck for most cases
5. **Add golangci-lint** - When you need team-wide standards and custom rules

## Migration Path

If you're new to these tools:

1. **Start with go vet** - It's already available
2. **Add Staticcheck** - Install and run with `-checks=all`
3. **Consider golangci-lint** - When you need more customization
4. **Configure exclusions** - Only when necessary

## Conclusion

Staticcheck truly is the most impactful tool. It provides the best balance of:
- **Coverage** - 150+ different checks
- **Performance** - Fast analysis
- **Simplicity** - Easy to configure and use
- **Accuracy** - Low false positive rate

Combined with go vet and golangci-lint, you get comprehensive coverage with excellent performance. 