package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Deep-Commit/gswarm/internal/api"
	"github.com/Deep-Commit/gswarm/internal/discord"
	"github.com/Deep-Commit/gswarm/internal/telegram"
)

func main() {
	// Parse command line flags
	var (
		apiAddr       = flag.String("api-addr", ":8080", "API server address")
		apiKey        = flag.String("api-key", "", "API key for secure endpoints")
		discordToken  = flag.String("discord-token", "", "Discord bot token")
		discordGuild  = flag.String("discord-guild", "", "Discord guild ID")
		discordRole   = flag.String("discord-role", "", "Discord role ID")
		telegramToken = flag.String("telegram-token", "", "Telegram bot token")
		telegramChat  = flag.String("telegram-chat", "", "Telegram chat ID")
		apiURL        = flag.String("api-url", "http://localhost:8080", "API URL for bots")
	)
	flag.Parse()

	// Validate required flags
	if *apiKey == "" {
		log.Fatal("API key is required")
	}
	if *discordToken == "" {
		log.Fatal("Discord token is required")
	}
	if *discordGuild == "" {
		log.Fatal("Discord guild ID is required")
	}
	if *telegramToken == "" {
		log.Fatal("Telegram token is required")
	}

	// Create API server
	apiServer := api.NewAPI(*apiKey)

	// Create Discord bot
	discordConfig := &discord.Config{
		DiscordToken: *discordToken,
		APIURL:       *apiURL,
		APISecret:    *apiKey,
		Guilds: []discord.GuildConfig{
			{
				ID:     *discordGuild,
				RoleID: *discordRole,
			},
		},
	}

	discordBot, err := discord.NewBot(discordConfig)
	if err != nil {
		log.Fatalf("Failed to create Discord bot: %v", err)
	}

	// Create Telegram service
	telegramConfig := &telegram.TelegramConfig{
		BotToken: *telegramToken,
		ChatID:   *telegramChat,
		APIURL:   *apiURL,
	}

	telegramService := telegram.NewTelegramService("", false, "", "", "")
	telegramService.Config = telegramConfig

	// Start API server in a goroutine
	go func() {
		log.Printf("Starting API server on %s", *apiAddr)
		if err := apiServer.Start(*apiAddr); err != nil {
			log.Fatalf("Failed to start API server: %v", err)
		}
	}()

	// Start Discord bot in a goroutine
	go func() {
		log.Println("Starting Discord bot...")
		if err := discordBot.Start(); err != nil {
			log.Fatalf("Failed to start Discord bot: %v", err)
		}
	}()

	// Start Telegram service in a goroutine
	go func() {
		log.Println("Starting Telegram service...")
		if err := telegramService.Run(); err != nil {
			log.Fatalf("Failed to start Telegram service: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	// Stop Discord bot
	if err := discordBot.Stop(); err != nil {
		log.Printf("Error stopping Discord bot: %v", err)
	}

	// Stop Telegram service
	close(telegramService.StopChan)

	log.Println("Server stopped successfully")
}
