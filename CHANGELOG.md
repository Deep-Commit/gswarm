# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project setup
- GitHub Actions CI/CD workflows
- Comprehensive documentation
- Multi-platform build support

## [1.0.0] - 2024-01-01

### Added
- Initial release of RL Swarm Supervisor
- Process supervision with automatic restart capabilities
- Error detection based on pattern matching
- Exponential backoff strategy for restart attempts
- Comprehensive logging system
- Graceful shutdown handling
- Real-time output streaming from managed processes
- Command-line interface with configurable options
- Support for HuggingFace tokens, identity files, and contract addresses

### Features
- **Process Management**: Monitors Python RL Swarm training processes
- **Auto-restart**: Automatically restarts processes when specific error patterns are detected
- **Smart Backoff**: Implements exponential backoff with configurable limits
- **Logging**: Detailed logging to `logs/rl_swarm_go.log` with timestamps and process IDs
- **Signal Handling**: Proper handling of SIGINT and SIGTERM for graceful shutdown
- **Error Detection**: Pattern-based error detection for reliable restart triggers

### Technical Details
- Built with Go 1.21+
- Cross-platform support (Linux, macOS, Windows)
- No external dependencies
- Lightweight and efficient process management

### Command Line Options
- `-script`: Bash entrypoint script (default: `run_rl_swarm.sh`)
- `-config`: Configuration file path (required)
- `-hf_token`: HuggingFace token (optional)
- `-identity_path`: Identity PEM file path (optional)
- `-org_id`: Organization ID (optional)
- `-contract_address`: Contract address (optional)
- `-game`: Game type (default: `gsm8k`)

### Error Patterns Detected
- `>> An error was detected while running rl-swarm.`
- `>> Shutting down trainer...`

### Restart Strategy
- Initial backoff: 5 seconds
- Maximum backoff: 5 minutes
- Backoff reset on clean process exit
- Backoff doubling on error-triggered restarts 