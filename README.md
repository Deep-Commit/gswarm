# GSwarm - Gensyn Telegram Monitoring Service

> **⚠️ Important Notice: This is a third-party application**
> 
> GSwarm is **NOT** affiliated with or endorsed by the official Gensyn team. This is an independent, community-developed Telegram monitoring service designed to enhance the user experience of monitoring Gensyn AI node activity.

A dedicated Go-based Telegram monitoring service for Gensyn AI that provides real-time notifications about blockchain activity, including votes, rewards, and balance changes.

## ✨ Features

- 🔔 **Real-time Notifications**: Get instant updates on blockchain activity
- 📊 **Vote Tracking**: Monitor your voting activity and changes
- 💰 **Reward Monitoring**: Track reward accumulation and changes
- 💎 **Balance Updates**: Monitor your wallet balance changes
- 📈 **Change Detection**: Only notified when values actually change
- 🛡️ **Secure Configuration**: Local config file storage
- 🔍 **Peer ID Monitoring**: Track all peer IDs associated with your EOA address
- ⏰ **Continuous Monitoring**: Checks every 5 minutes for changes
- 🤖 **Discord Integration**: Optional verification system with role assignment
- 🔐 **Secure Verification**: Time-limited codes for Discord community access

## 🔐 Discord Verification (Optional)

GSwarm includes an optional Discord verification system that allows you to get verified in the G-Swarm Discord community. **Verification is completely optional and not required to use the monitoring service.**

### What is Discord Verification?

The Discord verification system allows you to:
- Get a special "GSwarm" role in the Discord server
- Access verified-only channels and features
- Show your verification status to other community members
- Display your node statistics and ranking information

### How to Get Verified

