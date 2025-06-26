package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/Deep-Commit/gswarm/internal/telegram"
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
		Name:        "gswarm",
		Usage:       "G-Swarm Telegram Monitoring Service",
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
	return `G-Swarm Telegram Monitoring Service

A dedicated Telegram bot for monitoring Gensyn AI node activity and blockchain data.
This service tracks votes, rewards, and balance changes for your EOA address and
associated peer IDs, sending notifications only when changes are detected.

Features:
• Real-time blockchain monitoring
• Vote and reward tracking
• Balance change notifications
• Peer ID monitoring
• Secure local configuration
• Change detection (notifications only on changes)

This is a community project and is not affiliated with the official Gensyn team.`
}

func getAppFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "telegram-config-path",
			Usage:   "Path to telegram-config.json file for Telegram integration",
			EnvVars: []string{"GSWARM_TELEGRAM_CONFIG_PATH"},
		},
		&cli.BoolFlag{
			Name:    "update-telegram-config",
			Usage:   "Force update of Telegram config via CLI prompts",
			EnvVars: []string{"GSWARM_UPDATE_TELEGRAM_CONFIG"},
		},
	}
}

func getMainAction() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		return runTelegramService(c)
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
		fmt.Printf("GSwarm version %s\n", Version)
		fmt.Printf("Build date: %s\n", BuildDate)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
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
   # Start Telegram monitoring service
   gswarm

   # Use custom config path
   gswarm --telegram-config-path /path/to/config.json

   # Force update Telegram config
   gswarm --update-telegram-config

   # Show version
   gswarm version

ENVIRONMENT VARIABLES:
   All flags can be set via environment variables with the GSWARM_ prefix.
   For example: GSWARM_TELEGRAM_CONFIG_PATH=/path/to/config.json

LEARN MORE:
   • GitHub: https://github.com/Deep-Commit/gswarm
   • Documentation: https://github.com/Deep-Commit/gswarm#readme
   • Community: https://github.com/Deep-Commit/gswarm/discussions
`
}

func runTelegramService(c *cli.Context) error {
	telegramConfigPath := c.String("telegram-config-path")
	updateTelegramConfig := c.Bool("update-telegram-config")

	telegramService := telegram.NewTelegramService(telegramConfigPath, updateTelegramConfig)
	return telegramService.Run()
}
