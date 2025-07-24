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
	config     *Config
	session    *discordgo.Session
	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *http.Client
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

	// Create HTTP client for API calls
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	bot := &Bot{
		config:     config,
		session:    session,
		ctx:        ctx,
		cancel:     cancel,
		httpClient: httpClient,
	}

	// Set up event handlers
	session.AddHandler(bot.handleReady)
	session.AddHandler(bot.handleInteractionCreate)

	// Start role assignment checker
	go bot.startRoleAssignmentChecker()

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

	// Note: Role will be assigned after successful Telegram verification
	log.Printf("User %s will receive role after completing Telegram verification", discordID)

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
		"3. Your accounts will be linked automatically\n"+
		"4. You'll receive your GSwarm role automatically\n\n"+
		"⚠️ **Important:**\n"+
		"• This code expires in 10 minutes\n"+
		"• Keep it private and don't share it\n"+
		"• Each code can only be used once\n"+
		"• Role assignment happens after Telegram verification", code, code)

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

	// // Check if the configured role exists
	// role, err := b.session.State.Role(guildCfg.ID, guildCfg.RoleID)
	// if err != nil || role == nil {
	// 	log.Printf("Role %s not found in guild %s - role must be created manually by server admin", guildCfg.RoleID, guildCfg.ID)
	// 	return fmt.Errorf("configured role %s does not exist in guild %s - please create the role manually", guildCfg.RoleID, guildCfg.ID)
	// }

	// log.Printf("Found configured role: %s (%s)", role.Name, role.ID)

	// Double-check if user already has the role before assigning
	member, err := b.session.GuildMember(guildCfg.ID, discordID)
	if err != nil {
		log.Printf("Failed to get guild member for user %s: %v", discordID, err)
		// Check if it's a "user not in guild" error
		if strings.Contains(err.Error(), "10007") || strings.Contains(err.Error(), "Unknown Member") {
			log.Printf("User %s is not a member of guild %s, skipping role assignment", discordID, guildCfg.ID)
			return fmt.Errorf("user %s is not a member of guild %s", discordID, guildCfg.ID)
		}
		if strings.Contains(err.Error(), "10013") || strings.Contains(err.Error(), "Unknown User") {
			log.Printf("User %s does not exist, skipping role assignment", discordID)
			return fmt.Errorf("user %s does not exist", discordID)
		}
		return fmt.Errorf("failed to get guild member: %w", err)
	}

	for _, memberRoleID := range member.Roles {
		if memberRoleID == guildCfg.RoleID {
			log.Printf("User %s already has role %s, skipping assignment", discordID, guildCfg.RoleID)
			return nil
		}
	}

	log.Printf("User %s does not have role %s, proceeding with assignment", discordID, guildCfg.RoleID)

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

	// Update role assignment timestamp in API
	if err := b.updateRoleAssignmentTimestamp(discordID, true); err != nil {
		log.Printf("Warning: Failed to update role assignment timestamp for user %s: %v", discordID, err)
		// Don't return error here as role was successfully assigned
	}

	return nil
}

// startRoleAssignmentChecker periodically checks for pending role assignments
func (b *Bot) startRoleAssignmentChecker() {
	ticker := time.NewTicker(15 * time.Second) // Check every 15 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.checkPendingRoleAssignments()
		case <-b.ctx.Done():
			return
		}
	}
}

// checkPendingRoleAssignments checks for users who need role assignment
func (b *Bot) checkPendingRoleAssignments() {
	// Get pending role assignments from API
	pendingUsers, err := b.getPendingRoleAssignments()
	if err != nil {
		log.Printf("Failed to get pending role assignments: %v", err)
		return
	}

	if len(pendingUsers) == 0 {
		return // No pending assignments
	}

	log.Printf("Found %d pending role assignments", len(pendingUsers))

	// Assign roles to each pending user
	for _, discordID := range pendingUsers {
		log.Printf("Processing pending role assignment for user %s", discordID)

		// Find which guild this user belongs to
		for _, guildCfg := range b.config.Guilds {
			log.Printf("Attempting to assign role %s to user %s in guild %s", guildCfg.RoleID, discordID, guildCfg.ID)

			// Try to assign role in this guild
			if err := b.assignGSwarmRole(discordID, &guildCfg); err != nil {
				log.Printf("Failed to assign role to user %s in guild %s: %v", discordID, guildCfg.ID, err)

				// Update role assignment timestamp with failure
				if updateErr := b.updateRoleAssignmentTimestamp(discordID, false); updateErr != nil {
					log.Printf("Warning: Failed to update role assignment timestamp for failed assignment user %s: %v", discordID, updateErr)
				}

				// Continue with other users
			} else {
				log.Printf("Successfully assigned role to user %s in guild %s", discordID, guildCfg.ID)
				break // Role assigned successfully, move to next user
			}
		}
	}
}

// getPendingRoleAssignments fetches pending role assignments from the API
func (b *Bot) getPendingRoleAssignments() ([]string, error) {
	url := fmt.Sprintf("%s/telegrams/pending-role-assignments", b.config.APIURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+b.config.APISecret)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	// Read the response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("API Response: %s", string(body))

	// Try to parse the response with the actual API format
	var response struct {
		Success            bool `json:"success"`
		PendingAssignments []struct {
			TelegramID     interface{} `json:"telegram_id"` // Can be number or string
			DiscordID      interface{} `json:"discord_id"`  // Can be number or string
			VerifiedAt     string      `json:"verified_at"`
			RoleAssignedAt *string     `json:"role_assigned_at"`
		} `json:"pendingAssignments"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Log what we found
	log.Printf("Parsed response - Success: %v, Count: %d, PendingAssignments: %d", response.Success, response.Count, len(response.PendingAssignments))

	// Extract Discord IDs from pendingAssignments
	var discordIDs []string
	for _, assignment := range response.PendingAssignments {
		if assignment.RoleAssignedAt == nil { // Only include users who haven't had roles assigned
			// Convert DiscordID to string properly (handle large numbers)
			var discordIDStr string
			switch v := assignment.DiscordID.(type) {
			case float64:
				discordIDStr = fmt.Sprintf("%.0f", v) // Use %.0f to avoid scientific notation
			case int64:
				discordIDStr = fmt.Sprintf("%d", v)
			case string:
				discordIDStr = v
			default:
				discordIDStr = fmt.Sprintf("%v", v)
			}

			discordIDs = append(discordIDs, discordIDStr)
			log.Printf("Found pending user: %s (Telegram: %v)", discordIDStr, assignment.TelegramID)
		}
	}

	log.Printf("Extracted %d Discord IDs for role assignment", len(discordIDs))
	return discordIDs, nil
}

// updateRoleAssignmentTimestamp updates the role assignment timestamp in the API
func (b *Bot) updateRoleAssignmentTimestamp(discordID string, success bool) error {
	// Create the request payload
	request := map[string]interface{}{
		"discord_id": discordID,
		"success":    success,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/telegrams/update-role-assignment", b.config.APIURL)
	log.Printf("Updating role assignment timestamp for user %s: %s", discordID, url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.config.APISecret)

	// Make the request
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	if !apiResp.Success {
		return fmt.Errorf("API returned error: %s", apiResp.Error)
	}

	log.Printf("Successfully updated role assignment timestamp for user %s: %s", discordID, apiResp.Message)
	return nil
}
