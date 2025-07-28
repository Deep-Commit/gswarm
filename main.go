package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Deep-Commit/gswarm/internal/telegram"
)

func main() {
	// Parse command line flags for Telegram service
	var configPath = flag.String("config", "", "Telegram config file path")
	var forceUpdate = flag.Bool("force-update", false, "Force Telegram config update")
	var eoaAddress = flag.String("eoa", "", "EOA address for monitoring")
	var botToken = flag.String("bot-token", "", "Telegram bot token")
	var chatID = flag.String("chat-id", "", "Telegram chat ID")

	flag.Parse()

	fmt.Println("Starting G-Swarm Telegram Monitor...")

	// Create Telegram service
	service := telegram.NewTelegramService(*configPath, *forceUpdate, *eoaAddress, *botToken, *chatID)

	// Run the Telegram service
	if err := service.Run(); err != nil {
		fmt.Printf("Error running Telegram service: %v\n", err)
		os.Exit(1)
	}
}
