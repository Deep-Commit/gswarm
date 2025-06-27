package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Config holds the Discord bot configuration
type Config struct {
	DiscordToken string
	APIURL       string
	APISecret    string
	GuildID      string
	RoleID       string
}

// Bot represents the Discord bot instance
type Bot struct {
	config  *Config
	session *discordgo.Session
	ctx     context.Context
	cancel  context.CancelFunc
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
	case "verify":
		b.handleVerifyCommand(s, i)
	default:
		b.handleUnknownCommand(s, i)
	}
}

// registerCommands registers slash commands with Discord
func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "verify",
			Description: "Verify your G-Swarm node operator status",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "code",
					Description: "Your verification code from the Telegram bot",
					Required:    true,
				},
			},
		},
	}

	// Register commands for the guild
	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, b.config.GuildID, cmd)
		if err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.Name, err)
		}
		log.Printf("Registered command: %s", cmd.Name)
	}

	return nil
}

// handleVerifyCommand handles the /verify command
func (b *Bot) handleVerifyCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get the verification code from the interaction
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		b.respondToInteraction(s, i, "❌ No verification code provided. Please provide your verification code.", true)
		return
	}

	code := options[0].StringValue()
	discordID := i.Member.User.ID

	// Call the verification API
	if err := b.verifyCode(code, discordID); err != nil {
		log.Printf("Verification failed for user %s: %v", discordID, err)
		b.respondToInteraction(s, i, "❌ Verification failed. Please check your code and try again.", true)
		return
	}

	// Assign the role
	if err := b.assignRole(discordID); err != nil {
		log.Printf("Failed to assign role to user %s: %v", discordID, err)
		b.respondToInteraction(s, i, "⚠️ Verification successful but failed to assign role. Please contact an administrator.", true)
		return
	}

	b.respondToInteraction(s, i, "✅ You have been successfully verified! Welcome to the G-Swarm community!", false)
}

// handleUnknownCommand handles unknown commands
func (b *Bot) handleUnknownCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.respondToInteraction(s, i, "❓ Unknown command. Use `/verify <code>` to verify your status.", true)
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

// DiscordLinkRequest represents the request to link Discord with verification code
type DiscordLinkRequest struct {
	DiscordID string `json:"discordId"`
	Code      string `json:"code"`
}

// DiscordLinkResponse represents the response from the Discord link API
type DiscordLinkResponse struct {
	RoleID string `json:"roleId,omitempty"`
	Error  string `json:"error,omitempty"`
}

// verifyCode calls the verification API
func (b *Bot) verifyCode(code, discordID string) error {
	// Create the request payload
	request := DiscordLinkRequest{
		DiscordID: discordID,
		Code:      code,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/discord/link", b.config.APIURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-SECRET", b.config.APISecret)

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the raw response for debugging
	log.Printf("API Response (status %d): %s", resp.StatusCode, string(body))

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp DiscordLinkResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	if apiResp.Error != "" {
		return fmt.Errorf("API returned error: %s", apiResp.Error)
	}

	// Success if we got a roleId
	if apiResp.RoleID == "" {
		return fmt.Errorf("API returned success but no roleId")
	}

	log.Printf("Successfully verified code %s for Discord user %s (roleId: %s)", code, discordID, apiResp.RoleID)
	return nil
}

// assignRole assigns the G-Swarm verified role to a user
func (b *Bot) assignRole(discordID string) error {
	// Create or get a purple-colored verification role
	purpleRoleID, err := b.ensurePurpleVerificationRole()
	if err != nil {
		log.Printf("Warning: Failed to create/get purple verification role: %v", err)
		log.Printf("Falling back to original role assignment without color change")
		// Fall back to original role assignment
		err = b.session.GuildMemberRoleAdd(b.config.GuildID, discordID, b.config.RoleID)
		if err != nil {
			return fmt.Errorf("failed to assign role: %w", err)
		}
		log.Printf("Assigned original role to user %s", discordID)
		return nil
	}

	// Assign the purple verification role to the user
	err = b.session.GuildMemberRoleAdd(b.config.GuildID, discordID, purpleRoleID)
	if err != nil {
		return fmt.Errorf("failed to assign purple verification role: %w", err)
	}

	log.Printf("Assigned purple verification role to user %s", discordID)
	return nil
}

// ensurePurpleVerificationRole creates or gets a purple-colored verification role
func (b *Bot) ensurePurpleVerificationRole() (string, error) {
	// Purple color in Discord format
	purpleColor := 0x9370DB // Purple in hex = 9641179 in decimal

	// Role name for the purple verification role
	roleName := "GSwarm"

	// First, try to find an existing purple verification role
	roles, err := b.session.GuildRoles(b.config.GuildID)
	if err != nil {
		return "", fmt.Errorf("failed to get guild roles: %w", err)
	}

	// Look for existing purple verification role
	for _, role := range roles {
		if role.Name == roleName {
			log.Printf("Found existing purple verification role: %s (ID: %s)", role.Name, role.ID)
			return role.ID, nil
		}
	}

	// Create new purple verification role (cosmetic only, no permissions)
	log.Printf("Creating new purple verification role: %s", roleName)

	hoist := true
	mentionable := true
	permissions := int64(0) // No permissions - purely cosmetic

	roleParams := &discordgo.RoleParams{
		Name:        roleName,
		Color:       &purpleColor,
		Hoist:       &hoist,       // Display role members separately
		Mentionable: &mentionable, // Allow the role to be mentioned
		Permissions: &permissions, // No special permissions
	}

	role, err := b.session.GuildRoleCreate(b.config.GuildID, roleParams)
	if err != nil {
		return "", fmt.Errorf("failed to create purple verification role: %w", err)
	}

	log.Printf("Created purple verification role: %s (ID: %s) - Cosmetic only", role.Name, role.ID)
	return role.ID, nil
}

// checkBotPermissions checks if the bot has necessary permissions
func (b *Bot) checkBotPermissions() error {
	// Get the bot's user ID
	botUser, err := b.session.User("@me")
	if err != nil {
		return fmt.Errorf("failed to get bot user: %w", err)
	}

	// Get the guild member info for the bot
	member, err := b.session.GuildMember(b.config.GuildID, botUser.ID)
	if err != nil {
		return fmt.Errorf("failed to get bot member info: %w", err)
	}

	log.Printf("Bot permissions check - Bot ID: %s, Guild ID: %s", botUser.ID, b.config.GuildID)
	log.Printf("Bot roles: %v", member.Roles)

	return nil
}
