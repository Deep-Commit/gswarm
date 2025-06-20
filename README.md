<p align="center">
  <a href="https://gswarm.dev" target="_blank">
    <img
      src="https://gswarm.dev/favicon.svg"
      alt="GSWARM logo"
      width="140"                      
      height="140"
    />
  </a>
</p>

<p align="center">
  <a href="https://gswarm.dev/docs" target="_blank">
    <img
      alt="GSWARM Docs"
      src="https://img.shields.io/static/v1?label=Docs&message=gswarm.dev%2Fdocs&color=3E7BF6&style=for-the-badge&logo=readthedocs"
    />
  </a>
  <a href="https://gswarm.dev" target="_blank">
    <img
      alt="Live Site"
      src="https://img.shields.io/static/v1?label=Site&message=gswarm.dev&color=000000&style=for-the-badge&logo=vercel"
    />
  </a>
  <a href="https://discord.gg/wzHhFBcT" target="_blank">
    <img
      alt="Join Discord"
      src="https://img.shields.io/static/v1?label=Chat&message=Discord&color=5865F2&style=for-the-badge&logo=discord"
    />
  </a>
</p>

# GSwarm - Gensyn RL Swarm Supervisor

> **⚠️ Important Notice: This is a third-party application**
> 
> GSwarm is **NOT** affiliated with or endorsed by the official Gensyn team. This is an independent, community-developed supervisor tool designed to enhance the user experience of running Gensyn RL Swarm. We cannot modify the core RL Swarm functionality, training algorithms, or blockchain integration.

A robust Go-based supervisor for Gensyn RL Swarm that provides automatic restart capabilities, dependency management, comprehensive logging, and Telegram monitoring.

## ✨ Features

- 🔄 **Auto-restart**: Automatically restarts the RL Swarm process on errors
- 📊 **Comprehensive Logging**: Detailed logs with timestamps and process IDs
- 🐍 **Python Environment Management**: Automatic Python dependency installation
- 💬 **Interactive CLI**: Handles interactive prompts when flags are missing
- ⚡ **Performance Monitoring**: Real-time output streaming with error detection
- 🛡️ **Graceful Shutdown**: Proper signal handling for clean process termination
- 🚀 **Dual Mode**: Supports both command line flags and interactive prompts
- 📱 **Telegram Monitoring**: Real-time blockchain activity notifications via Telegram

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

**Verify installation:**
```bash
gswarm --version
```

#### Option 2: Clone and Build from Source
```bash
git clone https://github.com/Deep-Commit/gswarm.git
cd gswarm
make build
make install
```
After this, you can run `gswarm` from anywhere (if your Go bin directory is in your PATH).

### Basic Usage

1. **Run the supervisor**:
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
gswarm
```

### Command Line Mode

You can provide all configuration via command line flags for non-interactive operation:

```bash
gswarm --config-path config.yaml --hf-token YOUR_TOKEN --identity-path identity.pem --org-id YOUR_ORG_ID --contract-address 0x... --game gsm8k

gswarm --requirements requirements-cpu.txt

gswarm --model-size 32 --big-swarm --org-id YOUR_ORG_ID
```

### Beautiful CLI Help

GSwarm features a beautiful, comprehensive help system:

```bash
# Show main help
gswarm --help

# Show version information
gswarm version

# Show help for specific command
gswarm help version
```

### Command Line Options

| Flag | Description | Default | Environment Variable |
|------|-------------|---------|---------------------|
| `--testnet` | Connect to the Testnet | `false` | `GSWARM_TESTNET` |
| `--big-swarm` | Use big swarm (Math Hard) instead of small swarm (Math) | `false` | `GSWARM_BIG_SWARM` |
| `--model-size` | Parameter count in billions (0.5, 1.5, 7, 32, 72) | `0.5` | `GSWARM_MODEL_SIZE` |
| `--hf-token` | HuggingFace access token for model pushing | | `HUGGINGFACE_ACCESS_TOKEN`, `GSWARM_HF_TOKEN` |
| `--org-id` | Modal ORG_ID (required for testnet) | | `GSWARM_ORG_ID` |
| `--identity-path` | Path to identity PEM file | `swarm.pem` | `GSWARM_IDENTITY_PATH` |
| `--contract-address` | Override smart contract address | Auto-detected | `GSWARM_CONTRACT_ADDRESS` |
| `--game` | Game type ('gsm8k' or 'dapo') | Auto-detected | `GSWARM_GAME` |
| `--config-path` | Path to YAML config file | Auto-detected | `GSWARM_CONFIG_PATH` |
| `--cpu-only` | Force CPU-only mode | `false` | `GSWARM_CPU_ONLY` |
| `--requirements` | Requirements file path (overrides default) | | `GSWARM_REQUIREMENTS` |
| `--interactive` | Force interactive mode (prompt for all options) | `false` | `GSWARM_INTERACTIVE` |

### Environment Variables

All flags can be set via environment variables with the `GSWARM_` prefix:

```bash
export GSWARM_TESTNET=true
export GSWARM_MODEL_SIZE=7
export GSWARM_ORG_ID=your-org-id
export HUGGINGFACE_ACCESS_TOKEN=your-token
gswarm
```

### HuggingFace Token Handling

The supervisor intelligently handles HuggingFace tokens:

- **If provided via `--hf-token`**: Uses the provided token without prompting
- **If not provided**: Prompts interactively asking if you want to push models to HuggingFace Hub

```bash
# No prompt for HF token (provided via command line)
gswarm --hf-token YOUR_TOKEN --org-id YOUR_ORG_ID

