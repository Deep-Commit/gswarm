package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Config holds the Discord bot configuration
type GuildConfig struct {
	ID     string `yaml:"id"`
	RoleID string `yaml:"role_id"`
}

type Config struct {
	DiscordToken string `yaml:"token"`
	APIURL       string
	APISecret    string
	Guilds       []GuildConfig `yaml:"guilds"`
}

// Bot represents the Discord bot instance
type Bot struct {
	config  *Config
	session *discordgo.Session
	ctx     context.Context
	cancel  context.CancelFunc
}

// Helper to get GuildConfig by guild ID
func (b *Bot) getGuildConfig(guildID string) *GuildConfig {
	for _, g := range b.config.Guilds {
		if g.ID == guildID {
			return &g
		}
	}
	return nil
}

// NewBot creates a new Discord bot instance
func NewBot(config *Config) (*Bot, error) {
	// Create Discord session
	session, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Set up bot intents
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMembers

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	bot := &Bot{
		config:  config,
		session: session,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Set up event handlers
	session.AddHandler(bot.handleReady)
	session.AddHandler(bot.handleInteractionCreate)

	return bot, nil
}

// Start starts the Discord bot
func (b *Bot) Start() error {
	// Open Discord connection
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	// Register slash commands
	if err := b.registerCommands(); err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	log.Println("Discord bot started successfully!")
	return nil
}

// Stop stops the Discord bot
func (b *Bot) Stop() error {
	b.cancel()

	if b.session != nil {
		if err := b.session.Close(); err != nil {
			return fmt.Errorf("failed to close Discord session: %w", err)
		}
	}

	log.Println("Discord bot stopped successfully!")
	return nil
}

// handleReady handles the ready event
func (b *Bot) handleReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Bot is ready! Logged in as: %s#%s", event.User.Username, event.User.Discriminator)
}

// handleInteractionCreate handles slash command interactions
func (b *Bot) handleInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "link-telegram":
		b.handleLinkTelegramCommand(s, i)
	default:
		b.handleUnknownCommand(s, i)
	}
}

// registerCommands registers slash commands with Discord
func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "link-telegram",
			Description: "Generate a code to link your Discord account with Telegram",
		},
	}

	// Register commands for all configured guilds
	for _, guild := range b.config.Guilds {
		for _, cmd := range commands {
			_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, guild.ID, cmd)
			if err != nil {
				return fmt.Errorf("failed to register command %s for guild %s: %w", cmd.Name, guild.ID, err)
			}
			log.Printf("Registered command: %s for guild: %s", cmd.Name, guild.ID)
		}
	}

	return nil
}

// handleUnknownCommand handles unknown commands
func (b *Bot) handleUnknownCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.respondToInteraction(s, i, "❓ Unknown command. Use `/link-telegram` to link your Discord and Telegram accounts.", true)
}

// respondToInteraction responds to a Discord interaction
func (b *Bot) respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) {
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	}

	if ephemeral {
		response.Data.Flags = discordgo.MessageFlagsEphemeral
	}

	if err := s.InteractionRespond(i.Interaction, response); err != nil {
		log.Printf("Failed to respond to interaction: %v", err)
	}
}

// handleLinkTelegramCommand handles the /link-telegram command
func (b *Bot) handleLinkTelegramCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	discordID := i.Member.User.ID
	guildID := i.GuildID
	channelID := i.ChannelID

	log.Printf("Command received from Guild: %s, Channel: %s, User: %s", guildID, channelID, discordID)

	guildCfg := b.getGuildConfig(guildID)
	if guildCfg == nil {
		log.Printf("No config found for guild %s", guildID)
		b.respondToInteraction(s, i, "❌ This server is not configured for account linking.", true)
		return
	}

	// Check if user already has the role
	hasRole := false
	log.Printf("Checking roles for user %s in guild %s", discordID, guildID)
	log.Printf("User roles: %v", i.Member.Roles)
	log.Printf("Target role ID: %s", guildCfg.RoleID)

	for _, roleID := range i.Member.Roles {
		if roleID == guildCfg.RoleID {
			hasRole = true
			log.Printf("User already has role %s", roleID)
			break
		}
	}

	// Assign the role if they don't have it already
	if !hasRole {
		log.Printf("User does not have role %s, attempting to assign", guildCfg.RoleID)
		if err := b.assignGSwarmRole(discordID, guildCfg); err != nil {
			log.Printf("Failed to assign role to user %s: %v", discordID, err)
			// Continue with the linking process even if role assignment fails
		}
	} else {
		log.Printf("User already has role %s, skipping assignment", guildCfg.RoleID)
	}

	// Issue a linking code via the API
	code, err := b.issueLinkingCode(discordID)
	if err != nil {
		log.Printf("Failed to issue linking code for user %s: %v", discordID, err)
		b.respondToInteraction(s, i, "❌ Failed to generate linking code. Please try again later.", true)
		return
	}

	// Always send the code as an ephemeral message
	message := fmt.Sprintf("🔗 **Discord-Telegram Account Linking**\n\n"+
		"Here's your linking code: **`%s`**\n\n"+
		"**Instructions:**\n"+
		"1. Open your Telegram bot\n"+
		"2. Send `/verify %s`\n"+
		"3. Your accounts will be linked automatically\n\n"+
		"⚠️ **Important:**\n"+
		"• This code expires in 10 minutes\n"+
		"• Keep it private and don't share it\n"+
		"• Each code can only be used once", code, code)

	b.respondToInteraction(s, i, message, true)
}

