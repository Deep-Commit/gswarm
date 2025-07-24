package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

// LinkCode represents a verification code for linking Discord and Telegram accounts
type LinkCode struct {
	Code       string    `json:"code"`
	DiscordID  string    `json:"discord_id"`
	TelegramID string    `json:"telegram_id,omitempty"` // Optional pre-binding
	ExpiresAt  time.Time `json:"expires_at"`
	Used       bool      `json:"used"`
	CreatedAt  time.Time `json:"created_at"`
}

// LinkedAccount represents a linked Discord-Telegram account
type LinkedAccount struct {
	DiscordID  string    `json:"discord_id"`
	TelegramID string    `json:"telegram_id"`
	LinkedAt   time.Time `json:"linked_at"`
}

// IssueCodeRequest represents the request to issue a linking code
type IssueCodeRequest struct {
	DiscordID  string `json:"discord_id"`
	TelegramID string `json:"telegram_id,omitempty"` // Optional pre-binding
}

// IssueCodeResponse represents the response from issuing a code
type IssueCodeResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code,omitempty"`
	Error   string `json:"error,omitempty"`
}

// VerifyCodeRequest represents the request to verify a linking code
type VerifyCodeRequest struct {
	Code       string `json:"code"`
	TelegramID string `json:"telegram_id"`
}

// VerifyCodeResponse represents the response from verifying a code
type VerifyCodeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// API represents the API server for account linking
type API struct {
	apiKey       string
	codes        map[string]*LinkCode
	accounts     map[string]*LinkedAccount // Key: discord_id
	pendingRoles map[string]bool           // Key: discord_id, tracks users who need role assignment
	mu           sync.RWMutex
	codeExpiry   time.Duration
}

// NewAPI creates a new API instance
func NewAPI(apiKey string) *API {
	return &API{
		apiKey:       apiKey,
		codes:        make(map[string]*LinkCode),
		accounts:     make(map[string]*LinkedAccount),
		pendingRoles: make(map[string]bool),
		codeExpiry:   10 * time.Minute, // Codes expire after 10 minutes
	}
}

