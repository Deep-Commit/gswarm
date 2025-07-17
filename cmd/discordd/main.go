package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"io/ioutil"

	"github.com/Deep-Commit/gswarm/internal/discord"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// Version information
var (
	Version   = "1.0.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Load .env file if it exists
	if err := loadEnvFile(); err != nil {
		log.Printf("Warning: Failed to load .env file: %v", err)
	}

	app := createCLIApp()
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// loadEnvFile loads environment variables from .env file
func loadEnvFile() error {
	envFile := ".env"
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf(".env file not found")
	}

	content, err := os.ReadFile(envFile)
	if err != nil {
		return fmt.Errorf("failed to read .env file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove quotes if present
			if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"') {
				value = value[1 : len(value)-1]
			}
			os.Setenv(key, value)
		}
	}

	return nil
}

func createCLIApp() *cli.App {
	app := &cli.App{
		Name:        "discordd",
		Usage:       "G-Swarm Discord Account Linking Bot",
		Description: getAppDescription(),
		Version:     Version,
		Before:      getBeforeFunc(),
		Action:      getMainAction(),
		Flags:       getAppFlags(),
		Commands:    getAppCommands(),
	}
	return app
}

func getAppDescription() string {
	return `G-Swarm Discord Account Linking Bot

A Discord bot that handles account linking for G-Swarm users.
This bot generates linking codes to connect Discord and Telegram accounts.

Features:
• /link-telegram command for account linking
• Automatic GSwarm role assignment
• Integration with G-Swarm API
• Secure code-based linking system

Account Linking:
• Users can link their Discord and Telegram accounts
• Secure code-based verification system
• Integration with external API endpoints
• Automatic role assignment for verified users

This is a community project and is not affiliated with the official Gensyn team.`
}

func getAppFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "discord-token",
			Usage:    "Discord bot token",
			Required: false,
			EnvVars:  []string{"DISCORD_BOT_TOKEN"},
		},
		&cli.StringFlag{
			Name:    "api-url",
			Usage:   "External API URL for account linking (default: https://gswarm.dev/api)",
			Value:   "https://gswarm.dev/api",
			EnvVars: []string{"GSWARM_API_URL"},
		},
		&cli.StringFlag{
			Name:     "api-secret",
			Usage:    "API secret key for authentication",
			Required: false,
			EnvVars:  []string{"GSWARM_API_SECRET"},
		},
		// Remove guild-id and role-id flags
	}
}

func getMainAction() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		// Only run the bot if no subcommand is present
		if c.Args().Len() == 0 {
			// Remove strict CLI check for discord-token
			return runDiscordBot(c)
		}
		// If a subcommand is present, do nothing (let the subcommand run)
		return nil
	}
}

func getAppCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Show detailed version information",
			Action:  getVersionAction(),
		},
	}
}

func getVersionAction() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		fmt.Printf("Discord Bot version %s\n", Version)
		fmt.Printf("Build date: %s\n", BuildDate)
		fmt.Printf("Git commit: %s\n", GitCommit)
		return nil
	}
}

func getBeforeFunc() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		// Set up custom help template
		cli.AppHelpTemplate = getHelpTemplate()
		return nil
	}
}

func getHelpTemplate() string {
	return `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} \
   {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
   {{if len .Authors}}
AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author.Name}}{{if $author.Email}} <{{$author.Email}}>{{end}}{{end}}
   {{end}}{{if .Commands}}
COMMANDS:{{range .CommandCategories}}
   {{.Name}}:{{range .Commands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
GLOBAL OPTIONS:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}{{if .Version}}
VERSION:
   {{.Version}}
   {{end}}
EXAMPLES:
   # Start Discord bot with environment variables
   discordd --discord-token YOUR_TOKEN --api-secret YOUR_SECRET --guild-id YOUR_GUILD --role-id YOUR_ROLE

   # Show version
   discordd version

ENVIRONMENT VARIABLES:
   All flags can be set via environment variables:
   - DISCORD_BOT_TOKEN: Discord bot token
   - GSWARM_API_URL: G-Swarm API URL (default: https://gswarm.dev/api)
   - GSWARM_API_SECRET: G-Swarm API secret key
   - DISCORD_GUILD_ID: Discord guild (server) ID
   - DISCORD_ROLE_ID: Discord role ID to assign to verified users (optional)

FEATURES:
   • /link-telegram: Generate linking codes for Discord-Telegram account linking
   • Automatic role assignment: Assigns GSwarm role when users use /link-telegram
   • Secure API integration: Uses external G-Swarm API for account linking

LEARN MORE:
   • GitHub: https://github.com/Deep-Commit/gswarm
   • Documentation: https://github.com/Deep-Commit/gswarm#readme
`
}

func runDiscordBot(c *cli.Context) error {
	// Load Discord config from YAML file (config.yaml)
	configPath := "config.yaml"
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config.yaml: %w", err)
	}
	var yamlConfig struct {
		Discord discord.Config `yaml:"discord"`
	}
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return fmt.Errorf("failed to parse discord config: %w", err)
	}
	// Override token, API URL, and secret from CLI/env if provided
	if c.String("discord-token") != "" {
		yamlConfig.Discord.DiscordToken = c.String("discord-token")
	}
	if c.String("api-url") != "" {
		yamlConfig.Discord.APIURL = c.String("api-url")
	}
	if c.String("api-secret") != "" {
		yamlConfig.Discord.APISecret = c.String("api-secret")
	}

	if yamlConfig.Discord.DiscordToken == "" {
		return fmt.Errorf("discord-token is required (set in config.yaml or via CLI/env)")
	}

	bot, err := discord.NewBot(&yamlConfig.Discord)
	if err != nil {
		return fmt.Errorf("failed to create Discord bot: %w", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Starting Discord bot...")
	if err := bot.Start(); err != nil {
		return fmt.Errorf("failed to start Discord bot: %w", err)
	}

	// Wait for interrupt signal
	<-sigChan
	fmt.Println("\nReceived interrupt signal. Shutting down...")

	if err := bot.Stop(); err != nil {
		return fmt.Errorf("failed to stop Discord bot: %w", err)
	}

	return nil
}