// issueLinkingCode calls the API to issue a linking code
func (b *Bot) issueLinkingCode(discordID string) (string, error) {
	// Create the request payload
	request := map[string]interface{}{
		"discordId": discordID,
		// telegramId is optional and can be omitted
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/telegrams/issue-code", b.config.APIURL)
	log.Printf("Making API request to: %s", url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", b.config.APISecret)

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp struct {
		Success bool   `json:"success"`
		Code    string `json:"code,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}

	if !apiResp.Success {
		return "", fmt.Errorf("API returned error: %s", apiResp.Error)
	}

	if apiResp.Code == "" {
		return "", fmt.Errorf("API returned success but no code")
	}

	log.Printf("Successfully issued linking code %s for Discord user %s", apiResp.Code, discordID)
	return apiResp.Code, nil
}

// assignGSwarmRole assigns the GSwarm role to a user
func (b *Bot) assignGSwarmRole(discordID string, guildCfg *GuildConfig) error {
	if guildCfg.RoleID == "" {
		log.Printf("No role ID configured, skipping role assignment for user %s", discordID)
		return nil
	}

	log.Printf("Starting role assignment process for user %s with role ID %s", discordID, guildCfg.RoleID)

	// First, try to get the existing role to check if it exists
	role, err := b.session.State.Role(guildCfg.ID, guildCfg.RoleID)
	if err != nil || role == nil {
		log.Printf("Role %s not found in guild state, attempting to create it", guildCfg.RoleID)

		// Create the role if it doesn't exist
		role, err = b.createGSwarmRole(guildCfg.ID)
		if err != nil {
			log.Printf("Failed to create GSwarm role: %v", err)
			return fmt.Errorf("failed to create GSwarm role: %w", err)
		}

		log.Printf("Created new GSwarm role with ID: %s", role.ID)
		log.Printf("Note: Using configured role ID %s, not the newly created role ID %s", guildCfg.RoleID, role.ID)
	} else {
		log.Printf("Found existing role: %s (%s)", role.Name, role.ID)
	}

	// Double-check if user already has the role before assigning
	member, err := b.session.GuildMember(guildCfg.ID, discordID)
	if err != nil {
		log.Printf("Failed to get guild member for user %s: %v", discordID, err)
		return fmt.Errorf("failed to get guild member: %w", err)
	}

	for _, memberRoleID := range member.Roles {
		if memberRoleID == guildCfg.RoleID {
			log.Printf("User %s already has role %s, skipping assignment", discordID, guildCfg.RoleID)
			return nil
		}
	}

	// Add the role to the user
	log.Printf("Adding role %s to user %s in guild %s", guildCfg.RoleID, discordID, guildCfg.ID)
	err = b.session.GuildMemberRoleAdd(guildCfg.ID, discordID, guildCfg.RoleID)
	if err != nil {
		// Provide more detailed error information
		if strings.Contains(err.Error(), "50001") {
			return fmt.Errorf("missing permissions - ensure bot role is higher than target role in hierarchy: %w", err)
		}
		if strings.Contains(err.Error(), "50013") {
			return fmt.Errorf("missing permissions - ensure bot has 'Manage Roles' permission: %w", err)
		}
		return fmt.Errorf("failed to assign GSwarm role to user %s: %w", discordID, err)
	}

	log.Printf("Successfully assigned GSwarm role to user %s in guild %s", discordID, guildCfg.ID)
	return nil
}

// createGSwarmRole creates a new GSwarm role with purple color
func (b *Bot) createGSwarmRole(guildID string) (*discordgo.Role, error) {
	// Purple color (RGB: 128, 0, 128)
	purpleColor := 8388736
	hoist := false
	permissions := int64(0)
	mentionable := false

	role, err := b.session.GuildRoleCreate(guildID, &discordgo.RoleParams{
		Name:        "GSwarm",
		Color:       &purpleColor,
		Hoist:       &hoist,       // Don't show members with this role separately
		Permissions: &permissions, // No special permissions
		Mentionable: &mentionable, // Don't allow mentions
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create GSwarm role: %w", err)
	}

	log.Printf("Successfully created GSwarm role: %s (ID: %s)", role.Name, role.ID)
	return role, nil
}
