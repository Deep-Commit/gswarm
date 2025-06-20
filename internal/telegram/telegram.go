package telegram

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Blockchain constants
const (
	blockscoutURL     = "https://gensyn-testnet.explorer.alchemy.com/api"
	alchemyAPIURL     = "https://gensyn-testnet.g.alchemy.com/v2"
	alchemyPublicURL  = "https://gensyn-testnet.g.alchemy.com/public"
	rpcURL            = "https://gensyn-testnet.g.alchemy.com/public"
	coordAddrMath     = "0x69C6e1D608ec64885E7b185d39b04B491a71768C" // Proxy contract with activity
	coordAddrMathHard = "0x6947c6E196a48B77eFa9331EC1E3e45f3Ee5Fd58"
)

// TelegramConfig stores the info needed to send messages
// to Telegram
type TelegramConfig struct {
	BotToken    string `json:"bot_token"`
	ChatID      string `json:"chat_id"`
	WelcomeSent bool   `json:"welcome_sent"`
}

const DefaultConfigPath = "telegram-config.json"

// UserData represents the user data structure from Gensyn
type UserData struct {
	OrgID   string `json:"orgId"`
	Address string `json:"address"`
	UserID  string `json:"userId"`
	Email   string `json:"email"`
}

// UserDataFile represents the complete structure of userData.json
type UserDataFile struct {
	// The key is dynamic (like "org_12345"), so we'll use a map
	Data map[string]UserData `json:"-"`
}

// BlockchainData represents the blockchain data for a user
type BlockchainData struct {
	Votes   *big.Int
	Rewards *big.Int
	Balance *big.Int
}

// PreviousData stores the previous blockchain data for comparison
type PreviousData struct {
	Votes     *big.Int  `json:"votes"`
	Rewards   *big.Int  `json:"rewards"`
	LastCheck time.Time `json:"last_check"`
}

// TelegramService represents the telegram monitoring service
type TelegramService struct {
	UserDataPath      string
	ConfigPath        string
	ForceConfigUpdate bool
	Config            *TelegramConfig
	PreviousData      *PreviousData
	StopChan          chan bool
}

// NewTelegramService creates a new telegram service instance
func NewTelegramService(userDataPath string, configPath string, forceUpdate bool) *TelegramService {
	return &TelegramService{
		UserDataPath:      userDataPath,
		ConfigPath:        configPath,
		ForceConfigUpdate: forceUpdate,
		PreviousData:      &PreviousData{Votes: big.NewInt(0), Rewards: big.NewInt(0)},
		StopChan:          make(chan bool),
	}
}

// promptForTelegramConfig walks the user through CLI prompts
func promptForTelegramConfig() (*TelegramConfig, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Let's set up your Telegram integration!")

	fmt.Print("Enter your Telegram Bot Token: ")
	botToken, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	botToken = strings.TrimSpace(botToken)

	fmt.Print("Enter your Telegram Chat ID: ")
	chatID, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	chatID = strings.TrimSpace(chatID)

	return &TelegramConfig{
		BotToken: botToken,
		ChatID:   chatID,
	}, nil
}

// saveTelegramConfig writes the config to disk
func saveTelegramConfig(path string, cfg *TelegramConfig) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