// Start starts the API server
func (api *API) Start(addr string) error {
	// Set up routes
	http.HandleFunc("/api/telegrams/issue-code", api.handleIssueCode)
	http.HandleFunc("/api/telegrams/verify-code", api.handleVerifyCode)
	http.HandleFunc("/api/telegrams/linked-accounts", api.handleGetLinkedAccounts)
	http.HandleFunc("/api/telegrams/pending-role-assignments", api.handleGetPendingRoleAssignments)

	// Start cleanup goroutine
	go api.cleanupExpiredCodes()

	log.Printf("API server starting on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// handleIssueCode handles the code issuance endpoint (Discord bot only)
func (api *API) handleIssueCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify API key (Discord bot only)
	if !api.verifyAPIKey(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req IssueCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.DiscordID == "" {
		http.Error(w, "Discord ID is required", http.StatusBadRequest)
		return
	}

	// Check if account is already linked
	api.mu.RLock()
	if _, exists := api.accounts[req.DiscordID]; exists {
		api.mu.RUnlock()
		response := IssueCodeResponse{
			Success: false,
			Error:   "Account already linked",
		}
		api.sendJSONResponse(w, response)
		return
	}
	api.mu.RUnlock()

	// Generate a unique code
	code, err := api.generateCode()
	if err != nil {
		log.Printf("Failed to generate code: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store the code
	linkCode := &LinkCode{
		Code:       code,
		DiscordID:  req.DiscordID,
		TelegramID: req.TelegramID, // Optional pre-binding
		ExpiresAt:  time.Now().Add(api.codeExpiry),
		Used:       false,
		CreatedAt:  time.Now(),
	}

	api.mu.Lock()
	api.codes[code] = linkCode
	api.mu.Unlock()

	response := IssueCodeResponse{
		Success: true,
		Code:    code,
	}

	api.sendJSONResponse(w, response)
	log.Printf("Issued code %s for Discord ID %s", code, req.DiscordID)
}

// handleVerifyCode handles the code verification endpoint (Telegram bot)
func (api *API) handleVerifyCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" || req.TelegramID == "" {
		http.Error(w, "Code and Telegram ID are required", http.StatusBadRequest)
		return
	}

	// Verify the code
	success, message := api.verifyCode(req.Code, req.TelegramID)

	response := VerifyCodeResponse{
		Success: success,
		Message: message,
	}

	if !success {
		response.Error = message
	}

	api.sendJSONResponse(w, response)
}

// handleGetLinkedAccounts handles getting linked accounts (for debugging/admin)
func (api *API) handleGetLinkedAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify API key
	if !api.verifyAPIKey(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	api.mu.RLock()
	accounts := make([]*LinkedAccount, 0, len(api.accounts))
	for _, account := range api.accounts {
		accounts = append(accounts, account)
	}
	api.mu.RUnlock()

	api.sendJSONResponse(w, accounts)
}

// verifyAPIKey verifies the API key from the request
func (api *API) verifyAPIKey(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	// Expect "Bearer <api_key>"
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return false
	}

	key := authHeader[7:]
	return key == api.apiKey
}

// generateCode generates a random 6-character alphanumeric code
func (api *API) generateCode() (string, error) {
	bytes := make([]byte, 3) // 3 bytes = 6 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:6], nil
}

// verifyCode verifies a linking code and creates the account link
func (api *API) verifyCode(code, telegramID string) (bool, string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	linkCode, exists := api.codes[code]
	if !exists {
		return false, "Invalid code"
	}

	if linkCode.Used {
		return false, "Code already used"
	}

	if time.Now().After(linkCode.ExpiresAt) {
		return false, "Code expired"
	}

	// Check if Telegram ID is already linked to another Discord account
	for _, account := range api.accounts {
		if account.TelegramID == telegramID {
			return false, "Telegram account already linked to another Discord account"
		}
	}

	// Mark code as used
	linkCode.Used = true

	// Create the account link
	linkedAccount := &LinkedAccount{
		DiscordID:  linkCode.DiscordID,
		TelegramID: telegramID,
		LinkedAt:   time.Now(),
	}

	api.accounts[linkCode.DiscordID] = linkedAccount

	log.Printf("Successfully linked Discord ID %s with Telegram ID %s", linkCode.DiscordID, telegramID)

	// Trigger Discord role assignment
	go api.triggerDiscordRoleAssignment(linkCode.DiscordID)

	return true, "Account successfully linked"
}

// triggerDiscordRoleAssignment triggers role assignment for a Discord user
func (api *API) triggerDiscordRoleAssignment(discordID string) {
	api.mu.Lock()
	api.pendingRoles[discordID] = true
	api.mu.Unlock()

	log.Printf("Added Discord user %s to pending role assignments", discordID)
}

// handleGetPendingRoleAssignments handles getting pending role assignments
func (api *API) handleGetPendingRoleAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify API key
	if !api.verifyAPIKey(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	api.mu.Lock()
	pendingUsers := make([]string, 0, len(api.pendingRoles))
	for discordID := range api.pendingRoles {
		pendingUsers = append(pendingUsers, discordID)
		delete(api.pendingRoles, discordID) // Remove from pending after returning
	}
	api.mu.Unlock()

	response := struct {
		PendingUsers []string `json:"pending_users"`
	}{
		PendingUsers: pendingUsers,
	}

	api.sendJSONResponse(w, response)
}

// cleanupExpiredCodes removes expired codes periodically
func (api *API) cleanupExpiredCodes() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		api.mu.Lock()
		now := time.Now()
		for code, linkCode := range api.codes {
			if now.After(linkCode.ExpiresAt) {
				delete(api.codes, code)
			}
		}
		api.mu.Unlock()
	}
}

// sendJSONResponse sends a JSON response
func (api *API) sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// GetLinkedAccount retrieves a linked account by Discord ID
func (api *API) GetLinkedAccount(discordID string) (*LinkedAccount, bool) {
	api.mu.RLock()
	defer api.mu.RUnlock()

	account, exists := api.accounts[discordID]
	return account, exists
}

// GetLinkedAccountByTelegram retrieves a linked account by Telegram ID
func (api *API) GetLinkedAccountByTelegram(telegramID string) (*LinkedAccount, bool) {
	api.mu.RLock()
	defer api.mu.RUnlock()

	for _, account := range api.accounts {
		if account.TelegramID == telegramID {
			return account, true
		}
	}
	return nil, false
}
