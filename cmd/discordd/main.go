package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Deep-Commit/gswarm/internal/discord"
	"github.com/urfave/cli/v2"
)

// Version information
var (
	Version   = "1.0.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

func main() {
	app := createCLIApp()
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func createCLIApp() *cli.App {
	app := &cli.App{
		Name:        "discordd",
		Usage:       "G-Swarm Discord Verification Bot",
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
	return `G-Swarm Discord Verification Bot

A Discord bot that handles verification for G-Swarm node operators.
This bot grants the @GSwarm role to operators who prove ownership
of their node through a one-time verification code.

Features:
• /verify command for code-based verification
• Automatic role assignment
• Integration with G-Swarm API
• Secure verification flow

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
			Usage:   "G-Swarm API URL",
			Value:   "https://gswarm.dev/api",
			EnvVars: []string{"GSWARM_API_URL"},
		},
		&cli.StringFlag{
			Name:     "api-secret",
			Usage:    "G-Swarm API secret key",
			Required: false,
			EnvVars:  []string{"GSWARM_API_SECRET"},
		},
		&cli.StringFlag{
			Name:     "guild-id",
			Usage:    "Discord guild (server) ID",
			Required: false,
			EnvVars:  []string{"DISCORD_GUILD_ID"},
		},
		&cli.StringFlag{
			Name:     "role-id",
			Usage:    "Discord role ID for @GSwarm",
			Required: false,
			EnvVars:  []string{"DISCORD_ROLE_ID"},
		},
	}
}

func getMainAction() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		// Only run the bot if no subcommand is present
		if c.Args().Len() == 0 {
			if c.String("discord-token") == "" {
				return fmt.Errorf("discord-token is required")
			}
			if c.String("api-secret") == "" {
				return fmt.Errorf("api-secret is required")
			}
			if c.String("guild-id") == "" {
				return fmt.Errorf("guild-id is required")
			}
			if c.String("role-id") == "" {
				return fmt.Errorf("role-id is required")
			}
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
   - DISCORD_ROLE_ID: Discord role ID for @GSwarm

LEARN MORE:
   • GitHub: https://github.com/Deep-Commit/gswarm
   • Documentation: https://github.com/Deep-Commit/gswarm#readme
`
}

func runDiscordBot(c *cli.Context) error {
	config := &discord.Config{
		DiscordToken: c.String("discord-token"),
		APIURL:       c.String("api-url"),
		APISecret:    c.String("api-secret"),
		GuildID:      c.String("guild-id"),
		RoleID:       c.String("role-id"),
	}

	bot, err := discord.NewBot(config)
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