// loadTelegramConfig loads the config from disk
func loadTelegramConfig(path string) (*TelegramConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg TelegramConfig
	dec := json.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ensureTelegramConfig loads or prompts for config
func (t *TelegramService) ensureTelegramConfig() error {
	cfgPath := t.ConfigPath
	if cfgPath == "" {
		cfgPath = DefaultConfigPath
	}
	if t.ForceConfigUpdate {
		fmt.Println("Forcing Telegram config update...")
		cfg, err := promptForTelegramConfig()
		if err != nil {
			return err
		}
		if err := saveTelegramConfig(cfgPath, cfg); err != nil {
			return err
		}
		t.Config = cfg
		return nil
	}
	// Try to load config
	cfg, err := loadTelegramConfig(cfgPath)
	if err == nil {
		t.Config = cfg
		return nil
	}
	fmt.Println("No Telegram config found. Let's set it up.")
	cfg, err = promptForTelegramConfig()
	if err != nil {
		return err
	}
	if err := saveTelegramConfig(cfgPath, cfg); err != nil {
		return err
	}
	t.Config = cfg
	return nil
}

func printBanner() {
	banner := `
 ██████  ███████ ██     ██  █████  ██████  ███    ███ 
██       ██      ██     ██ ██   ██ ██   ██ ████  ████ 
██   ███ ███████ ██  █  ██ ███████ ██████  ██ ████ ██ 
██    ██      ██ ██ ███ ██ ██   ██ ██   ██ ██  ██  ██ 
 ██████  ███████  ███ ███  ██   ██ ██   ██ ██      ██ 
		G-SWARM Supervisor (Community Project)
`
	fmt.Println("\033[38;5;224m")
	fmt.Println(banner)
	fmt.Println("\033[0m")
}

// findUserDataFile attempts to locate the userData.json file
func (t *TelegramService) findUserDataFile() (string, error) {
	// If a specific path was provided, use it
	if t.UserDataPath != "" {
		if _, err := os.Stat(t.UserDataPath); err == nil {
			return t.UserDataPath, nil
		}
		return "", fmt.Errorf("userData.json not found at specified path: %s", t.UserDataPath)
	}

	// Try to find the file in common locations
	possiblePaths := []string{
		"modal-login/temp-data/userData.json",
		"rl-swarm/modal-login/temp-data/userData.json",
		"../modal-login/temp-data/userData.json",
		"./modal-login/temp-data/userData.json",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("userData.json not found in any of the expected locations. Please specify the path manually")
}

// loadUserData loads and parses the userData.json file
func (t *TelegramService) loadUserData() (*UserData, error) {
	filePath, err := t.findUserDataFile()
	if err != nil {
		return nil, fmt.Errorf("failed to find userData.json: %w", err)
	}

	fmt.Printf("Found userData.json at: %s\n", filePath)

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read userData.json: %w", err)
	}

	// Parse the JSON as a map first to handle dynamic keys
	var rawData map[string]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse userData.json: %w", err)
	}

	// Get the first (and presumably only) organization data
	if len(rawData) == 0 {
		return nil, fmt.Errorf("userData.json is empty")
	}

	// Get the first key (org ID)
	var orgKey string
	for key := range rawData {
		orgKey = key
		break
	}

	// Extract the user data for this org
	orgData, ok := rawData[orgKey].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid structure in userData.json")
	}

	// Convert to UserData struct
	userData := &UserData{
		OrgID:   orgKey,
		Address: getStringValue(orgData, "address"),
		UserID:  getStringValue(orgData, "userId"),
		Email:   getStringValue(orgData, "email"),
	}

	return userData, nil
}

// getStringValue safely extracts a string value from a map
func getStringValue(data map[string]interface{}, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// sendTelegramMessage sends a message to Telegram using the Bot API
func (t *TelegramService) sendTelegramMessage(text string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.Config.BotToken)

	// Prepare the request data
	data := url.Values{}
	data.Set("chat_id", t.Config.ChatID)
	data.Set("text", text)

	// Make the HTTP request
	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API error: %s - %s", resp.Status, string(body))
	}

	// Parse the response to check for Telegram API errors
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result["ok"].(bool) {
		return fmt.Errorf("Telegram API error: %v", result["description"])
	}

	fmt.Printf("Message sent successfully to Telegram!\n")
	return nil
}

