# Contributing to RL Swarm Supervisor

Thank you for your interest in contributing to RL Swarm Supervisor! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites

- Go 1.21 or later
- Git
- Make (optional, for using the Makefile)

### Setting Up Your Development Environment

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/your-username/gensyn-rl-swarm-supervisor.git
   cd gensyn-rl-swarm-supervisor
   ```
3. Add the original repository as upstream:
   ```bash
   git remote add upstream https://github.com/deep-commit/gensyn-rl-swarm-supervisor.git
   ```
4. Create a new branch for your feature:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Workflow

### Building the Project

```bash
# Build the application
make build

# Or build directly with go
go build -o gswarm ./cmd/gswarm
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint
```

## Code Style Guidelines

### Go Code Style

- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` to format your code
- Keep functions small and focused
- Add comments for exported functions and types
- Use meaningful variable and function names

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `style:` for formatting changes
- `refactor:` for code refactoring
- `test:` for adding or updating tests
- `chore:` for maintenance tasks

Example:
```
feat: add exponential backoff configuration option

- Add -backoff flag to configure initial backoff duration
- Add -max-backoff flag to configure maximum backoff duration
- Update documentation with new configuration options
```

## Pull Request Process

1. **Create a feature branch** from the main branch
2. **Make your changes** following the code style guidelines
3. **Add tests** for new functionality
4. **Update documentation** if needed
5. **Run tests** to ensure everything works:
   ```bash
   make test
   make lint
   ```
6. **Commit your changes** with a descriptive commit message
7. **Push to your fork** and create a pull request
8. **Wait for review** and address any feedback

### Pull Request Guidelines

- Provide a clear description of the changes
- Include any relevant issue numbers
- Add screenshots or examples if applicable
- Ensure all CI checks pass
- Keep pull requests focused and reasonably sized

## Issue Reporting

When reporting issues, please include:

- **Description**: Clear description of the problem
- **Steps to reproduce**: Detailed steps to reproduce the issue
- **Expected behavior**: What you expected to happen
- **Actual behavior**: What actually happened
- **Environment**: OS, Go version, and any relevant details
- **Logs**: Relevant log files or error messages

## Feature Requests

When requesting features, please:

- Describe the feature clearly
- Explain the use case and benefits
- Provide examples if possible
- Consider implementation complexity

## Code Review Process

- All pull requests require at least one review
- Reviews focus on code quality, functionality, and maintainability
- Be respectful and constructive in feedback
- Address all review comments before merging

## Release Process

Releases are managed through GitHub releases:

1. Create a new tag:
   ```bash
   git tag v1.0.1
   git push origin v1.0.1
   ```
2. GitHub Actions will automatically build and create a release
3. Update the changelog in the README.md

## Getting Help

If you need help with contributing:

- Check existing issues and pull requests
- Ask questions in GitHub issues
- Review the documentation in the README.md

## License

By contributing to this project, you agree that your contributions will be licensed under the MIT License.

Thank you for contributing to RL Swarm Supervisor! 🚀 