# Quick Start Guide - Secure Account Linking

This guide will help you set up and run the secure cross-platform account linking system for G-Swarm.

## Prerequisites

- Go 1.24 or later
- Discord bot token and guild ID
- Telegram bot token
- A secure API key

## Step 1: Set Up Discord Bot

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application
3. Go to the "Bot" section and create a bot
4. Copy the bot token
5. Go to "OAuth2" > "URL Generator"
6. Select scopes: `bot`, `applications.commands`
7. Select bot permissions: `Send Messages`, `Use Slash Commands`, `Manage Roles`
8. Use the generated URL to invite the bot to your server
9. Copy the guild (server) ID

## Step 2: Set Up Telegram Bot

1. Message [@BotFather](https://t.me/botfather) on Telegram
2. Send `/newbot` and follow the instructions
3. Copy the bot token
4. Start a chat with your bot and send `/start`

## Step 3: Generate API Key

Generate a secure API key:

```bash
# Generate a random 32-character API key
openssl rand -hex 16
```

## Step 4: Configure the System

1. Copy the example configuration:
```bash
cp config.example.yaml config.yaml
```

2. Edit `config.yaml` with your values:
```yaml
api:
  key: "your_generated_api_key"
  addr: ":8080"
  url: "http://localhost:8080"

discord:
  token: "your_discord_bot_token"
  guild_id: "your_guild_id"
  role_id: "optional_role_id"

telegram:
  token: "your_telegram_bot_token"
  chat_id: "optional_chat_id"
```

## Step 5: Build and Run

1. Build the system:
```bash
make build
```

2. Run the server:
```bash
make run-server
```

Or run with custom configuration:
```bash
./build/gswarm-server \
  --api-key=your_api_key \
  --discord-token=your_discord_token \
  --discord-guild=your_guild_id \
  --telegram-token=your_telegram_token
```

## Step 6: Test the System

1. Test the API endpoints:
```bash
make test-account-linking
```

2. Test manually:
```bash
# Issue a code (requires API key)
curl -X POST http://localhost:8080/api/telegrams/issue-code \
  -H "Authorization: Bearer your_api_key" \
  -H "Content-Type: application/json" \
  -d '{"discord_id":"123456789012345678"}'

# Verify a code (public endpoint)
curl -X POST http://localhost:8080/api/telegrams/verify-code \
  -H "Content-Type: application/json" \
  -d '{"code":"a1b2c3","telegram_id":"987654321"}'
```

## Step 7: Use the System

### Discord Commands

1. In your Discord server, use `/link-telegram`
2. The bot will send you a linking code via DM
3. Follow the instructions in the DM

### Telegram Commands

1. In your Telegram bot, use `/verify <code>`
2. Replace `<code>` with the code from Discord
3. Your accounts will be linked automatically

## User Flow Example

1. **User in Discord**: Runs `/link-telegram`
2. **Discord Bot**: Sends code `a1b2c3` via DM
3. **User in Telegram**: Runs `/verify a1b2c3`
4. **System**: Links the accounts and confirms success

## Security Features

- ✅ Only Discord bot can issue codes (API key required)
- ✅ Codes expire after 10 minutes
- ✅ Each code can only be used once
- ✅ Prevents duplicate account links
- ✅ Public verification endpoint (no API key needed)

## Troubleshooting

### Common Issues

1. **Discord bot not responding**
   - Check bot token is correct
   - Verify bot has proper permissions
   - Ensure bot is in the server

2. **Telegram bot not responding**
   - Check bot token is correct
   - Verify bot is not blocked
   - Send `/start` to the bot first

3. **API connection issues**
   - Check server is running on correct port
   - Verify API key is correct
   - Check firewall settings

4. **Codes not working**
   - Codes expire after 10 minutes
   - Each code can only be used once
   - Check for typos in code entry

### Debug Commands

Check linked accounts:
```bash
curl -H "Authorization: Bearer your_api_key" \
  http://localhost:8080/api/telegrams/linked-accounts
```

Check server status:
```bash
curl http://localhost:8080/
```

## Production Deployment

For production deployment:

1. **Use environment variables** instead of config files
2. **Generate a strong API key** and keep it secure
3. **Set up proper logging** and monitoring
4. **Use HTTPS** for the API server
5. **Implement rate limiting** and IP restrictions
6. **Set up database persistence** (PostgreSQL/Redis)
7. **Use a reverse proxy** (nginx/traefik)
8. **Set up monitoring** and alerting

### Environment Variables

```bash
export API_KEY="your_secure_api_key"
export DISCORD_BOT_TOKEN="your_discord_token"
export DISCORD_GUILD_ID="your_guild_id"
export TELEGRAM_BOT_TOKEN="your_telegram_token"
export API_ADDR=":8080"
export API_URL="https://your-domain.com"
```

## Next Steps

- Read the [full documentation](account-linking.md)
- Set up monitoring and logging
- Implement database persistence
- Add rate limiting and security enhancements
- Create admin dashboard
- Set up automated testing

## Support

For issues and questions:
- Check the [troubleshooting section](account-linking.md#troubleshooting)
- Review the [full documentation](account-linking.md)
- Contact the G-Swarm team in Discord 