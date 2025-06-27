#!/bin/bash

# Test script for Telegram bot with local API

set -e

echo "🔧 G-Swarm Telegram Bot Test Script"
echo "==================================="

# Check if telegram-config.json exists
if [ ! -f "telegram-config.json" ]; then
    echo "❌ telegram-config.json not found"
    echo "   Run the Telegram bot first to generate the config"
    exit 1
fi

echo "✅ Found telegram-config.json"
echo ""

echo "🔨 Building Telegram bot..."
make build-telegram

echo ""
echo "🚀 Starting Telegram bot..."
echo "   The bot will auto-generate API key if not present"
echo "   API URL is hardcoded to your backend"
echo ""
echo "📋 Test Instructions:"
echo "   1. Start the bot and provide your EOA address"
echo "   2. Use /verify command in Telegram"
echo "   3. Copy the verification code"
echo "   4. Use the code in Discord with /verify"
echo ""
echo "Press Ctrl+C to stop the bot"
echo ""

# Run the Telegram bot
./build/gswarm 