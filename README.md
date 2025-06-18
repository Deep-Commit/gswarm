# Gensyn RL Swarm Supervisor

A robust Go-based supervisor for Gensyn RL Swarm that provides automatic restart capabilities, dependency management, and comprehensive logging.

## ✨ Features

- 🔄 **Auto-restart**: Automatically restarts the RL Swarm process on errors
- 📊 **Comprehensive Logging**: Detailed logs with timestamps and process IDs
- 🐍 **Python Environment Management**: Automatic Python dependency installation
- 💬 **Interactive CLI**: Handles interactive prompts when flags are missing
- 🎯 **Smart Script Discovery**: Automatically finds the entrypoint script in common locations
- ⚡ **Performance Monitoring**: Real-time output streaming with error detection
- 🛡️ **Graceful Shutdown**: Proper signal handling for clean process termination
- 🚀 **Dual Mode**: Supports both command line flags and interactive prompts

## 🚀 Quick Start

### Prerequisites

- Go 1.21+ (for building the supervisor)
- Python 3.10+ (for the RL Swarm application)
- Bash (for running the entrypoint script)

### Installation

1. **Install globally**:
   ```bash
   go install github.com/deep-commit/gensyn-rl-swarm-supervisor/cmd/gswarm@latest
   ```

2. **Navigate to your Gensyn RL Swarm directory** (where `run_rl_swarm.sh` is located):
   ```bash
   cd /path/to/your/gensyn-rl-swarm
   ```

3. **Run the supervisor**:
   ```bash
   gswarm
   ```

The supervisor will:
- Automatically find `run_rl_swarm.sh` and handle any interactive prompts
- Install Python dependencies from `requirements.txt` or `requirements-*.txt`
- Start the RL Swarm process with automatic restart on errors

## 📖 Usage

### Interactive Mode (Default)

When run without any flags, the supervisor will prompt for all necessary configuration:

```bash
# Navigate to directory with run_rl_swarm.sh
cd /path/to/gensyn-rl-swarm

# Run with interactive prompts
gswarm
```

### Command Line Mode

You can provide all configuration via command line flags for non-interactive operation:

```bash
# Navigate to directory with run_rl_swarm.sh
cd /path/to/gensyn-rl-swarm

# Run with all parameters
gswarm -config config.yaml -hf_token YOUR_TOKEN -identity_path identity.pem -org_id YOUR_ORG_ID -contract_address 0x... -game chess

# Run with custom requirements file
gswarm -requirements requirements-cpu.txt

# Run with large model and big swarm
gswarm -model-size 32 -big-swarm -org_id YOUR_ORG_ID
```

### Command Line Options

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `-hf_token` | HuggingFace token | | No |
| `-org_id` | Organization ID | | No |
| `-identity_path` | Identity PEM path | `swarm.pem` | No |
| `-contract_address` | Contract address | Auto-detected | No |
| `-game` | Game type | Auto-detected | No |
| `-config` | Config file path | Auto-detected | No |
| `-requirements` | Requirements file path | | No |
| `-model-size` | Model size in billions (0.5, 1.5, 7, 32, 72) | `0.5` | No |
| `-big-swarm` | Use big swarm (Math Hard) instead of small swarm (Math) | `false` | No |
| `-cpu-only` | Force CPU-only mode | `true` | No |

### HuggingFace Token Handling

The supervisor intelligently handles HuggingFace tokens:

- **If provided via `-hf_token`**: Uses the provided token without prompting
- **If not provided**: Prompts interactively asking if you want to push models to HuggingFace Hub

```bash
# No prompt for HF token (provided via command line)
gswarm -hf_token YOUR_TOKEN -org_id YOUR_ORG_ID

# Will prompt for HF token (not provided)
gswarm -org_id YOUR_ORG_ID
```

### Non-Interactive Mode Examples

```bash
# Navigate to directory with run_rl_swarm.sh
cd /path/to/gensyn-rl-swarm

# Run with all configuration via flags (no CLI prompts)
gswarm \
  -org_id YOUR_ORG_ID \
  -identity_path /path/to/identity.pem \
  -hf_token YOUR_HF_TOKEN \
  -model-size 7 \
  -big-swarm \
  -cpu-only

# Or run with small swarm (default)
gswarm \
  -org_id YOUR_ORG_ID \
  -identity_path /path/to/identity.pem \
  -hf_token YOUR_HF_TOKEN \
  -model-size 0.5 \
  -cpu-only
```

**Key Benefits of Non-Interactive Mode:**
- No manual input required during startup
- Perfect for automated deployments and scripts
- Consistent configuration across runs
- Faster startup time