# Will prompt for HF token (not provided)
gswarm --org-id YOUR_ORG_ID
```

### Non-Interactive Mode Examples

```bash
gswarm \
  --org-id YOUR_ORG_ID \
  --identity-path /path/to/identity.pem \
  --hf-token YOUR_HF_TOKEN \
  --model-size 7 \
  --big-swarm \
  --cpu-only

gswarm \
  --org-id YOUR_ORG_ID \
  --identity-path /path/to/identity.pem \
  --hf-token YOUR_HF_TOKEN \
  --model-size 0.5 \
  --cpu-only
```

**Key Benefits of Non-Interactive Mode:**
- No manual input required during startup
- Perfect for automated deployments and scripts
- Consistent configuration across runs
- Faster startup time

### Examples

```bash
# Interactive mode (default)
gswarm

# Non-interactive mode with all options
gswarm --testnet --big-swarm --model-size 7 --org-id YOUR_ORG_ID --hf-token YOUR_TOKEN

# CPU-only mode
gswarm --cpu-only --model-size 0.5

# Custom requirements file
gswarm --requirements requirements-gpu.txt

# Show version
gswarm version
```

## 📱 Telegram Monitoring

GSwarm includes a powerful Telegram monitoring service that provides real-time notifications about your blockchain activity, including votes, rewards, and balance changes.

### Features

- 🔔 **Real-time Notifications**: Get instant updates on blockchain activity
- 📊 **Vote Tracking**: Monitor your voting activity and changes
- 💰 **Reward Monitoring**: Track reward accumulation and changes
- 💎 **Balance Updates**: Monitor your wallet balance changes
- 📈 **Change Detection**: Only notified when values actually change
- 🛡️ **Secure Configuration**: Local config file storage
- 🔍 **Peer ID Monitoring**: Track all peer IDs associated with your EOA address

### Setup Instructions

#### 1. Create a Telegram Bot

1. **Start a chat with @BotFather** on Telegram
2. **Send `/newbot`** and follow the instructions
3. **Choose a name** for your bot (e.g., "GSwarm Monitor")
4. **Choose a username** (must end with 'bot', e.g., "gswarm_monitor_bot")
5. **Save the bot token** provided by BotFather

#### 2. Get Your Chat ID

1. **Start a chat with your bot** by clicking the link or searching for it
2. **Send any message** to the bot (e.g., "Hello")
3. **Visit this URL** in your browser (replace `YOUR_BOT_TOKEN` with your actual token):
   ```
   https://api.telegram.org/botYOUR_BOT_TOKEN/getUpdates
   ```
4. **Find your chat ID** in the response (look for `"chat":{"id":123456789}`)

#### 3. Get Your EOA Address

1. **Visit the Gensyn Dashboard** at https://dashboard.gensyn.ai
2. **Log in** to your account
3. **Navigate to your profile or settings** section
4. **Copy your EOA (Externally Owned Account) address** - this is your Ethereum wallet address
5. **The address should start with "0x"** and be 42 characters long (e.g., `0x1234567890abcdef...`)

#### 4. Run the Telegram Service

```bash
# Basic usage (will prompt for bot token, chat ID, and EOA address)
gswarm --telegram

# With custom config path
gswarm --telegram --telegram-config-path /path/to/telegram-config.json

# Force update configuration
gswarm --telegram --update-telegram-config
```

### Telegram Command Options

| Flag | Description | Default | Environment Variable |
|------|-------------|---------|---------------------|
| `--telegram` | Start Telegram monitoring service | `false` | `GSWARM_TELEGRAM` |
| `--telegram-config-path` | Path to telegram-config.json | `telegram-config.json` | `GSWARM_TELEGRAM_CONFIG_PATH` |
| `--update-telegram-config` | Force update of Telegram config | `false` | `GSWARM_UPDATE_TELEGRAM_CONFIG` |

### Configuration Files

The Telegram service creates and manages these files:

- **`telegram-config.json`**: Stores your bot token and chat ID
- **`telegram_previous_data.json`**: Tracks previous blockchain data for change detection

### Example Usage

```bash
# First time setup (interactive)
gswarm --telegram

# Run with existing config
gswarm --telegram

# Update configuration
gswarm --telegram --update-telegram-config