1. **Start the Telegram bot** and use the `/verify` command
2. **Receive a verification code** (valid for 15 minutes)
3. **Join the Discord server**: [https://discord.gg/gswarm](https://discord.gg/gswarm)
4. **Use the verification command**: `/verify <code>` in Discord
5. **Get automatically assigned** the "GSwarm" role

### Verification Features

- **Cosmetic Role**: The "GSwarm" role is purely cosmetic with no special permissions
- **Automatic Assignment**: The Discord bot automatically assigns the role upon successful verification
- **No Conflicts**: The verification role won't interfere with your existing role colors
- **Fallback Support**: If verification fails, you can still use all monitoring features

### Verification Requirements

- Active Gensyn AI node with peer IDs
- Valid EOA address registered with Gensyn
- Telegram bot access for verification codes
- Discord account to join the server

### Important Notes

- **Verification is optional**: You can use all monitoring features without verification
- **No special permissions**: The verification role is cosmetic only
- **Privacy focused**: Only your verification status is shared, not personal data
- **Community driven**: This is a community feature, not an official Gensyn requirement

### Troubleshooting Verification

1. **"Verification code expired"**
   - Generate a new code with `/verify` in Telegram
   - Codes expire after 15 minutes for security

2. **"Verification failed"**
   - Ensure your EOA address is correct
   - Check that you have active peer IDs
   - Verify the code was entered correctly

3. **"Role not assigned"**
   - Check that the Discord bot has "Manage Roles" permission
   - Ensure the bot's role is positioned correctly in the hierarchy
   - Contact server administrators if issues persist

## 🚀 Quick Start

### Prerequisites

- Go 1.24+ (for building the service)
- Telegram Bot Token (from @BotFather)
- Telegram Chat ID (your chat ID or group chat ID)

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

1. **Run the monitoring service**:
   ```bash
   gswarm
   ```

The service will:
- Prompt for Telegram configuration if not already set up
- Ask for your EOA address
- Fetch associated peer IDs
- Start monitoring blockchain activity every 5 minutes
- Send notifications only when changes are detected

## 📖 Usage

### Interactive Setup (Default)

When run for the first time, the service will guide you through setup:

```bash
gswarm
```

You'll be prompted for:
1. **Telegram Bot Token**: Get this from @BotFather on Telegram
2. **Telegram Chat ID**: Your personal chat ID or group chat ID
3. **EOA Address**: Your Ethereum address from the Gensyn dashboard

### Command Line Options

| Flag | Description | Default | Environment Variable |
|------|-------------|---------|---------------------|
| `--telegram-config-path` | Path to telegram-config.json file | `telegram-config.json` | `GSWARM_TELEGRAM_CONFIG_PATH` |
| `--update-telegram-config` | Force update of Telegram config via CLI prompts | `false` | `GSWARM_UPDATE_TELEGRAM_CONFIG` |

### Environment Variables

All flags can be set via environment variables with the `GSWARM_` prefix:

```bash
export GSWARM_TELEGRAM_CONFIG_PATH=/path/to/config.json
export GSWARM_UPDATE_TELEGRAM_CONFIG=true
gswarm
```

### Examples

```bash
# Start monitoring service (interactive setup)
gswarm

# Use custom config path
gswarm --telegram-config-path /path/to/config.json

# Force update Telegram config
gswarm --update-telegram-config

# Show version
gswarm version
```

## 📱 Telegram Monitoring

GSwarm includes a powerful Telegram monitoring service that provides real-time notifications about your blockchain activity, including votes, rewards, and balance changes.

### Setup Instructions

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

1. **Configuration Management**:
   - Stores Telegram bot token and chat ID locally
   - Prompts for EOA address on first run
   - Supports custom config file paths

2. **Blockchain Monitoring**:
   - Queries GenRL-Swarm contract (0xFaD7C5e93f28257429569B854151A1B8DCD404c2)
   - Uses Alchemy API for blockchain data access
   - Monitors votes, rewards, and balance changes

3. **Peer ID Detection**:
   - Automatically fetches all peer IDs associated with your EOA address
   - Monitors each peer ID individually
   - Provides per-peer breakdown in notifications

4. **Change Detection**:
   - Compares current values with previous values
   - Only sends notifications when changes are detected
   - Stores previous data locally for comparison

5. **Telegram Integration**:
   - Sends formatted HTML messages via Telegram Bot API
   - Includes change indicators (📈 for increases, 📉 for decreases)
   - Provides detailed per-peer breakdown

## 📝 Logging

The service provides detailed console output including:
- Configuration loading and validation
- Blockchain data queries and results
- Peer ID detection and monitoring
- Telegram message sending status
- Error handling and recovery

Example output:
```
[2025-01-01 12:00:00] Starting Telegram monitoring service...
[2025-01-01 12:00:01] Loaded Telegram config: BotToken=***, ChatID=123456789
[2025-01-01 12:00:02] Sending welcome message...
[2025-01-01 12:00:03] Message sent successfully to Telegram!
[2025-01-01 12:00:04] Please provide your EOA address to start monitoring...
[2025-01-01 12:00:10] Fetching peer IDs for address: 0x1234567890abcdef...
[2025-01-01 12:00:15] Successfully loaded 4 peer IDs for monitoring
[2025-01-01 12:00:16] Starting continuous monitoring loop (checking every 5 minutes)...
```

## 🛠️ Development

### Building from Source

```bash
git clone https://github.com/Deep-Commit/gswarm.git
cd gswarm
make build
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

1. **"gswarm command not found"**
   - Ensure Go 1.24+ is installed and `$GOPATH/bin` is in your PATH
   - Reinstall with: `go install github.com/Deep-Commit/gswarm/cmd/gswarm@latest`

2. **"Telegram API error"**
   - Verify your bot token is correct
   - Ensure your bot is active and not blocked
   - Check that the chat ID is valid

3. **"No blockchain data found"**
   - Verify your EOA address is correct
   - Check that you have active peer IDs on the Gensyn network
   - Ensure the contract address is accessible

4. **"Permission denied"**
   - Make sure all files are executable as needed
   - Ensure proper file permissions for config files

### Debug Mode

The service provides detailed output by default. For additional debugging, check the console output for specific error messages.

## 📋 Requirements

### System Requirements
- Go 1.24+
- Network connectivity for blockchain queries and Telegram API
- Telegram Bot Token (from @BotFather)
- Telegram Chat ID

### File Structure
```
~/.gswarm/
├── telegram-config.json      # Telegram configuration
└── telegram_previous_data.json # Previous blockchain data
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
- Maintain and improve the monitoring service
- Add new features and enhancements
- Provide better documentation and support
- Keep the project free and open source

Thank you for your support! 🙏

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Related

- [Gensyn AI](https://gensyn.ai) - Official Gensyn AI platform
- [Documentation](https://docs.gensyn.ai) - Official documentation

## 📋 About This Project

### Third-Party Status
GSwarm is an **independent, community-developed tool** that provides Telegram monitoring for Gensyn AI node activity. We are not affiliated with the Gensyn team and operate as a separate monitoring service.

### What We Do
- Monitor blockchain activity via smart contract queries
- Track votes, rewards, and balance changes
- Send real-time notifications via Telegram
- Provide detailed per-peer breakdown
- Store configuration and data locally

### Support
For issues related to the core Gensyn AI platform, please contact the official Gensyn team. For issues with GSwarm itself, please use our GitHub issues page.

## 🗺️ Roadmap

For detailed information about upcoming features and development plans, see our [Development Roadmap](ROADMAP.md).

### Current Development Focus (Q3 2025)
- **Enhanced Notifications**: More detailed analytics and charts
- **Custom Schedules**: Configurable monitoring intervals
- **Multiple Addresses**: Support for monitoring multiple EOA addresses
- **Historical Data**: Track and display historical trends
- **Advanced Filtering**: Custom notification filters and thresholds

### Upcoming Features
- **Web Dashboard**: Visual monitoring interface
- **Mobile App**: Native mobile monitoring app
- **API Integration**: REST API for external integrations
- **Advanced Analytics**: Detailed performance metrics and insights
- **Multi-Platform Support**: Support for additional messaging platforms

## Account Linking and Verification

To securely link your Discord and Telegram accounts for G-Swarm:

### 1. Join the Gensyn Discord
- [https://discord.gg/gensyn](https://discord.gg/gensyn)

### 2. Get a Linking Code in Discord
- In the Gensyn Discord, use the command: `/link-telegram`
- The bot will send you a unique linking code via DM

### 3. Verify in Telegram
- In the G-Swarm Telegram bot, use the command: `/verify <code>`
  - Replace `<code>` with the code you received from Discord
- Example: `/verify ABC123`

### 4. Success
- If the code is valid, your Discord and Telegram accounts will be linked
- You will receive a confirmation message in Telegram

**Notes:**
- Each code can only be used once and expires after 10 minutes
- If you have issues, check the debug output or contact the G-Swarm team in Discord