// Run starts the telegram monitoring service
func (t *TelegramService) Run() error {
	// Print banner
	printBanner()

	fmt.Println("Starting Telegram monitoring service...")
	if err := t.ensureTelegramConfig(); err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}
	fmt.Printf("Loaded Telegram config: BotToken=%s, ChatID=%s\n", t.Config.BotToken, t.Config.ChatID)

	// Send welcome message if not sent before
	if !t.Config.WelcomeSent {
		fmt.Println("Sending welcome message...")
		if err := t.sendWelcomeMessage(); err != nil {
			fmt.Printf("Warning: Could not send welcome message: %v\n", err)
		} else {
			// Mark welcome message as sent and save config
			t.Config.WelcomeSent = true

			// Determine the config path to save to
			configPath := t.ConfigPath
			if configPath == "" {
				configPath = DefaultConfigPath
			}

			fmt.Printf("Saving updated config to: %s\n", configPath)
			if err := saveTelegramConfig(configPath, t.Config); err != nil {
				fmt.Printf("Warning: Could not save updated config: %v\n", err)
			} else {
				fmt.Println("Welcome message sent and config updated!")
			}
		}
	} else {
		fmt.Println("Welcome message already sent previously.")
	}

	// Try to load user data
	userData, err := t.loadUserData()
	if err != nil {
		fmt.Printf("Warning: Could not load user data: %v\n", err)
		fmt.Println("Please ensure you have run the Gensyn application and created a user account.")
		fmt.Println("You can also specify the path to userData.json manually.")
		return err
	}

	fmt.Printf("Successfully loaded user data for: %s\n", userData.Email)
	fmt.Printf("User ID: %s\n", userData.UserID)
	fmt.Printf("Org ID: %s\n", userData.OrgID)
	fmt.Printf("Address: %s\n", userData.Address)

	// Load previous data from persistent storage
	previousData, err := t.loadPreviousData()
	if err != nil {
		fmt.Printf("Warning: Could not load previous data: %v\n", err)
		previousData = &PreviousData{Votes: big.NewInt(0), Rewards: big.NewInt(0), LastCheck: time.Now()}
	} else {
		fmt.Printf("Loaded previous data - Votes: %s, Rewards: %s, Last Check: %s\n",
			previousData.Votes.String(), previousData.Rewards.String(), previousData.LastCheck.Format("2006-01-02 15:04:05"))
	}

	fmt.Println("Starting continuous monitoring loop (checking every 5 minutes)...")
	fmt.Println("Press Ctrl+C to stop monitoring")

	// Start the monitoring loop
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Do initial check
	if err := t.checkAndNotify(userData, previousData); err != nil {
		fmt.Printf("Error in initial check: %v\n", err)
	}

	// Continuous monitoring loop
	for {
		select {
		case <-ticker.C:
			if err := t.checkAndNotify(userData, previousData); err != nil {
				fmt.Printf("Error in monitoring check: %v\n", err)
			}
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal. Stopping monitoring...")
			return nil
		case <-t.StopChan:
			fmt.Println("Monitoring stopped by user")
			return nil
		}
	}
}

// checkAndNotify checks blockchain data and sends notification if there are changes
func (t *TelegramService) checkAndNotify(userData *UserData, previousData *PreviousData) error {
	fmt.Printf("\n[%s] Checking blockchain data...\n", time.Now().Format("2006-01-02 15:04:05"))

	// Query blockchain data
	blockchainData, err := t.GetBlockchainData(userData.Address)
	if err != nil {
		fmt.Printf("Warning: Could not query blockchain data: %v\n", err)
		return err
	}

	fmt.Printf("Current - Votes: %s, Rewards: %s\n", blockchainData.Votes.String(), blockchainData.Rewards.String())
	fmt.Printf("Previous - Votes: %s, Rewards: %s\n", previousData.Votes.String(), previousData.Rewards.String())

	// Check if votes or rewards have increased
	votesIncreased := blockchainData.Votes.Cmp(previousData.Votes) > 0
	rewardsIncreased := blockchainData.Rewards.Cmp(previousData.Rewards) > 0

	if votesIncreased || rewardsIncreased {
		fmt.Println("Changes detected! Sending notification...")

		// Create notification message
		message := fmt.Sprintf(`🤖 <b>G-Swarm Node Update</b>

📊 <b>Node Information:</b>
• <b>Peer ID:</b> <code>%s</code>
• <b>User ID:</b> %s

🏆 <b>Blockchain Stats:</b>
• <b>Votes:</b> %s 🗳️%s
• <b>Rewards:</b> %s 💰%s

✨ <b>Status:</b> Progress Detected!`,
			userData.Address,
			userData.UserID,
			blockchainData.Votes.String(),
			func() string {
				if votesIncreased {
					voteDiff := new(big.Int).Sub(blockchainData.Votes, previousData.Votes)
					return fmt.Sprintf(" <i>(+%s)</i>", voteDiff.String())
				}
				return ""
			}(),
			blockchainData.Rewards.String(),
			func() string {
				if rewardsIncreased {
					rewardDiff := new(big.Int).Sub(blockchainData.Rewards, previousData.Rewards)
					return fmt.Sprintf(" <i>(+%s)</i>", rewardDiff.String())
				}
				return ""
			}())

		// Send the notification
		if err := t.sendTelegramMessageHTML(message); err != nil {
			fmt.Printf("Failed to send Telegram message: %v\n", err)
			return err
		}

		fmt.Println("Notification sent successfully!")
	} else {
		fmt.Println("No changes detected. Skipping notification.")
	}

	// Update and save the previous data
	previousData.Votes = blockchainData.Votes
	previousData.Rewards = blockchainData.Rewards
	previousData.LastCheck = time.Now()

	if err := t.savePreviousData(previousData); err != nil {
		fmt.Printf("Warning: Could not save previous data: %v\n", err)
	} else {
		fmt.Println("Previous data saved successfully.")
	}

	return nil
}

