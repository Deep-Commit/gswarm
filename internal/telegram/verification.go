package telegram

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// VerificationCode represents a verification code for Discord linking
type VerificationCode struct {
	Code       string    `json:"code"`
	TelegramID string    `json:"telegram_id"`
	PeerID     string    `json:"peer_id"`
	EOA        string    `json:"eoa"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// APIResponse represents the response from the verification API
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Rank represents rank data for a peer
type Rank struct {
	PeerID       string    `json:"peerId"`
	Rank         int       `json:"rank"`
	TotalWins    int       `json:"totalWins"`
	TotalRewards int       `json:"totalRewards"`
	LastSeen     time.Time `json:"lastSeen"`
}

// UnmarshalJSON custom unmarshaling for Rank to handle the time format
func (r *Rank) UnmarshalJSON(data []byte) error {
	// Create a temporary struct to unmarshal the JSON
	type rankAlias Rank
	aux := &struct {
		LastSeen string `json:"lastSeen"`
		*rankAlias
	}{
		rankAlias: (*rankAlias)(r),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Only parse if LastSeen is present and non-empty
	timeStr := aux.LastSeen
	if timeStr != "" {
		var parsedTime time.Time
		var err error
		formats := []string{
			"2006-01-02T15:04:05.999999-07:00",
			"2006-01-02T15:04:05.999999+00:00",
			"2006-01-02T15:04:05.999999",
			"2006-01-02T15:04:05.999999Z",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
		}
		for _, format := range formats {
			parsedTime, err = time.Parse(format, timeStr)
			if err == nil {
				r.LastSeen = parsedTime
				break
			}
		}
		if err != nil {
			return fmt.Errorf("failed to parse time '%s': %w", timeStr, err)
		}
	}
	// If timeStr is empty, leave r.LastSeen as zero value

	return nil
}

// VerificationStatus represents the verification status response from the API
type VerificationStatus struct {
	TelegramID int64     `json:"telegramId"`
	IsVerified bool      `json:"isVerified"`
	DiscordID  int64     `json:"discordId"`
	VerifiedAt time.Time `json:"verifiedAt"`
	Nodes      []Node    `json:"nodes"`
}

// UnmarshalJSON custom unmarshaling for VerificationStatus to handle the time format
func (v *VerificationStatus) UnmarshalJSON(data []byte) error {
	// Create a temporary struct to unmarshal the JSON
	type verificationStatusAlias VerificationStatus
	aux := &struct {
		VerifiedAt string `json:"verifiedAt"`
		*verificationStatusAlias
	}{
		verificationStatusAlias: (*verificationStatusAlias)(v),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Only parse if VerifiedAt is present and non-empty
	timeStr := aux.VerifiedAt
	if timeStr != "" {
		var parsedTime time.Time
		var err error
		formats := []string{
			"2006-01-02T15:04:05.999999-07:00",
			"2006-01-02T15:04:05.999999+00:00",
			"2006-01-02T15:04:05.999999",
			"2006-01-02T15:04:05.999999Z",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
		}
		for _, format := range formats {
			parsedTime, err = time.Parse(format, timeStr)
			if err == nil {
				v.VerifiedAt = parsedTime
				break
			}
		}
		if err != nil {
			return fmt.Errorf("failed to parse time '%s': %w", timeStr, err)
		}
	}
	// If timeStr is empty, leave v.VerifiedAt as zero value

	return nil
}

// Node represents a node in the verification status response
type Node struct {
	PeerID  string    `json:"peer_id"`
	EOA     string    `json:"eoa"`
	AddedAt time.Time `json:"added_at"`
}

// UnmarshalJSON custom unmarshaling for Node to handle the time format
func (n *Node) UnmarshalJSON(data []byte) error {
	// Create a temporary struct to unmarshal the JSON
	type nodeAlias Node
	aux := &struct {
		AddedAt string `json:"added_at"`
		*nodeAlias
	}{
		nodeAlias: (*nodeAlias)(n),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Parse the time string - try multiple formats
	timeStr := aux.AddedAt
	var parsedTime time.Time
	var err error

	// Try different time formats
	formats := []string{
		"2006-01-02T15:04:05.999999-07:00",
		"2006-01-02T15:04:05.999999+00:00",
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05.999999Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
	}

	for _, format := range formats {
		parsedTime, err = time.Parse(format, timeStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		return fmt.Errorf("failed to parse time '%s': %w", timeStr, err)
	}

	n.AddedAt = parsedTime
	return nil
}

// UserRankData represents the user rank data response from the API
type UserRankData struct {
	TelegramID int64     `json:"telegramId"`
	DiscordID  int64     `json:"discordId"`
	VerifiedAt time.Time `json:"verifiedAt"`
	Ranks      []Rank    `json:"ranks"`
	Stats      Stats     `json:"stats"`
}

// UnmarshalJSON custom unmarshaling for UserRankData to handle the time format
func (u *UserRankData) UnmarshalJSON(data []byte) error {
	// Create a temporary struct to unmarshal the JSON
	type userRankDataAlias UserRankData
	aux := &struct {
		VerifiedAt string `json:"verifiedAt"`
		*userRankDataAlias
	}{
		userRankDataAlias: (*userRankDataAlias)(u),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Only parse if VerifiedAt is present and non-empty
	timeStr := aux.VerifiedAt
	if timeStr != "" {
		var parsedTime time.Time
		var err error
		formats := []string{
			"2006-01-02T15:04:05.999999-07:00",
			"2006-01-02T15:04:05.999999+00:00",
			"2006-01-02T15:04:05.999999",
			"2006-01-02T15:04:05.999999Z",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
		}
		for _, format := range formats {
			parsedTime, err = time.Parse(format, timeStr)
			if err == nil {
				u.VerifiedAt = parsedTime
				break
			}
		}
		if err != nil {
			return fmt.Errorf("failed to parse time '%s': %w", timeStr, err)
		}
	}
	// If timeStr is empty, leave u.VerifiedAt as zero value

	return nil
}

// Stats represents user statistics
type Stats struct {
	TotalNodes  int `json:"totalNodes"`
	RankedNodes int `json:"rankedNodes"`
}

// generateVerificationCode creates an 8-character alphanumeric code
func generateVerificationCode() (string, error) {
	// Define the character set: A-Z and 0-9
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// Generate 8 random characters
	code := make([]byte, 8)
	for i := range code {
		// Generate a random byte
		randomBytes := make([]byte, 1)
		if _, err := rand.Read(randomBytes); err != nil {
			return "", fmt.Errorf("failed to generate random bytes: %w", err)
		}

		// Use modulo to get a random index in the charset
		index := int(randomBytes[0]) % len(charset)
		code[i] = charset[index]
	}

	return string(code), nil
}

// IssueVerificationCode sends a verification code to the API
func (t *TelegramService) IssueVerificationCode(telegramID string) (*VerificationCode, error) {
	// Generate verification code
	code, err := generateVerificationCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification code: %w", err)
	}

	// Convert chat_id to number for API
	var chatIDNum int64
	if _, err := fmt.Sscanf(t.Config.ChatID, "%d", &chatIDNum); err != nil {
		return nil, fmt.Errorf("failed to parse chat ID as number: %w", err)
	}

	// Create verification request with correct field names (camelCase) and types
	verificationReq := map[string]interface{}{
		"code":       code,
		"telegramId": chatIDNum, // Use chat_id as number
		"peerIds":    t.PeerIDs, // Send all peer IDs
		"eoa":        t.UserEOAAddress,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(verificationReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal verification request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/telegrams/issue-code", t.Config.APIURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API returned error: %s", apiResp.Error)
	}

	// Create verification code object (still use first peer ID for local reference)
	verificationCode := &VerificationCode{
		Code:       code,
		TelegramID: t.Config.ChatID, // Use chat_id
		PeerID:     t.PeerIDs[0],    // Keep for backward compatibility
		EOA:        t.UserEOAAddress,
		ExpiresAt:  time.Now().Add(15 * time.Minute), // 15 minute expiration
	}

	return verificationCode, nil
}

// handleVerificationCommand handles the /verify command from Telegram
func (t *TelegramService) handleVerificationCommand(telegramID string) error {
	fmt.Printf("Handling verification request for Telegram ID: %s\n", telegramID)

	// Check if we have the required data
	if len(t.PeerIDs) == 0 {
		return fmt.Errorf("no peer IDs available for verification")
	}

	if t.UserEOAAddress == "" {
		return fmt.Errorf("no EOA address available for verification")
	}

	// Issue verification code using chat_id
	verificationCode, err := t.IssueVerificationCode(t.Config.ChatID)
	if err != nil {
		return fmt.Errorf("failed to issue verification code: %w", err)
	}

	fmt.Printf("Generated verification code: %s for Chat ID: %s\n", verificationCode.Code, t.Config.ChatID)

	// Send instructions to user
	if err := t.sendVerificationInstructions(verificationCode.Code); err != nil {
		return fmt.Errorf("failed to send verification instructions: %w", err)
	}

	return nil
}

// CheckVerificationStatus checks if a user has been verified in Discord
func (t *TelegramService) CheckVerificationStatus(telegramID int64) (*VerificationStatus, error) {
	// Create verification check request
	verificationReq := map[string]interface{}{
		"telegramId": telegramID,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(verificationReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal verification check request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/telegrams/check-verification", t.Config.APIURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Debug: Print the raw response
	fmt.Printf("DEBUG: Verification status API response (status %d): %s\n", resp.StatusCode, string(body))

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var verificationStatus VerificationStatus
	if err := json.Unmarshal(body, &verificationStatus); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	// Debug: Print parsed verification status
	fmt.Printf("DEBUG: Parsed verification status: TelegramID=%d, IsVerified=%t, DiscordID=%d\n",
		verificationStatus.TelegramID, verificationStatus.IsVerified, verificationStatus.DiscordID)

	return &verificationStatus, nil
}

// GetUserRankData gets rank data for verified users' peer IDs
func (t *TelegramService) GetUserRankData(telegramID int64) (*UserRankData, error) {
	// Build peerIds query parameter as comma-separated string
	peerIdsParam := strings.Join(t.PeerIDs, ",")
	// Create HTTP request with peerIds in query
	url := fmt.Sprintf("%s/user/data?telegramId=%d&peerIds=%s", t.Config.APIURL, telegramID, url.QueryEscape(peerIdsParam))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Debug: Print the raw response
	fmt.Printf("DEBUG: User rank data API response (status %d): %s\n", resp.StatusCode, string(body))

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var userRankData UserRankData
	if err := json.Unmarshal(body, &userRankData); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	// Debug: Print parsed data
	fmt.Printf("DEBUG: Parsed user rank data: TelegramID=%d, DiscordID=%d, Ranks=%d\n",
		userRankData.TelegramID, userRankData.DiscordID, len(userRankData.Ranks))

	return &userRankData, nil
}
