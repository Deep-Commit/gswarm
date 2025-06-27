#!/bin/bash

# Test script for Discord bot with local API

set -e

echo "🔧 G-Swarm Discord Bot Test Script"
echo "=================================="

# Check if required environment variables are set
if [ -z "$DISCORD_BOT_TOKEN" ]; then
    echo "❌ DISCORD_BOT_TOKEN environment variable is required"
    echo "   export DISCORD_BOT_TOKEN=your_discord_bot_token"
    exit 1
fi

if [ -z "$GSWARM_API_SECRET" ]; then
    echo "❌ GSWARM_API_SECRET environment variable is required"
    echo "   export GSWARM_API_SECRET=your_api_secret"
    exit 1
fi

if [ -z "$DISCORD_GUILD_ID" ]; then
    echo "❌ DISCORD_GUILD_ID environment variable is required"
    echo "   export DISCORD_GUILD_ID=your_guild_id"
    exit 1
fi

if [ -z "$DISCORD_ROLE_ID" ]; then
    echo "❌ DISCORD_ROLE_ID environment variable is required"
    echo "   export DISCORD_ROLE_ID=your_role_id"
    exit 1
fi

# Set default API URL to gswarm.dev if not provided
if [ -z "$GSWARM_API_URL" ]; then
    export GSWARM_API_URL="https://gswarm.dev/api"
    echo "ℹ️  Using default API URL: $GSWARM_API_URL"
fi

echo "✅ Environment variables configured"
echo ""

echo "🔨 Building Discord bot..."
make build-discord

echo ""
echo "🚀 Starting Discord bot..."
echo "   API URL: $GSWARM_API_URL"
echo "   Guild ID: $DISCORD_GUILD_ID"
echo "   Role ID: $DISCORD_ROLE_ID"
echo ""
echo "📋 Test Instructions:"
echo "   1. Join your Discord server"
echo "   2. Use the /verify command with a code from Telegram bot"
echo "   3. Check the logs for API calls and responses"
echo ""
echo "Press Ctrl+C to stop the bot"
echo ""

# Run the Discord bot
./build/discordd \
    --discord-token "$DISCORD_BOT_TOKEN" \
    --api-url "$GSWARM_API_URL" \
    --api-secret "$GSWARM_API_SECRET" \
    --guild-id "$DISCORD_GUILD_ID" \
    --role-id "$DISCORD_ROLE_ID" 