# Custom config path
gswarm --telegram --telegram-config-path /path/to/telegram-config.json
```

### What You'll Receive

The Telegram service monitors and notifies you about:

- **Vote Changes**: When your vote count increases or decreases
- **Reward Changes**: When your accumulated rewards change
- **Balance Changes**: When your wallet balance changes
- **Peer ID Activity**: Monitoring of all peer IDs associated with your EOA address
- **Welcome Message**: Initial setup confirmation

### Sample Notifications

```
🚀 G-Swarm Update

📊 Blockchain Data Update

👤 EOA Address: 0x1234567890abcdef...
🔍 Peer IDs Monitored: 4

📈 Total Votes: 1,456 📈
💰 Total Rewards: 0.75 ETH 📈

📋 Per-Peer Breakdown:
🔹 Peer 1: QmZkyXja166VBTMU76xLR17XKAny9kAkFgx4fNpoceQ8LT
   📈 Votes: 22
   💰 Rewards: 3075

🔹 Peer 2: QmYJeqmiqLNC5cosqE76wZSVdBEHL5Mq9zwFUX61d2fAzn
   📈 Votes: 0
   💰 Rewards: 0

⏰ Last Check: 2025-06-20 20:05:23
```

### Troubleshooting Telegram

1. **"Bot token invalid"**
   - Verify your bot token from @BotFather
   - Use `--update-telegram-config` to re-enter it

2. **"Chat ID not found"**
   - Make sure you've sent a message to your bot
   - Check the getUpdates URL response
   - Use `--update-telegram-config` to re-enter it

3. **"No notifications received"**
   - Check that your bot is active
   - Verify the chat ID is correct
   - Ensure you have blockchain activity to monitor

4. **"EOA address not found"**
   - Get your EOA address from the Gensyn dashboard
   - Make sure the address starts with "0x" and is 42 characters long
   - Verify the address is correct

5. **"No peer IDs found"**
   - Ensure your EOA address is correct
   - Check that you have active peer IDs on the Gensyn network
   - The service will automatically detect and monitor all your peer IDs

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

5. **Telegram Monitoring**:
   - Monitors blockchain data via Alchemy API
   - Tracks changes in votes, rewards, and balance
   - Sends formatted notifications via Telegram Bot API

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
   - Use `--hf-token YOUR_TOKEN` to provide the token via command line
   - The prompt only appears when no token is provided

6. **"Invalid model-size value"**
   - Use one of the valid values: `0.5`, `1.5`, `7`, `32`, `72`
   - Example: `gswarm --model-size 7`

7. **"Invalid game type"**
   - Use either `gsm8k` or `dapo` for the game parameter
   - Example: `gswarm --game gsm8k`

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

## 💝 Support the Project

If you find GSwarm helpful and would like to support its development, consider making a donation:

**Ethereum Address:**
```
0xA22e20BA3336f5Bd6eCE959F5ac4083C9693e316
```

Your support helps us:
- Maintain and improve the supervisor tool
- Add new features and enhancements
- Provide better documentation and support
- Keep the project free and open source

Thank you for your support! 🙏

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Related

- [Gensyn RL Swarm](https://github.com/gensyn-ai/rl-swarm) - The main RL Swarm application
- [Documentation](https://docs.gensyn.ai) - Official documentation

## 📋 About This Project

### Third-Party Status
GSwarm is an **independent, community-developed tool** that operates as a supervisor/wrapper around the official Gensyn RL Swarm application. We are not affiliated with the Gensyn team and cannot modify the core RL Swarm functionality.

### What We Can Do
- Process management and supervision
- Environment setup and dependency management
- Monitoring and logging
- Configuration management
- User experience improvements
- Telegram monitoring and notifications

### What We Cannot Do
- Modify training algorithms
- Change blockchain smart contracts
- Alter model architectures
- Modify core hivemind functionality
- Change the official Gensyn protocol

### Support
For issues related to the core RL Swarm application, please contact the official Gensyn team. For issues with GSwarm itself, please use our GitHub issues page.

## 🗺️ Roadmap

For detailed information about upcoming features and development plans, see our [Development Roadmap](ROADMAP.md).

**Note**: GSwarm is a **third-party supervisor tool** for Gensyn RL Swarm. We cannot modify the core RL Swarm functionality, training algorithms, or blockchain integration. Our scope is limited to process management, monitoring, and user experience improvements.

### Current Development Focus (Q3 2025)
- **Enhanced Monitoring**: Real-time performance metrics collection
- **Configuration Profiles**: Save/load configuration presets  
- **Improved Error Handling**: Better error classification and recovery
- **Multi-Node Support**: Basic management of multiple GPU nodes
- **Telegram Enhancements**: More detailed notifications and analytics

### Upcoming Features
- **Local GUI Application**: Desktop application for monitoring and control
- **Real-time Dashboard**: Visual monitoring with charts and graphs
- **Configuration Management**: Visual profile editor and templates
- **System Integration**: System tray, notifications, and auto-start
- **Advanced Telegram Features**: Custom notification schedules and filters