// queryUserVotes queries the smart contract for user votes using Alchemy API
// Function selector: 0xdfb3c7df
// Function signature: getVoterVoteCount(string memory peerId) public view returns (uint256)
func (t *TelegramService) queryUserVotes(peerId string, contractAddress string) (*big.Int, error) {
	// Function selector for getVoterVoteCount: 0xdfb3c7df
	methodID := "0xdfb3c7df"

	// Create the call data for string parameter
	// First, encode the offset to the string data (32 bytes)
	offset := "0000000000000000000000000000000000000000000000000000000000000020"

	// Then encode the string length
	stringLength := fmt.Sprintf("%064x", len(peerId))

	// Then encode the string data (padded to 32 bytes)
	stringBytes := []byte(peerId)
	stringHex := fmt.Sprintf("%x", stringBytes)
	// Pad to 32 bytes (64 hex chars)
	for len(stringHex) < 64 {
		stringHex += "0"
	}

	// Combine all parts
	data := methodID + offset + stringLength + stringHex

	// Create the JSON-RPC request
	request := AlchemyRequest{
		JSONRPC: "2.0",
		Method:  "eth_call",
		Params: []interface{}{
			map[string]interface{}{
				"to":   contractAddress,
				"data": data,
			},
			"latest",
		},
		ID: 1,
	}

	// Make the request to Alchemy API
	result, err := t.makeAlchemyRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call Alchemy API: %w", err)
	}

	// Parse the result
	if resultStr, ok := result.(string); ok {
		if strings.HasPrefix(resultStr, "0x") {
			resultStr = strings.TrimPrefix(resultStr, "0x")
			if len(resultStr) >= 64 {
				votes := new(big.Int)
				votes.SetString(resultStr, 16)
				return votes, nil
			}
		}
	}

	return big.NewInt(0), nil
}

// queryUserRewards queries the smart contract for user rewards using Alchemy API
// Function selector: 0x80c3d97f
// Function signature: getTotalRewards(string[] memory peerIds) public view returns (int256[])
func (t *TelegramService) queryUserRewards(peerIds []string, contractAddress string) (*big.Int, error) {
	// Function selector for getTotalRewards: 0x80c3d97f
	methodID := "0x80c3d97f"

	// Create the call data for string[] parameter
	// First, encode the offset to the array data (32 bytes)
	offset := "0000000000000000000000000000000000000000000000000000000000000020"

	// Then encode the array length
	arrayLength := fmt.Sprintf("%064x", len(peerIds))

	// Then encode each string in the array
	var stringData string
	currentOffset := len(peerIds) * 32 // Start of string content after the offsets
	for _, peerId := range peerIds {
		// Encode string offset (relative to start of array data)
		stringOffsetHex := fmt.Sprintf("%064x", currentOffset)
		stringData += stringOffsetHex
		// The length of the string content in bytes
		currentOffset += ((len(peerId) + 31) / 32) * 32
	}

	// Then encode the actual string data
	for _, peerId := range peerIds {
		// Encode string length
		stringLength := fmt.Sprintf("%064x", len(peerId))
		// Encode string data and pad to 32 bytes
		stringBytes := []byte(peerId)
		stringHex := fmt.Sprintf("%x", stringBytes)
		// Pad to a multiple of 32 bytes (64 hex chars)
		for len(stringHex)%64 != 0 {
			stringHex += "0"
		}
		stringData += stringLength + stringHex
	}

	// Combine all parts
	data := methodID + offset + arrayLength + stringData

	// Create the JSON-RPC request
	request := AlchemyRequest{
		JSONRPC: "2.0",
		Method:  "eth_call",
		Params: []interface{}{
			map[string]interface{}{
				"to":   contractAddress,
				"data": data,
			},
			"latest",
		},
		ID: 1,
	}

	// Make the request to Alchemy API
	result, err := t.makeAlchemyRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call Alchemy API: %w", err)
	}

	// Parse the result - this returns int256[] (array of rewards)
	if resultStr, ok := result.(string); ok {
		if strings.HasPrefix(resultStr, "0x") {
			resultStr = strings.TrimPrefix(resultStr, "0x")
			// The result is an array, so we need to parse it correctly
			// Format: [offset][length][value1][value2]...
			if len(resultStr) >= 128 { // At least offset + length
				// Get array length
				arrayLengthHex := resultStr[64:128]
				arrayLength := new(big.Int)
				arrayLength.SetString(arrayLengthHex, 16)

				// If we have at least one value, get the first one
				if arrayLength.Cmp(big.NewInt(0)) > 0 && len(resultStr) >= 192 {
					firstValueHex := resultStr[128:192]
					rewards := new(big.Int)
					rewards.SetString(firstValueHex, 16)
					return rewards, nil
				}
			}
		}
	}

	return big.NewInt(0), nil
}

