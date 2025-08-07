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
	"github.com/jackc/pgx/v5/pgxpool"
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
	DatabaseDSN  string        `yaml:"database_dsn"`
}

// Bot represents the Discord bot instance
type Bot struct {
	config     *Config
	session    *discordgo.Session
	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *http.Client
	dbPool     *pgxpool.Pool
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
		Timeout: 30 * time.Second, // Match the timeout used in API calls
	}

	// Connect to database if DSN is provided
	var dbPool *pgxpool.Pool
	if config.DatabaseDSN != "" {
		dbPool, err = pgxpool.New(ctx, config.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		log.Println("Connected to database successfully")
	}

	bot := &Bot{
		config:     config,
		session:    session,
		ctx:        ctx,
		cancel:     cancel,
		httpClient: httpClient,
		dbPool:     dbPool,
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

	if b.dbPool != nil {
		b.dbPool.Close()
		log.Println("Database connection closed")
	}

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
	case "block":
		b.handleBlockCommand(s, i)
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
		{
			Name:        "block",
			Description: "Verify your HFUploadVerified event and get BLOCK role",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "user_address",
					Description: "Your Ethereum wallet address",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "training_id",
					Description: "Your training ID",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "hugging_face_id",
					Description: "Your Hugging Face ID",
					Required:    false,
				},
			},
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

	// Assume user exists (they're in our database) and proceed with role assignment
	log.Printf("Proceeding with role assignment for user %s (assumed to exist)", discordID)

	// Add the role to the user
	log.Printf("Adding role %s to user %s in guild %s", guildCfg.RoleID, discordID, guildCfg.ID)
	err := b.session.GuildMemberRoleAdd(guildCfg.ID, discordID, guildCfg.RoleID)
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

	req.Header.Set("x-api-key", b.config.APISecret)

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
	req.Header.Set("x-api-key", b.config.APISecret)

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

// handleBlockCommand handles the /block command
func (b *Bot) handleBlockCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	discordID := i.Member.User.ID
	guildID := i.GuildID
	channelID := i.ChannelID

	log.Printf("Block command received from Guild: %s, Channel: %s, User: %s", guildID, channelID, discordID)

	// Check if database is connected
	if b.dbPool == nil {
		b.respondToInteraction(s, i, "❌ Database connection not available. Please try again later.", true)
		return
	}

	// Get guild config
	guildCfg := b.getGuildConfig(guildID)
	if guildCfg == nil {
		log.Printf("No config found for guild %s", guildID)
		b.respondToInteraction(s, i, "❌ This server is not configured for block verification.", true)
		return
	}

	// Parse command options
	var userAddress, trainingID, huggingFaceID string
	for _, option := range i.ApplicationCommandData().Options {
		switch option.Name {
		case "user_address":
			userAddress = strings.TrimSpace(option.StringValue())
		case "training_id":
			trainingID = strings.TrimSpace(option.StringValue())
		case "hugging_face_id":
			huggingFaceID = strings.TrimSpace(option.StringValue())
		}
	}

	// Validate required parameters
	if userAddress == "" {
		b.respondToInteraction(s, i, "❌ User address is required. Please provide your Ethereum wallet address.", true)
		return
	}

	if trainingID == "" && huggingFaceID == "" {
		b.respondToInteraction(s, i, "❌ Either training_id or hugging_face_id is required.", true)
		return
	}

	// Input validation and sanitization
	if err := b.validateInputs(userAddress, trainingID, huggingFaceID); err != nil {
		log.Printf("Security: Invalid input detected from user %s: %v", discordID, err)
		b.respondToInteraction(s, i, "❌ Invalid input format. Please check your wallet address and training ID/Hugging Face ID.", true)
		return
	}

	// Check if user is already verified
	var existingVerified bool
	err := b.dbPool.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM verified_users WHERE discord_id = $1)
	`, discordID).Scan(&existingVerified)
	if err != nil {
		log.Printf("Failed to check if user is already verified: %v", err)
		b.respondToInteraction(s, i, "❌ Failed to check verification status. Please try again later.", true)
		return
	}

	if existingVerified {
		b.respondToInteraction(s, i, "✅ You are already verified and have the BLOCK role!", true)
		return
	}

	// Check if HFUploadVerified event exists
	var eventExists bool
	var query string
	var args []interface{}

	if trainingID != "" {
		query = `
			SELECT EXISTS(
				SELECT 1 FROM events 
				WHERE LOWER(user_address) = LOWER($1) AND training_id = $2
			)
		`
		args = []interface{}{userAddress, trainingID}
	} else {
		query = `
			SELECT EXISTS(
				SELECT 1 FROM events 
				WHERE LOWER(user_address) = LOWER($1) AND hugging_face_id = $2
			)
		`
		args = []interface{}{userAddress, huggingFaceID}
	}

	err = b.dbPool.QueryRow(context.Background(), query, args...).Scan(&eventExists)
	if err != nil {
		log.Printf("Failed to check for HFUploadVerified event: %v", err)
		b.respondToInteraction(s, i, "❌ Failed to verify event. Please try again later.", true)
		return
	}

	if !eventExists {
		b.respondToInteraction(s, i, "❌ No HFUploadVerified event found for the provided information. Please check your user address and training ID/Hugging Face ID.", true)
		return
	}

	// Insert verified user record
	_, err = b.dbPool.Exec(context.Background(), `
		INSERT INTO verified_users (discord_id, user_address, training_id, hugging_face_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (discord_id) DO NOTHING
	`, discordID, strings.ToLower(userAddress), trainingID, huggingFaceID)
	if err != nil {
		log.Printf("Failed to insert verified user: %v", err)
		b.respondToInteraction(s, i, "❌ Failed to record verification. Please try again later.", true)
		return
	}

	// Assign BLOCK role
	err = b.assignBlockRole(discordID, guildCfg)
	if err != nil {
		log.Printf("Failed to assign BLOCK role: %v", err)
		b.respondToInteraction(s, i, "✅ Verification successful! However, there was an issue assigning the BLOCK role. Please contact an administrator.", true)
		return
	}

	b.respondToInteraction(s, i, "✅ Verification successful! You have been assigned the BLOCK role.", true)
}

// assignBlockRole assigns the BLOCK role to a user
func (b *Bot) assignBlockRole(discordID string, guildCfg *GuildConfig) error {
	// For now, we'll use the existing role assignment logic
	// You may want to create a separate BLOCK role ID in the config
	return b.assignGSwarmRole(discordID, guildCfg)
}

// validateInputs validates and sanitizes user inputs
func (b *Bot) validateInputs(userAddress, trainingID, huggingFaceID string) error {
	// Check input lengths to prevent DoS attacks
	if len(userAddress) > 42 || len(trainingID) > 255 || len(huggingFaceID) > 255 {
		return fmt.Errorf("input too long")
	}

	// Check for SQL injection patterns
	if b.containsSQLInjectionPattern(userAddress) ||
		b.containsSQLInjectionPattern(trainingID) ||
		b.containsSQLInjectionPattern(huggingFaceID) {
		return fmt.Errorf("invalid characters detected in input")
	}

	// Validate Ethereum address format
	if !b.isValidEthereumAddress(userAddress) {
		return fmt.Errorf("invalid Ethereum address format")
	}

	// Validate training ID if provided
	if trainingID != "" {
		if len(trainingID) > 255 || !b.isValidTrainingID(trainingID) {
			return fmt.Errorf("invalid training ID format")
		}
	}

	// Validate Hugging Face ID if provided
	if huggingFaceID != "" {
		if len(huggingFaceID) > 255 || !b.isValidHuggingFaceID(huggingFaceID) {
			return fmt.Errorf("invalid Hugging Face ID format")
		}
	}

	return nil
}

// containsSQLInjectionPattern checks for common SQL injection patterns
func (b *Bot) containsSQLInjectionPattern(input string) bool {
	// Convert to lowercase for case-insensitive checking
	lowerInput := strings.ToLower(input)

	// Check for common SQL injection patterns
	sqlPatterns := []string{
		"';", "';--", "';/*", "';#",
		"union", "select", "insert", "update", "delete", "drop", "create", "alter",
		"exec", "execute", "script", "javascript:", "vbscript:", "onload=",
		"<script", "</script>", "javascript:", "vbscript:",
		"1=1", "1=1--", "1=1/*", "1=1#",
		"or 1=1", "or 1=1--", "or 1=1/*", "or 1=1#",
		"and 1=1", "and 1=1--", "and 1=1/*", "and 1=1#",
	}

	for _, pattern := range sqlPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}

	return false
}

// isValidEthereumAddress validates Ethereum address format
func (b *Bot) isValidEthereumAddress(address string) bool {
	// Remove 0x prefix if present for validation
	addr := strings.TrimPrefix(strings.ToLower(address), "0x")

	// Check length (40 hex characters)
	if len(addr) != 40 {
		return false
	}

	// Check if all characters are valid hex
	for _, char := range addr {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}

	return true
}

// isValidTrainingID validates training ID format
func (b *Bot) isValidTrainingID(trainingID string) bool {
	// Training ID should be alphanumeric with underscores and hyphens
	// Length between 1 and 255 characters
	if len(trainingID) == 0 || len(trainingID) > 255 {
		return false
	}

	// Check for valid characters (alphanumeric, underscore, hyphen)
	for _, char := range trainingID {
		if !((char >= '0' && char <= '9') ||
			(char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			char == '_' || char == '-') {
			return false
		}
	}

	return true
}

// isValidHuggingFaceID validates Hugging Face ID format
func (b *Bot) isValidHuggingFaceID(hfID string) bool {
	// Hugging Face ID should be in format: username/repository-name
	// Length between 1 and 255 characters
	if len(hfID) == 0 || len(hfID) > 255 {
		return false
	}

	// Check for valid characters (alphanumeric, underscore, hyphen, forward slash)
	for _, char := range hfID {
		if !((char >= '0' && char <= '9') ||
			(char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			char == '_' || char == '-' || char == '/') {
			return false
		}
	}

	// Must contain exactly one forward slash
	parts := strings.Split(hfID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}

	return true
}
