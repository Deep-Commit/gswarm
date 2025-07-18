# Docker Deployment Guide

This guide explains how to deploy the G-Swarm Discord Bot using Docker and Docker Compose.

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+
- Discord Bot Token and Server Configuration
- G-Swarm API Access

## Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/Deep-Commit/gswarm.git
   cd gswarm
   ```

2. **Set up environment variables**
   ```bash
   cp env.example .env
   # Edit .env with your actual values
   ```

3. **Build and start the Discord bot**
   ```bash
   docker-compose up -d discord-bot
   ```

4. **Check the logs**
   ```bash
   docker-compose logs -f discord-bot
   ```

## Environment Variables

Copy `env.example` to `.env` and configure the following variables:

### Required Variables

- `DISCORD_BOT_TOKEN`: Your Discord bot token from the Discord Developer Portal
- `DISCORD_GUILD_ID`: The ID of your Discord server (guild)
- `DISCORD_ROLE_ID`: The ID of the role to assign to verified users
- `GSWARM_API_SECRET`: Your G-Swarm API secret key

### Optional Variables

- `GSWARM_API_URL`: G-Swarm API URL (default: https://gswarm.dev/api)
- `LOG_LEVEL`: Logging level (default: info)
- `VERSION`: Build version (auto-detected from git)
- `BUILD_DATE`: Build date (auto-detected)
- `GIT_COMMIT`: Git commit hash (auto-detected)

## Docker Compose Services

### Discord Bot Service

The main Discord bot service with the following features:

- **Multi-stage build**: Optimized for size and security
- **Non-root user**: Runs as `gswarm` user for security
- **Health checks**: Monitors bot process health
- **Volume mounts**: Persistent logs and configuration
- **Restart policy**: Automatically restarts on failure

### Redis Service (Optional)

Included as an optional caching layer for future features:

```bash
# Start with Redis
docker-compose --profile cache up -d
```

## Commands

### Basic Operations

```bash
# Start the Discord bot
docker-compose up -d discord-bot

# Stop the Discord bot
docker-compose down

# View logs
docker-compose logs -f discord-bot

# Restart the bot
docker-compose restart discord-bot

# Check status
docker-compose ps
```

### Building

```bash
# Build the image
docker-compose build discord-bot

# Build with custom version
VERSION=2.0.0 docker-compose build discord-bot

# Force rebuild
docker-compose build --no-cache discord-bot
```

### Development

```bash
# Run in development mode with logs
docker-compose up discord-bot

# Run with custom environment
docker-compose run --rm discord-bot ./discordd --help
```

## Configuration

### Discord Bot Setup

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application
3. Add a bot to your application
4. Copy the bot token to `DISCORD_BOT_TOKEN`
5. Enable required intents (Server Members Intent)
6. **Important**: Ensure the bot has the following permissions:
   - **Manage Roles**: Required to update role colors and assign roles
   - **Send Messages**: Required to respond to slash commands
   - **Use Slash Commands**: Required for the `/verify` command
7. Invite the bot to your server with appropriate permissions

**Note**: The bot will only assign the role specified in the configuration. The role must be created manually by a server administrator before the bot can assign it to users. The bot will not create any roles automatically.

### G-Swarm API Setup

1. Obtain your G-Swarm API credentials
2. Set `GSWARM_API_SECRET` in your `.env` file
3. Optionally customize `GSWARM_API_URL` if using a different endpoint

## Monitoring

### Health Checks

The container includes health checks that monitor the bot process:

```bash
# Check health status
docker-compose ps

# View health check logs
docker inspect gswarm-discord-bot | grep -A 10 Health
```

### Logs

Logs are available through Docker Compose:

```bash
# Follow logs in real-time
docker-compose logs -f discord-bot

# View recent logs
docker-compose logs --tail=100 discord-bot

# View logs with timestamps
docker-compose logs -t discord-bot
```

### Metrics

The bot exposes basic metrics through the process monitoring:

```bash
# Check resource usage
docker stats gswarm-discord-bot

# View container details
docker inspect gswarm-discord-bot
```

## Troubleshooting

### Common Issues

1. **Bot not connecting**
   - Check `DISCORD_BOT_TOKEN` is correct
   - Verify bot has proper permissions
   - Check Discord server status

2. **API connection issues**
   - Verify `GSWARM_API_SECRET` is correct
   - Check network connectivity
   - Validate API endpoint URL

3. **Permission denied errors**
   - Ensure bot has required Discord permissions
   - Check role assignment permissions
   - Verify guild ID is correct

4. **Role assignment fails**
   - Ensure bot has "Manage Roles" permission to assign the configured role
   - Check that the configured role exists in the server
   - Verify the bot's role is positioned high enough in the hierarchy to assign the target role
   - Check bot logs for permission errors during role assignment

### Debug Mode

Run with debug logging:

```