// queryUserBalance queries the user's ETH balance using Alchemy API
func (t *TelegramService) queryUserBalance(userAddress string) (*big.Int, error) {
	// Create the JSON-RPC request
	request := AlchemyRequest{
		JSONRPC: "2.0",
		Method:  "eth_getBalance",
		Params: []interface{}{
			userAddress,
			"latest",
		},
		ID: 1,
	}

	// Make the request to Alchemy API
	result, err := t.makeAlchemyRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call Alchemy API: %w", err)
	}

	// Parse the result
	if resultStr, ok := result.(string); ok {
		if strings.HasPrefix(resultStr, "0x") {
			resultStr = strings.TrimPrefix(resultStr, "0x")
			balance := new(big.Int)
			balance.SetString(resultStr, 16)
			return balance, nil
		}
	}

	return big.NewInt(0), nil
}

// makeAlchemyRequest makes a request to the Alchemy API
func (t *TelegramService) makeAlchemyRequest(request AlchemyRequest) (interface{}, error) {
	// Prepare the request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request - use public endpoint
	url := alchemyPublicURL
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Debug: Print the response
	fmt.Printf("Alchemy API Response: %s\n", string(body))

	// Check if response is JSON
	if !strings.HasPrefix(strings.TrimSpace(string(body)), "{") {
		return nil, fmt.Errorf("non-JSON response from Alchemy API: %s", string(body))
	}

	// Parse the response
	var response AlchemyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if response.Error != nil {
		return nil, fmt.Errorf("Alchemy API error: %s (code: %d)", response.Error.Message, response.Error.Code)
	}

	return response.Result, nil
}

// GetBlockchainData queries all blockchain data for a user using Alchemy API
func (t *TelegramService) GetBlockchainData(userAddress string) (*BlockchainData, error) {
	fmt.Printf("Querying blockchain data for address: %s\n", userAddress)

	// Try both contract addresses
	contracts := []string{coordAddrMath, coordAddrMathHard}

	var votes *big.Int
	var rewards *big.Int

	// Try to get votes from either contract
	// For votes, we pass the address as a peer ID
	for _, contract := range contracts {
		if v, err := t.queryUserVotes(userAddress, contract); err == nil && v.Cmp(big.NewInt(0)) > 0 {
			votes = v
			fmt.Printf("Found votes in contract %s: %s\n", contract, votes.String())
			break
		} else {
			fmt.Printf("No votes found in contract %s: %v\n", contract, err)
		}
	}

	// Try to get rewards from either contract
	// For rewards, we need to pass an array of peer IDs
	peerIds := []string{userAddress} // For now, treat the address as a peer ID
	for _, contract := range contracts {
		if r, err := t.queryUserRewards(peerIds, contract); err == nil && r.Cmp(big.NewInt(0)) > 0 {
			rewards = r
			fmt.Printf("Found rewards in contract %s: %s\n", contract, rewards.String())
			break
		} else {
			fmt.Printf("No rewards found in contract %s: %v\n", contract, err)
		}
	}

	// Get ETH balance (only if it's an Ethereum address)
	var balance *big.Int
	if strings.HasPrefix(userAddress, "0x") && len(userAddress) == 42 {
		balance, err := t.queryUserBalance(userAddress)
		if err != nil {
			fmt.Printf("Failed to get balance: %v\n", err)
			balance = big.NewInt(0)
		} else {
			fmt.Printf("Found balance: %s wei\n", balance.String())
		}
	} else {
		fmt.Printf("Skipping balance query - not an Ethereum address: %s\n", userAddress)
		balance = big.NewInt(0)
	}

	// Initialize with zero values if not found
	if votes == nil {
		votes = big.NewInt(0)
	}
	if rewards == nil {
		rewards = big.NewInt(0)
	}

	return &BlockchainData{
		Votes:   votes,
		Rewards: rewards,
		Balance: balance,
	}, nil
}

