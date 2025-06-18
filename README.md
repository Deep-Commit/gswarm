# Gensyn RL Swarm Supervisor

A robust Go-based supervisor for Gensyn RL Swarm that provides automatic restart capabilities, dependency management, and comprehensive logging.

## ✨ Features

- 🔄 **Auto-restart**: Automatically restarts the RL Swarm process on errors
- 📊 **Comprehensive Logging**: Detailed logs with timestamps and process IDs
- 🐍 **Python Environment Management**: Automatic Python dependency installation
- 💬 **Interactive CLI**: Handles interactive prompts when flags are missing
- ⚡ **Performance Monitoring**: Real-time output streaming with error detection
- 🛡️ **Graceful Shutdown**: Proper signal handling for clean process termination
- 🚀 **Dual Mode**: Supports both command line flags and interactive prompts

## 🚀 Quick Start

### Prerequisites

- Go 1.21+ (for building the supervisor)
- Python 3.10+ (for the RL Swarm application)

### Installation

#### Option 1: Install with Go (Recommended)
```bash
go install github.com/Deep-Commit/gswarm/cmd/gswarm@latest
```
This will place the `gswarm` binary in your `$GOPATH/bin` or `$HOME/go/bin` (make sure this is in your PATH).

#### Option 2: Clone and Build from Source
```bash
git clone https://github.com/Deep-Commit/gswarm.git
cd gswarm
make build
make install
```
After this, you can run `gswarm` from anywhere (if your Go bin directory is in your PATH).

2. **Navigate to your Gensyn RL Swarm directory** (where your RL Swarm code and config are located):
   ```bash
   cd /path/to/your/gensyn-rl-swarm
   ```

3. **Run the supervisor**:
   ```bash
   gswarm
   ```

The supervisor will:
- Automatically handle Python dependencies from `requirements.txt` or `requirements-*.txt`
- Start and supervise the RL Swarm process with automatic restart on errors

## 📖 Usage

### Interactive Mode (Default)

When run without any flags, the supervisor will prompt for all necessary configuration:

```bash
cd /path/to/gensyn-rl-swarm

gswarm
```

### Command Line Mode

You can provide all configuration via command line flags for non-interactive operation:

```bash
gswarm -config config.yaml -hf_token YOUR_TOKEN -identity_path identity.pem -org_id YOUR_ORG_ID -contract_address 0x... -game chess

gswarm -requirements requirements-cpu.txt

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
gswarm \
  -org_id YOUR_ORG_ID \
  -identity_path /path/to/identity.pem \
  -hf_token YOUR_HF_TOKEN \
  -model-size 7 \
  -big-swarm \
  -cpu-only

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
- `PUB_MULTI_ADDRS=` (empty to use defaults)

### Examples

```bash
gswarm -config config.yaml
gswarm -requirements requirements-gpu.txt
gswarm -model-size 32 -big-swarm -org_id YOUR_ORG_ID
gswarm -game chess -org_id YOUR_ORG_ID
gswarm -hf_token YOUR_TOKEN -org_id YOUR_ORG_ID
gswarm -org_id YOUR_ORG_ID
```

## 🔧 How It Works

1. **Dependency Management**: 
   - Checks Python 3.10+ availability
   - Installs dependencies from `requirements.txt` or `requirements-*.txt`
   - Supports custom requirements files

2. **Process Management**:
   - Starts and supervises the RL Swarm process with provided arguments
   - Streams output in real-time
   - Monitors for error patterns

3. **Error Handling**:
   - Detects specific error messages
   - Automatically restarts the process
   - Implements exponential backoff

4. **Configuration Modes**:
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
2024-01-01 12:00:00.000000 Starting RL Swarm with config: config.yaml
2024-01-01 12:00:01.123456 [PID 12345] >> Starting RL Swarm...
2024-01-01 12:00:02.234567 [PID 12345] >> Loading configuration...
```

## 🛠️ Development

### Building from Source

```bash
git clone https://github.com/Deep-Commit/gswarm.git
cd gswarm
make build
make build-all
make install
```

### Testing

```bash
make test
make test-coverage
```

### Code Quality

```bash
make fmt
make lint
```

## 🐛 Troubleshooting

### Common Issues

1. **"python3 not found"**
   - Ensure Python 3.10+ is installed and in PATH
   - Use `python3 --version` to verify

2. **"Requirements installation failed"**
   - Ensure `requirements.txt` exists in your RL Swarm directory
   - Check network connectivity for pip install

3. **"Permission denied"**
   - Make sure all files are executable as needed
   - Ensure proper file permissions

4. **"gswarm command not found"**
   - Ensure Go is installed and `$GOPATH/bin` is in your PATH
   - Reinstall with: `go install github.com/Deep-Commit/gswarm/cmd/gswarm@latest`

5. **"HF token prompt appears when not expected"**
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
- Network connectivity for dependency installation

### File Structure
```
your-gensyn-rl-swarm-project/
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