**Environment Variables Set Automatically:**
- `CONNECT_TO_TESTNET=true` (when ORG_ID is provided)
- `GAME=gsm8k` (small swarm) or `GAME=dapo` (big swarm)
- `USE_BIG_SWARM=true/false`
- `PARAM_B=<model-size>`
- `CPU_ONLY=true/false`
- `HUGGINGFACE_ACCESS_TOKEN=` (empty to skip prompts)
- `PUB_MULTI_ADDRS=` (empty to use bash script defaults)

The supervisor lets the bash script handle multi-address defaults internally, avoiding hardcoded networking configuration.

### Examples

```bash
# Basic usage with default script discovery
gswarm -config config.yaml

# Specify custom script location
gswarm -script /home/user/projects/gensyn-rl-swarm/run_rl_swarm.sh

# Run with specific requirements
gswarm -requirements requirements-gpu.txt

# Run with large model and big swarm
gswarm -model-size 32 -big-swarm -org_id YOUR_ORG_ID

# Run with custom game type
gswarm -game chess -org_id YOUR_ORG_ID

# Run with HF token provided (no prompt)
gswarm -hf_token YOUR_TOKEN -org_id YOUR_ORG_ID

# Run without HF token (will prompt)
gswarm -org_id YOUR_ORG_ID
```

## 🔧 How It Works

1. **Script Discovery**: The supervisor automatically finds `run_rl_swarm.sh` in:
   - Current directory
   - Parent directory
   - `./scripts/` subdirectory
   - `./bin/` subdirectory

2. **Dependency Management**: 
   - Checks Python 3.10+ availability
   - Installs dependencies from `requirements.txt` or `requirements-*.txt`
   - Supports custom requirements files

3. **Process Management**:
   - Starts the bash script with provided arguments
   - Streams output in real-time
   - Monitors for error patterns

4. **Error Handling**:
   - Detects specific error messages
   - Automatically restarts the process
   - Implements exponential backoff

5. **Configuration Modes**:
   - **Command Line Mode**: Uses provided flags, prompts only for missing required values
   - **Interactive Mode**: Prompts for all configuration interactively

## 📝 Logging

Logs are written to `logs/gensyn_rl_swarm_go.log` with:
- Timestamps with microsecond precision
- Process IDs for tracking
- All stdout/stderr output
- Supervisor events (starts, restarts, errors)

Example log entry:
```
2024-01-01 12:00:00.000000 Starting with command: bash /path/to/run_rl_swarm.sh --config config.yaml
2024-01-01 12:00:01.123456 [PID 12345] >> Starting RL Swarm...
2024-01-01 12:00:02.234567 [PID 12345] >> Loading configuration...
```

## 🛠️ Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/deep-commit/gensyn-rl-swarm-supervisor.git
cd gensyn-rl-swarm-supervisor

# Build for current platform
make build

# Build for all platforms
make build-all

# Install locally
make install
```

### Testing

```bash
# Run tests
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

## 🐛 Troubleshooting

### Common Issues

1. **"python3 not found"**
   - Ensure Python 3.10+ is installed and in PATH
   - Use `python3 --version` to verify

2. **"run_rl_swarm.sh not found"**
   - Navigate to the directory containing `run_rl_swarm.sh`
   - Or specify the full path with `-script /path/to/run_rl_swarm.sh`

3. **"Requirements installation failed"**
   - Ensure `requirements.txt` exists in the same directory as `run_rl_swarm.sh`
   - Check network connectivity for pip install
   - Use `-skip-install` to bypass if needed

4. **"Permission denied"**
   - Make the script executable: `chmod +x run_rl_swarm.sh`
   - Ensure proper file permissions

5. **"gswarm command not found"**
   - Ensure Go is installed and `$GOPATH/bin` is in your PATH
   - Reinstall with: `go install github.com/deep-commit/gensyn-rl-swarm-supervisor/cmd/gswarm@latest`

6. **"HF token prompt appears when not expected"**
   - Use `-hf_token YOUR_TOKEN` to provide the token via command line
   - The prompt only appears when no token is provided

### Debug Mode

Set environment variable for verbose logging:
```bash
export SWARM_DEBUG=1
gswarm
```

## 📋 Requirements

### System Requirements
- Go 1.21+
- Python 3.10+
- Bash
- Network connectivity for dependency installation

### File Structure
```
your-gensyn-rl-swarm-project/
├── run_rl_swarm.sh          # Main entrypoint script
├── requirements.txt          # Python dependencies
├── config.yaml              # Configuration file
└── ...                      # Other RL Swarm files
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Related

- [Gensyn RL Swarm](https://github.com/gensyn-ai/rl-swarm) - The main RL Swarm application
- [Documentation](https://docs.gensyn.ai) - Official documentation