// AlchemyRequest represents a JSON-RPC request to Alchemy
type AlchemyRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// AlchemyResponse represents a JSON-RPC response from Alchemy
type AlchemyResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// sendTelegramMessageWithMarkdown sends a message to Telegram using the Bot API with MarkdownV2 formatting
func (t *TelegramService) sendTelegramMessageWithMarkdown(text string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.Config.BotToken)

	// Prepare the request data
	data := url.Values{}
	data.Set("chat_id", t.Config.ChatID)
	data.Set("text", text)
	data.Set("parse_mode", "MarkdownV2")

	// Make the HTTP request
	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API error: %s - %s", resp.Status, string(body))
	}

	// Parse the response to check for Telegram API errors
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// If parsing fails, it might not be a JSON response, but could still be a success.
		// For now, let's log the raw response and assume success if status is OK.
		fmt.Printf("Message sent successfully to Telegram! (non-JSON response: %s)\n", string(body))
		return nil
	}

	if val, ok := result["ok"]; !ok || !val.(bool) {
		return fmt.Errorf("Telegram API error: %v", result["description"])
	}

	fmt.Printf("Message sent successfully to Telegram!\n")
	return nil
}

// escapeLineStartHyphens escapes hyphens at the start of lines for MarkdownV2
func escapeLineStartHyphens(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "-") {
			lines[i] = "\\-" + line[1:]
		}
	}
	return strings.Join(lines, "\n")
}

// sendTelegramMessageHTML sends a message to Telegram using the Bot API with HTML formatting
func (t *TelegramService) sendTelegramMessageHTML(text string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.Config.BotToken)

	// Prepare the request data
	data := url.Values{}
	data.Set("chat_id", t.Config.ChatID)
	data.Set("text", text)
	data.Set("parse_mode", "HTML")

	// Make the HTTP request
	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API error: %s - %s", resp.Status, string(body))
	}

	// Parse the response to check for Telegram API errors
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// If parsing fails, it might not be a JSON response, but could still be a success.
		// For now, let's log the raw response and assume success if status is OK.
		fmt.Printf("Message sent successfully to Telegram! (non-JSON response: %s)\n", string(body))
		return nil
	}

	if val, ok := result["ok"]; !ok || !val.(bool) {
		return fmt.Errorf("Telegram API error: %v", result["description"])
	}

	fmt.Printf("Message sent successfully to Telegram!\n")
	return nil
}

// savePreviousData saves the previous data to a JSON file
func (t *TelegramService) savePreviousData(data *PreviousData) error {
	// Convert big.Int to string for JSON serialization
	dataToSave := map[string]interface{}{
		"votes":      data.Votes.String(),
		"rewards":    data.Rewards.String(),
		"last_check": data.LastCheck.Format(time.RFC3339),
	}

	filePath := "telegram_previous_data.json"
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create previous data file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(dataToSave)
}

// loadPreviousData loads the previous data from a JSON file
func (t *TelegramService) loadPreviousData() (*PreviousData, error) {
	filePath := "telegram_previous_data.json"
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return default data
			return &PreviousData{
				Votes:     big.NewInt(0),
				Rewards:   big.NewInt(0),
				LastCheck: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to open previous data file: %w", err)
	}
	defer file.Close()

	var dataMap map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&dataMap); err != nil {
		return nil, fmt.Errorf("failed to decode previous data: %w", err)
	}

	// Parse votes
	votesStr, ok := dataMap["votes"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid votes data")
	}
	votes := new(big.Int)
	votes.SetString(votesStr, 10)

	// Parse rewards
	rewardsStr, ok := dataMap["rewards"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid rewards data")
	}
	rewards := new(big.Int)
	rewards.SetString(rewardsStr, 10)

	// Parse last check time
	lastCheckStr, ok := dataMap["last_check"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid last_check data")
	}
	lastCheck, err := time.Parse(time.RFC3339, lastCheckStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse last_check time: %w", err)
	}

	return &PreviousData{
		Votes:     votes,
		Rewards:   rewards,
		LastCheck: lastCheck,
	}, nil
}

// sendWelcomeMessage sends a welcome message to new users
func (t *TelegramService) sendWelcomeMessage() error {
	message := `🤖 <b>Welcome to G-Swarm Monitor!</b>

This bot monitors your Gensyn AI node activity and notifies you when your votes or rewards increase.

<b>Features:</b>
• Monitors votes and rewards every 5 minutes
• Sends notifications only when there are changes
• Tracks progress across multiple contracts

<b>Support Development:</b>
If you find this bot useful, please consider donating to support ongoing development and new features:

ETH: <code>0xA22e20BA3336f5Bd6eCE959F5ac4083C9693e316</code>

Thank you for using G-Swarm Monitor! 🚀`

	return t.sendTelegramMessageHTML(message)
}
