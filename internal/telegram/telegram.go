package telegram

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Blockchain constants
const (
	blockscoutURL    = "https://gensyn-testnet.explorer.alchemy.com/api"
	alchemyAPIURL    = "https://gensyn-testnet.g.alchemy.com/v2"
	alchemyPublicURL = "https://gensyn-testnet.g.alchemy.com/public"
	rpcURL           = "https://gensyn-testnet.g.alchemy.com/public"
	coordAddrGenRL   = "0xFaD7C5e93f28257429569B854151A1B8DCD404c2" // GenRL-Swarm contract
)

// API constants
const (
	defaultAPIURL = "https://gswarm.dev/api" // Change this for custom backends
)

// ABI for the getPeerId function
const coordABI = `[{"constant":true,"inputs":[{"name":"eoaAddresses","type":"address[]"}],"name":"getPeerId","outputs":[{"name":"","type":"string[][]"}],"stateMutability":"view","type":"function"}]`

// TelegramConfig stores the info needed to send messages
// to Telegram
type TelegramConfig struct {
	BotToken    string `json:"bot_token"`
	ChatID      string `json:"chat_id"`
	WelcomeSent bool   `json:"welcome_sent"`
	APIURL      string `json:"api_url"`
}

const DefaultConfigPath = "telegram-config.json"

// BlockchainData represents the blockchain data for a user
type BlockchainData struct {
	Votes   *big.Int
	Rewards *big.Int
	Balance *big.Int
}

// PeerPreviousData stores previous values for a single peer
// Used for per-peer delta tracking
// Add more fields as needed (e.g., Rank, Wins)
type PeerPreviousData struct {
	Votes   string `json:"votes"`
	Rewards string `json:"rewards"`
}

// PreviousData stores the previous blockchain data for comparison
// Now supports both total and per-peer tracking
// The Peers map key is the peer ID
type PreviousData struct {
	Votes     *big.Int                    `json:"votes"`   // total
	Rewards   *big.Int                    `json:"rewards"` // total
	Peers     map[string]PeerPreviousData `json:"peers"`   // per-peer
	LastCheck time.Time                   `json:"last_check"`
}

// TelegramService represents the telegram monitoring service
type TelegramService struct {
	ConfigPath        string
	ForceConfigUpdate bool
	Config            *TelegramConfig
	UserEOAAddress    string
	EOAAddress        string // For non-interactive mode
	BotToken          string // For non-interactive mode
	ChatID            string // For non-interactive mode
	PeerIDs           []string
	PreviousData      *PreviousData
	StopChan          chan bool
	messageMutex      sync.Mutex // Protect message building
}

// NewTelegramService creates a new telegram service instance
func NewTelegramService(configPath string, forceUpdate bool, eoaAddress string, botToken string, chatID string) *TelegramService {
	return &TelegramService{
		ConfigPath:        configPath,
		ForceConfigUpdate: forceUpdate,
		EOAAddress:        eoaAddress,
		BotToken:          botToken,
		ChatID:            chatID,
		PreviousData:      &PreviousData{Votes: big.NewInt(0), Rewards: big.NewInt(0), Peers: make(map[string]PeerPreviousData)},
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
		APIURL:   defaultAPIURL,
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
		return nil, fmt.Errorf("failed to open config file %q: %w", path, err)
	}
	defer f.Close()
	var cfg TelegramConfig
	dec := json.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w (expected JSON with fields: bot_token, chat_id, welcome_sent, api_url)", path, err)
	}

	// Validate required fields
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("config file %q is missing required field 'bot_token'", path)
	}
	if cfg.ChatID == "" {
		return nil, fmt.Errorf("config file %q is missing required field 'chat_id'", path)
	}

	return &cfg, nil
}

// EnsureTelegramConfig loads or prompts for config
func (t *TelegramService) EnsureTelegramConfig() error {
	// Check if we have bot token and chat ID from command line
	if t.BotToken != "" && t.ChatID != "" {
		t.Config = &TelegramConfig{
			BotToken:    t.BotToken,
			ChatID:      t.ChatID,
			WelcomeSent: false,
			APIURL:      defaultAPIURL,
		}
		return nil
	}

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
		// Ensure API URL is set
		if cfg.APIURL == "" {
			cfg.APIURL = defaultAPIURL
			if err := saveTelegramConfig(cfgPath, cfg); err != nil {
				return err
			}
		}
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
	if err := t.EnsureTelegramConfig(); err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}
	fmt.Printf("Loaded Telegram config: BotToken=[REDACTED], ChatID=%s\n", t.Config.ChatID)

	// Send welcome message if not sent before
	if !t.Config.WelcomeSent {
		fmt.Println("Welcome message will be sent after collecting EOA address and peer IDs...")
	} else {
		fmt.Println("Welcome message already sent previously.")
	}

	// Get EOA address from command line or prompt user
	var eoaAddress string
	if t.EOAAddress != "" {
		eoaAddress = t.EOAAddress
		fmt.Printf("Using EOA address from command line: %s\n", eoaAddress)
	} else {
		fmt.Println("Please provide your EOA address to start monitoring...")
		var err error
		eoaAddress, err = promptForEOAAddress()
		if err != nil {
			return fmt.Errorf("failed to get EOA address: %w", err)
		}
	}
	t.UserEOAAddress = eoaAddress

	// Fetch peer IDs for the EOA address
	fmt.Printf("Fetching peer IDs for address: %s\n", eoaAddress)
	peerIDs, err := t.GetPeerIDs(eoaAddress)
	if err != nil {
		return fmt.Errorf("failed to fetch peer IDs: %w", err)
	}
	t.PeerIDs = peerIDs

	fmt.Printf("Successfully loaded %d peer IDs for monitoring\n", len(peerIDs))

	// Send welcome message if not already sent
	if !t.Config.WelcomeSent {
		fmt.Println("Sending welcome message...")
		if err := t.sendWelcomeMessage(""); err != nil {
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
	}

	// Load previous data from persistent storage
	previousData, err := t.loadPreviousData()
	if err != nil {
		fmt.Printf("Warning: Could not load previous data: %v\n", err)
		previousData = &PreviousData{Votes: big.NewInt(0), Rewards: big.NewInt(0), Peers: make(map[string]PeerPreviousData)}
	} else {
		fmt.Printf("Loaded previous data - Votes: %s, Rewards: %s, Last Check: %s\n",
			previousData.Votes.String(), previousData.Rewards.String(), previousData.LastCheck.Format("2006-01-02 15:04:05"))
	}

	fmt.Println("Starting continuous monitoring loop (checking every hour)...")
	fmt.Println("Press Ctrl+C to stop monitoring")

	// Start message polling in a separate goroutine
	go t.startMessagePolling()

	// Start the monitoring loop
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Do initial check
	if err := t.checkAndNotifyWithPeerIDs(previousData); err != nil {
		fmt.Printf("Error in initial check: %v\n", err)
	}

	// Continuous monitoring loop
	for {
		select {
		case <-ticker.C:
			if err := t.checkAndNotifyWithPeerIDs(previousData); err != nil {
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

// checkAndNotifyWithPeerIDs checks blockchain data for all peer IDs and sends notification if there are changes
func (t *TelegramService) checkAndNotifyWithPeerIDs(previousData *PreviousData) error {
	fmt.Printf("\n[%s] Checking blockchain data for %d peer IDs...\n", time.Now().Format("2006-01-02 15:04:05"), len(t.PeerIDs))

	// Convert chat_id to number for API calls
	var chatIDNum int64
	if _, err := fmt.Sscanf(t.Config.ChatID, "%d", &chatIDNum); err != nil {
		return fmt.Errorf("failed to parse chat ID as number: %w", err)
	}

	// Check verification status first
	fmt.Printf("Checking verification status for Telegram ID: %d\n", chatIDNum)
	verificationStatus, err := t.CheckVerificationStatus(chatIDNum)
	if err != nil {
		fmt.Printf("Warning: Could not check verification status: %v\n", err)
		// Continue without verification data
		verificationStatus = nil
	}

	var userRankData *UserRankData
	if verificationStatus != nil && verificationStatus.IsVerified {
		fmt.Printf("User is verified!\n")

		// Get rank data for verified user
		fmt.Printf("Fetching rank data for verified user...\n")
		userRankData, err = t.GetUserRankData(chatIDNum)
		if err != nil {
			fmt.Printf("Warning: Could not get user rank data: %v\n", err)
			// Continue without rank data
			userRankData = nil
		} else {
			fmt.Printf("Successfully fetched rank data:\n")
			fmt.Printf("  - Total ranks: %d\n", len(userRankData.Ranks))
			fmt.Printf("  - Stats: TotalNodes=%d, RankedNodes=%d\n", userRankData.Stats.TotalNodes, userRankData.Stats.RankedNodes)
			for i, rank := range userRankData.Ranks {
				fmt.Printf("  - Rank %d: PeerID=%s, Rank=%d, Wins=%d, Rewards=%d\n",
					i+1, rank.PeerID, rank.Rank, rank.TotalWins, rank.TotalRewards)
			}
		}
	} else {
		fmt.Printf("User is not verified or verification check failed\n")
	}

	var totalVotes *big.Int = big.NewInt(0)
	var totalRewards *big.Int = big.NewInt(0)
	var peerData []struct {
		PeerID  string
		Votes   *big.Int
		Rewards *big.Int
		Rank    *Rank
	}

	// Check each peer ID with rate limiting (1 second delay between requests)
	for i, peerID := range t.PeerIDs {
		fmt.Printf("Checking peer ID %d/%d: %s\n", i+1, len(t.PeerIDs), peerID)

		// Query blockchain data for this peer ID
		blockchainData, err := t.GetBlockchainDataForPeerID(peerID, previousData)
		if err != nil {
			fmt.Printf("Warning: Could not get blockchain data for peer ID %s: %v\n", peerID, err)
			continue
		}

		// Add to totals
		totalVotes.Add(totalVotes, blockchainData.Votes)
		totalRewards.Add(totalRewards, blockchainData.Rewards)

		// Find rank data for this peer if available
		var rankData *Rank
		if userRankData != nil {
			fmt.Printf("DEBUG: Looking for rank data for peer ID: %s\n", peerID)
			fmt.Printf("DEBUG: Available ranks: %d\n", len(userRankData.Ranks))
			for i, rank := range userRankData.Ranks {
				fmt.Printf("DEBUG: Rank %d: PeerID=%s\n", i+1, rank.PeerID)
				if rank.PeerID == peerID {
					rankData = &rank
					fmt.Printf("DEBUG: Found matching rank data for peer %s: Rank=%d\n", peerID, rank.Rank)
					break
				}
			}
			if rankData == nil {
				fmt.Printf("DEBUG: No rank data found for peer %s\n", peerID)
			}
		}

		// Store per-peer data
		peerData = append(peerData, struct {
			PeerID  string
			Votes   *big.Int
			Rewards *big.Int
			Rank    *Rank
		}{
			PeerID:  peerID,
			Votes:   blockchainData.Votes,
			Rewards: blockchainData.Rewards,
			Rank:    rankData,
		})

		// Rate limiting: 1 second delay between requests
		if i < len(t.PeerIDs)-1 { // Don't delay after the last request
			time.Sleep(1 * time.Second)
		}
	}

	// Check if there are any changes
	votesChanged := totalVotes.Cmp(previousData.Votes) != 0
	rewardsChanged := totalRewards.Cmp(previousData.Rewards) != 0

	if votesChanged || rewardsChanged {
		fmt.Printf("Changes detected!\n")
		fmt.Printf("Previous - Votes: %s, Rewards: %s\n", previousData.Votes.String(), previousData.Rewards.String())
		fmt.Printf("Current  - Votes: %s, Rewards: %s\n", totalVotes.String(), totalRewards.String())

		// Build per-peer breakdown
		var peerBreakdown strings.Builder
		if previousData.Peers == nil {
			previousData.Peers = make(map[string]PeerPreviousData)
		}
		for i, data := range peerData {
			// Truncate the peer ID for better readability
			peerID := data.PeerID
			if len(peerID) > 20 {
				peerID = peerID[:3] + "..." + peerID[len(peerID)-3:]
			}

			// Get previous values for this peer (if any)
			prevPeer := previousData.Peers[data.PeerID]
			prevVotes := new(big.Int) // Defaults to 0
			if prevPeer.Votes != "" {
				prevVotes.SetString(prevPeer.Votes, 10)
			}
			prevRewards := new(big.Int) // Defaults to 0
			if prevPeer.Rewards != "" {
				prevRewards.SetString(prevPeer.Rewards, 10)
			}

			// Compute deltas - ALWAYS use Alchemy API data for blockchain calculations
			votesDelta := getChangeIndicator(prevVotes, data.Votes)
			rewardsDelta := getChangeIndicator(prevRewards, data.Rewards)

			peerBreakdown.WriteString(fmt.Sprintf("🔹 <b>Peer %d:</b> %s\n", i+1, peerID))
			peerBreakdown.WriteString(fmt.Sprintf("   📈 Votes: %s %s\n", data.Votes.String(), votesDelta))
			peerBreakdown.WriteString(fmt.Sprintf("   💰 Rewards: %s %s\n", data.Rewards.String(), rewardsDelta))

			// Add rank information if available
			if data.Rank != nil {
				peerBreakdown.WriteString(fmt.Sprintf("   🏆 Rank: #%d\n", data.Rank.Rank))
			}
			peerBreakdown.WriteString("\n")
		}

		// Build verification status section
		t.messageMutex.Lock()
		var verificationSection strings.Builder
		if verificationStatus != nil && verificationStatus.IsVerified {
			verificationSection.WriteString("✅ <b>Verified User</b>\n")

			if userRankData != nil {
				verificationSection.WriteString(fmt.Sprintf("📊 Total Nodes: %d\n", userRankData.Stats.TotalNodes))
				verificationSection.WriteString(fmt.Sprintf("🏆 Ranked Nodes: %d\n", userRankData.Stats.RankedNodes))
			}
			verificationSection.WriteString("\n")
		} else {
			verificationSection.WriteString("❌ <b>Not Verified</b>\n")
			verificationSection.WriteString("🔗 Get verified in Discord to see rank data!\n")
			verificationSection.WriteString("💬 Use <code>/link-telegram</code> in the gensyn discord channel\n\n")
		}
		t.messageMutex.Unlock()

		// Prepare notification message
		message := fmt.Sprintf(`🚀 <b>G-Swarm Update</b>

%s📊 <b>Blockchain Data Update</b>

👤 <b>EOA Address:</b> <code>%s</code>
🔍 <b>Peer IDs Monitored:</b> %d

📈 <b>Total Votes:</b> %s %s
💰 <b>Total Rewards:</b> %s %s

📋 <b>Per-Peer Breakdown:</b>
%s⏰ <b>Last Check:</b> %s`,
			verificationSection.String(),
			t.UserEOAAddress,
			len(t.PeerIDs),
			totalVotes.String(),
			getChangeIndicator(previousData.Votes, totalVotes),
			totalRewards.String(),
			getChangeIndicator(previousData.Rewards, totalRewards),
			peerBreakdown.String(),
			time.Now().Format("2006-01-02 15:04:05"))

		// Send notification
		if err := t.sendTelegramMessageHTML(message); err != nil {
			fmt.Printf("Failed to send Telegram message: %v\n", err)
		}

		// Update previous data
		previousData.Votes = totalVotes
		previousData.Rewards = totalRewards
		previousData.LastCheck = time.Now()

		// Update per-peer previous data - ALWAYS use Alchemy API data
		for _, data := range peerData {
			peerID := data.PeerID
			votesStr := data.Votes.String()
			rewardsStr := data.Rewards.String() // Always use blockchain rewards from Alchemy
			previousData.Peers[peerID] = PeerPreviousData{
				Votes:   votesStr,
				Rewards: rewardsStr,
			}
		}

		// Save updated data
		if err := t.savePreviousData(previousData); err != nil {
			fmt.Printf("Warning: Could not save previous data: %v\n", err)
		}
	} else {
		fmt.Printf("No changes detected. Votes: %s, Rewards: %s\n", totalVotes.String(), totalRewards.String())
	}

	return nil
}

// GetBlockchainDataForPeerID gets blockchain data for a specific peer ID
// If API calls fail, it preserves the previous values to prevent negative changes
func (t *TelegramService) GetBlockchainDataForPeerID(peerID string, previousData *PreviousData) (*BlockchainData, error) {
	fmt.Printf("Querying blockchain data for peer ID: %s\n", peerID)

	// Get previous values for this peer to preserve them if API calls fail
	var prevVotes *big.Int = big.NewInt(0)
	var prevRewards *big.Int = big.NewInt(0)
	if previousData != nil && previousData.Peers != nil {
		if prevPeer, exists := previousData.Peers[peerID]; exists {
			if prevPeer.Votes != "" {
				prevVotes.SetString(prevPeer.Votes, 10)
			}
			if prevPeer.Rewards != "" {
				prevRewards.SetString(prevPeer.Rewards, 10)
			}
		}
	}

	// Query the GenRL-Swarm contract
	contract := coordAddrGenRL
	var totalVotes *big.Int = nil
	var totalRewards *big.Int = nil

	// For votes, we pass the peer ID directly
	if v, err := t.queryUserVotes(peerID, contract); err == nil && v.Cmp(big.NewInt(0)) > 0 {
		totalVotes = v
		fmt.Printf("Found votes for peer ID %s on contract %s: %s\n", peerID, contract, v.String())
	} else if err != nil {
		fmt.Printf("Warning: Failed to query votes for peer ID %s on contract %s: %v\n", peerID, contract, err)
	}

	// For rewards, we pass the peer ID as part of the array
	peerIds := []string{peerID}
	if r, err := t.queryUserRewards(peerIds, contract); err == nil && r.Cmp(big.NewInt(0)) > 0 {
		totalRewards = r
		fmt.Printf("Found rewards for peer ID %s on contract %s: %s\n", peerID, contract, r.String())
	} else if err != nil {
		fmt.Printf("Warning: Failed to query rewards for peer ID %s on contract %s: %v\n", peerID, contract, err)
	}

	// If API calls failed and we couldn't get new data, preserve previous values
	if totalVotes == nil {
		fmt.Printf("No votes data available for peer ID %s, preserving previous value: %s\n", peerID, prevVotes.String())
		totalVotes = prevVotes
	}
	if totalRewards == nil {
		fmt.Printf("No rewards data available for peer ID %s, preserving previous value: %s\n", peerID, prevRewards.String())
		totalRewards = prevRewards
	}

	// Get ETH balance for the EOA address (only if it's an Ethereum address)
	var balance *big.Int = big.NewInt(0)
	if strings.HasPrefix(t.UserEOAAddress, "0x") && len(t.UserEOAAddress) == 42 {
		if b, err := t.queryUserBalance(t.UserEOAAddress); err == nil {
			balance = b
			fmt.Printf("Found balance for EOA %s: %s\n", t.UserEOAAddress, balance.String())
		} else {
			fmt.Printf("Warning: Failed to get balance for EOA %s: %v\n", t.UserEOAAddress, err)
		}
	} else {
		fmt.Printf("Skipping balance query - not an Ethereum address: %s\n", t.UserEOAAddress)
	}

	return &BlockchainData{
		Votes:   totalVotes,
		Rewards: totalRewards,
		Balance: balance,
	}, nil
}

// queryUserVotes queries the smart contract for user votes using Alchemy API
// Function selector: 0xdfb3c7df
// Function signature: getVoterVoteCount(string memory peerId) public view returns (uint256)
func (t *TelegramService) queryUserVotes(peerId string, contractAddress string) (*big.Int, error) {
	fmt.Printf("Querying votes for peer ID: %s on contract: %s\n", peerId, contractAddress)

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

	// Create the eth_call request
	request := AlchemyRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_call",
		Params: []interface{}{
			map[string]interface{}{
				"data":  data,
				"to":    coordAddrGenRL, // Use the GenRL-Swarm contract
				"value": "0x0",
			},
			"latest",
		},
	}

	// Make the request
	result, err := t.makeAlchemyRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call Alchemy API for votes (peerId: %s): %w", peerId, err)
	}

	// Parse the result
	if resultStr, ok := result.(string); ok {
		if strings.HasPrefix(resultStr, "0x") {
			resultStr = strings.TrimPrefix(resultStr, "0x")
			if len(resultStr) >= 64 {
				votes := new(big.Int)
				votes.SetString(resultStr, 16)
				fmt.Printf("Successfully parsed votes for peer ID %s: %s\n", peerId, votes.String())
				return votes, nil
			}
		}
	}

	fmt.Printf("No valid votes data found for peer ID %s\n", peerId)
	return big.NewInt(0), nil
}

// queryUserRewards queries the smart contract for user rewards using Alchemy API
// Function selector: 0x80c3d97f
// Function signature: getTotalRewards(string[] memory peerIds) public view returns (int256[])
func (t *TelegramService) queryUserRewards(peerIds []string, contractAddress string) (*big.Int, error) {
	fmt.Printf("Querying rewards for peer IDs: %v on contract: %s\n", peerIds, contractAddress)

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

	// Create the eth_call request
	request := AlchemyRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_call",
		Params: []interface{}{
			map[string]interface{}{
				"data":  data,
				"to":    coordAddrGenRL, // Use the GenRL-Swarm contract
				"value": "0x0",
			},
			"latest",
		},
	}

	// Make the request
	result, err := t.makeAlchemyRequest(request)
	if err != nil {
		return nil, fmt.Errorf("failed to call Alchemy API for rewards (peerIds: %v): %w", peerIds, err)
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
					fmt.Printf("Successfully parsed rewards for peer IDs %v: %s\n", peerIds, rewards.String())
					return rewards, nil
				}
			}
		}
	}

	fmt.Printf("No valid rewards data found for peer IDs %v\n", peerIds)
	return big.NewInt(0), nil
}

// queryUserBalance queries the user's ETH balance using Alchemy API
func (t *TelegramService) queryUserBalance(userAddress string) (*big.Int, error) {
	fmt.Printf("Querying balance for address: %s\n", userAddress)

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
		return nil, fmt.Errorf("failed to call Alchemy API for balance (address: %s): %w", userAddress, err)
	}

	// Parse the result
	if resultStr, ok := result.(string); ok {
		if strings.HasPrefix(resultStr, "0x") {
			resultStr = strings.TrimPrefix(resultStr, "0x")
			balance := new(big.Int)
			balance.SetString(resultStr, 16)
			fmt.Printf("Successfully parsed balance for address %s: %s\n", userAddress, balance.String())
			return balance, nil
		}
	}

	fmt.Printf("No valid balance data found for address %s\n", userAddress)
	return big.NewInt(0), nil
}

// makeAlchemyRequest makes a request to the Alchemy API with retries and exponential backoff
func (t *TelegramService) makeAlchemyRequest(request AlchemyRequest) (interface{}, error) {
	// Prepare the request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Retry configuration
	maxRetries := 5
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
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
			lastErr = fmt.Errorf("failed to make request: %w", err)
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * baseDelay
				if delay > maxDelay {
					delay = maxDelay
				}
				fmt.Printf("Request failed, retrying in %v (attempt %d/%d): %v\n", delay, attempt+1, maxRetries+1, err)
				time.Sleep(delay)
				continue
			}
			return nil, lastErr
		}
		defer resp.Body.Close()

		// Read the response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * baseDelay
				if delay > maxDelay {
					delay = maxDelay
				}
				fmt.Printf("Failed to read response, retrying in %v (attempt %d/%d): %v\n", delay, attempt+1, maxRetries+1, err)
				time.Sleep(delay)
				continue
			}
			return nil, lastErr
		}

		// Debug: Print the response
		fmt.Printf("Alchemy API Response: %s\n", string(body))

		// Check if response is JSON
		if !strings.HasPrefix(strings.TrimSpace(string(body)), "{") {
			lastErr = fmt.Errorf("non-JSON response from Alchemy API: %s", string(body))
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * baseDelay
				if delay > maxDelay {
					delay = maxDelay
				}
				fmt.Printf("Non-JSON response, retrying in %v (attempt %d/%d)\n", delay, attempt+1, maxRetries+1)
				time.Sleep(delay)
				continue
			}
			return nil, lastErr
		}

		// Parse the response
		var response AlchemyResponse
		if err := json.Unmarshal(body, &response); err != nil {
			lastErr = fmt.Errorf("failed to parse response: %w", err)
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * baseDelay
				if delay > maxDelay {
					delay = maxDelay
				}
				fmt.Printf("Failed to parse response, retrying in %v (attempt %d/%d): %v\n", delay, attempt+1, maxRetries+1, err)
				time.Sleep(delay)
				continue
			}
			return nil, lastErr
		}

		// Check for errors
		if response.Error != nil {
			// Check if it's a rate limit error (code 429 or specific rate limit messages)
			isRateLimit := response.Error.Code == 429 ||
				strings.Contains(strings.ToLower(response.Error.Message), "rate limit") ||
				strings.Contains(strings.ToLower(response.Error.Message), "too many requests")

			if isRateLimit && attempt < maxRetries {
				// Exponential backoff for rate limiting
				delay := time.Duration(1<<uint(attempt)) * baseDelay
				if delay > maxDelay {
					delay = maxDelay
				}
				fmt.Printf("Rate limited, retrying in %v (attempt %d/%d): %s\n", delay, attempt+1, maxRetries+1, response.Error.Message)
				time.Sleep(delay)
				continue
			}

			lastErr = fmt.Errorf("Alchemy API error: %s (code: %d)", response.Error.Message, response.Error.Code)
			if attempt < maxRetries {
				delay := time.Duration(attempt+1) * baseDelay
				if delay > maxDelay {
					delay = maxDelay
				}
				fmt.Printf("API error, retrying in %v (attempt %d/%d): %s\n", delay, attempt+1, maxRetries+1, response.Error.Message)
				time.Sleep(delay)
				continue
			}
			return nil, lastErr
		}

		// Success - return the result
		return response.Result, nil
	}

	return nil, fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// GetBlockchainData queries all blockchain data for a user using Alchemy API
func (t *TelegramService) GetBlockchainData(userAddress string) (*BlockchainData, error) {
	fmt.Printf("Querying blockchain data for address: %s\n", userAddress)

	// Query the GenRL-Swarm contract
	contract := coordAddrGenRL

	var votes *big.Int
	var rewards *big.Int

	// Try to get votes from the contract
	// For votes, we pass the address as a peer ID
	if v, err := t.queryUserVotes(userAddress, contract); err == nil && v.Cmp(big.NewInt(0)) > 0 {
		votes = v
		fmt.Printf("Found votes in contract %s: %s\n", contract, votes.String())
	} else {
		fmt.Printf("No votes found in contract %s: %v\n", contract, err)
	}

	// Try to get rewards from the contract
	// For rewards, we need to pass an array of peer IDs
	peerIds := []string{userAddress} // For now, treat the address as a peer ID
	if r, err := t.queryUserRewards(peerIds, contract); err == nil && r.Cmp(big.NewInt(0)) > 0 {
		rewards = r
		fmt.Printf("Found rewards in contract %s: %s\n", contract, rewards.String())
	} else {
		fmt.Printf("No rewards found in contract %s: %v\n", contract, err)
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
	if data.Peers != nil {
		dataToSave["peers"] = data.Peers
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
				Peers:     make(map[string]PeerPreviousData),
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

	// Parse peers map (optional, for backward compatibility)
	peers := make(map[string]PeerPreviousData)
	if peersRaw, ok := dataMap["peers"]; ok && peersRaw != nil {
		if peersMap, ok := peersRaw.(map[string]interface{}); ok {
			for peerID, v := range peersMap {
				if peerData, ok := v.(map[string]interface{}); ok {
					votesStr, _ := peerData["votes"].(string)
					rewardsStr, _ := peerData["rewards"].(string)
					peers[peerID] = PeerPreviousData{
						Votes:   votesStr,
						Rewards: rewardsStr,
					}
				}
			}
		}
	}

	return &PreviousData{
		Votes:     votes,
		Rewards:   rewards,
		Peers:     peers,
		LastCheck: lastCheck,
	}, nil
}

// sendWelcomeMessage sends a welcome message to new users
func (t *TelegramService) sendWelcomeMessage(code string) error {
	var message string

	if code != "" {
		// Welcome message with verification instructions (for backward compatibility)
		message = fmt.Sprintf(`🤖 <b>Welcome to G-Swarm Monitor!</b>

This bot monitors your Gensyn AI node activity and notifies you when your votes or rewards increase.

<b>Features:</b>
• Monitors votes and rewards every hour
• Sends notifications only when there are changes
• Tracks progress across multiple contracts

<b>Account Linking:</b>
🔗 Join the Gensyn Discord: <a href="https://discord.gg/gensyn">https://discord.gg/gensyn</a>
To link your Discord and Telegram accounts:
1️⃣ Use <code>/link-telegram</code> in Discord to get a code
2️⃣ Use <code>/verify %s</code> here in Telegram
3️⃣ Your accounts will be linked automatically!

⏰ <b>Code expires in 10 minutes</b>

<b>Support Development:</b>
If you find this bot useful, please consider donating to support ongoing development and new features:

ETH: <code>0xA22e20BA3336f5Bd6eCE959F5ac4083C9693e316</code>

Thank you for using G-Swarm Monitor! 🚀`, code)
	} else {
		// Welcome message without verification code
		message = `🤖 <b>Welcome to G-Swarm Monitor!</b>

This bot monitors your Gensyn AI node activity and notifies you when your votes or rewards increase.

<b>Features:</b>
• Monitors votes and rewards every hour
• Sends notifications only when there are changes
• Tracks progress across multiple contracts

<b>Account Linking:</b>
🔗 Join the Gensyn Discord: <a href="https://discord.gg/gensyn">https://discord.gg/gensyn</a>
To link your Discord and Telegram accounts:
1️⃣ Use <code>/link-telegram</code> in Discord to get a code
2️⃣ Use <code>/verify &lt;code&gt;</code> here in Telegram
3️⃣ Your accounts will be linked automatically!

<b>Support Development:</b>
If you find this bot useful, please consider donating to support ongoing development and new features:

ETH: <code>0xA22e20BA3336f5Bd6eCE959F5ac4083C9693e316</code>

Thank you for using G-Swarm Monitor! 🚀`
	}

	return t.sendTelegramMessageHTML(message)
}

// promptForEOAAddress prompts the user for their EOA address
func promptForEOAAddress() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your EOA address (from Gensyn dashboard): ")
	address, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	address = strings.TrimSpace(address)

	if address == "" {
		return "", fmt.Errorf("address cannot be empty")
	}

	return address, nil
}

// GetPeerIDs fetches the peer IDs associated with the given EOA address
func (t *TelegramService) GetPeerIDs(eoaAddress string) ([]string, error) {
	// Use the correct function selector for getPeerId: 0xb894a469
	// Function signature: getPeerId(eoas address[]) returns (string[][])
	// We need to encode an array of addresses

	// Remove 0x prefix if present and pad to 32 bytes
	addressParam := strings.TrimPrefix(eoaAddress, "0x")
	// Pad the address to 32 bytes (64 hex chars)
	addressParam = fmt.Sprintf("%064s", addressParam)

	// For a single address in an array, we need to encode it as:
	// - offset to array data (32 bytes)
	// - array length (32 bytes)
	// - address data (32 bytes)
	offset := "0000000000000000000000000000000000000000000000000000000000000020"      // offset to array data
	arrayLength := "0000000000000000000000000000000000000000000000000000000000000001" // array length = 1
	addressData := addressParam                                                       // the actual address

	// Construct the data field: function selector + encoded array
	data := "0xb894a469" + offset + arrayLength + addressData

	fmt.Printf("Debug: Calling getPeerId with data: %s\n", data)
	fmt.Printf("Debug: Address parameter: %s\n", addressParam)

	// Query the GenRL-Swarm contract
	contract := coordAddrGenRL
	fmt.Printf("Debug: Querying contract: %s\n", contract)

	// Create the eth_call request
	request := AlchemyRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_call",
		Params: []interface{}{
			map[string]interface{}{
				"data":  data,
				"to":    contract,
				"value": "0x0",
			},
			"latest",
		},
	}

	// Make the request
	result, err := t.makeAlchemyRequest(request)
	if err != nil {
		fmt.Printf("Debug: Error with contract %s: %v\n", contract, err)
		return nil, fmt.Errorf("failed to query peer IDs: %w", err)
	}

	// Parse the result
	resultStr, ok := result.(string)
	if !ok {
		fmt.Printf("Debug: Unexpected result type: %T\n", result)
		return nil, fmt.Errorf("unexpected result type from API")
	}

	fmt.Printf("Debug: Got result: %s\n", resultStr)

	// Use ABI-aware decoder to extract peer IDs
	peerIDs, err := decodePeerIDs(resultStr)
	if err != nil {
		fmt.Printf("Debug: Failed to decode peer IDs from contract %s: %v\n", contract, err)
		return nil, fmt.Errorf("failed to decode peer IDs: %w", err)
	}

	if len(peerIDs) > 0 {
		fmt.Printf("Found %d peer IDs for address %s on contract %s\n", len(peerIDs), eoaAddress, contract)
		for i, peerID := range peerIDs {
			fmt.Printf("  %d: %s\n", i+1, peerID)
		}
		return peerIDs, nil
	} else {
		fmt.Printf("Debug: No peer IDs found for this EOA on contract %s\n", contract)
		return nil, fmt.Errorf("no peer IDs found for address: %s", eoaAddress)
	}
}

// getChangeIndicator returns an emoji and the numeric delta if a value changed
// For votes and rewards, we never report negative changes to prevent confusion
func getChangeIndicator(previous, current *big.Int) string {
	cmp := current.Cmp(previous)
	if cmp > 0 {
		delta := new(big.Int).Sub(current, previous)
		return fmt.Sprintf("📈 (+%s)", delta.String())
	} else if cmp < 0 {
		// For votes and rewards, never report negative changes
		// This prevents confusion when API calls fail and return zero values
		return "➡️ (no change)"
	}
	return "➡️ (0)"
}

// decodePeerIDs uses ABI-aware decoding to extract peer IDs from the contract response
func decodePeerIDs(rawHex string) ([]string, error) {
	// strip "0x"
	raw := strings.TrimPrefix(rawHex, "0x")

	// ABI-bytes are big-endian hex
	data, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("hex decode: %w", err)
	}

	// parse ABI
	parsed, err := abi.JSON(strings.NewReader(coordABI))
	if err != nil {
		return nil, fmt.Errorf("ABI parse: %w", err)
	}

	// unpack; returns []interface{} where the first (and only) element is [][]string
	outs, err := parsed.Unpack("getPeerId", data)
	if err != nil {
		return nil, fmt.Errorf("ABI unpack: %w", err)
	}

	// type-assert
	raw2d, ok := outs[0].([][]string)
	if !ok {
		return nil, fmt.Errorf("unexpected output type %T, want [][]string", outs[0])
	}

	// Since you queried with a single address, take raw2d[0]
	if len(raw2d) == 0 {
		return nil, nil // no peer IDs
	}
	return raw2d[0], nil
}

// handleTelegramCommand handles incoming Telegram commands
func (t *TelegramService) handleTelegramCommand(command string, args string, telegramID string) error {
	switch command {
	case "/verify":
		return t.handleVerifyCommand(args, telegramID)
	case "/help":
		return t.sendHelpMessage()
	case "/stats":
		return t.handleStatsCommand(telegramID)
	case "/start":
		// Explicitly ignore /start (no response)
		return nil
	default:
		return t.sendUnknownCommandMessage()
	}
}

// sendHelpMessage sends help information to the user
func (t *TelegramService) sendHelpMessage() error {
	message := `🤖 <b>G-Swarm Bot Commands</b>

<b>Available commands:</b>
<code>/verify &lt;code&gt;</code> - Link your Discord and Telegram accounts
<code>/stats</code> - Check your current stats and rank data
<code>/help</code> - Show this help message

<b>Monitoring:</b>
This bot automatically monitors your Gensyn AI node activity and sends notifications when your votes or rewards change.

<b>Support:</b>
For issues or questions, contact the G-Swarm team in Discord.`

	return t.sendTelegramMessageHTML(message)
}

// handleStatsCommand handles the /stats command to show current user stats
func (t *TelegramService) handleStatsCommand(telegramID string) error {
	fmt.Printf("Handling stats request for Telegram ID: %s\n", telegramID)

	// Load previous data for delta calculations
	previousData, err := t.loadPreviousData()
	if err != nil {
		fmt.Printf("Warning: Could not load previous data: %v\n", err)
		// Initialize empty previous data
		previousData = &PreviousData{
			Votes:   big.NewInt(0),
			Rewards: big.NewInt(0),
			Peers:   make(map[string]PeerPreviousData),
		}
	}

	// Convert chat_id to number for API calls
	var chatIDNum int64
	if _, err := fmt.Sscanf(t.Config.ChatID, "%d", &chatIDNum); err != nil {
		return fmt.Errorf("failed to parse chat ID as number: %w", err)
	}

	// Check verification status first
	fmt.Printf("Checking verification status for Telegram ID: %d\n", chatIDNum)
	verificationStatus, err := t.CheckVerificationStatus(chatIDNum)
	if err != nil {
		fmt.Printf("Warning: Could not check verification status: %v\n", err)
		// Continue without verification data
		verificationStatus = nil
	}

	var userRankData *UserRankData
	if verificationStatus != nil && verificationStatus.IsVerified {
		fmt.Printf("User is verified!\n")

		// Get rank data for verified user
		fmt.Printf("Fetching rank data for verified user...\n")
		userRankData, err = t.GetUserRankData(chatIDNum)
		if err != nil {
			fmt.Printf("Warning: Could not get user rank data: %v\n", err)
			// Continue without rank data
			userRankData = nil
		}
	}

	// Get current blockchain data
	fmt.Printf("Fetching current blockchain data...\n")
	var totalVotes *big.Int = big.NewInt(0)
	var totalRewards *big.Int = big.NewInt(0)
	var peerData []struct {
		PeerID  string
		Votes   *big.Int
		Rewards *big.Int
		Rank    *Rank
	}

	// Check each peer ID
	for i, peerID := range t.PeerIDs {
		fmt.Printf("Checking peer ID %d/%d: %s\n", i+1, len(t.PeerIDs), peerID)

		// Query blockchain data for this peer ID
		blockchainData, err := t.GetBlockchainDataForPeerID(peerID, previousData)
		if err != nil {
			fmt.Printf("Warning: Could not get blockchain data for peer ID %s: %v\n", peerID, err)
			continue
		}

		// Add to totals
		totalVotes.Add(totalVotes, blockchainData.Votes)
		totalRewards.Add(totalRewards, blockchainData.Rewards)

		// Find rank data for this peer if available
		var rankData *Rank
		if userRankData != nil {
			for _, rank := range userRankData.Ranks {
				if rank.PeerID == peerID {
					rankData = &rank
					break
				}
			}
		}

		// Store per-peer data
		peerData = append(peerData, struct {
			PeerID  string
			Votes   *big.Int
			Rewards *big.Int
			Rank    *Rank
		}{
			PeerID:  peerID,
			Votes:   blockchainData.Votes,
			Rewards: blockchainData.Rewards,
			Rank:    rankData,
		})
	}

	// Build verification status section
	var verificationSection strings.Builder
	if verificationStatus != nil && verificationStatus.IsVerified {
		verificationSection.WriteString("✅ <b>Verified User</b>\n")

		if userRankData != nil {
			verificationSection.WriteString(fmt.Sprintf("📊 Total Nodes: %d\n", userRankData.Stats.TotalNodes))
			verificationSection.WriteString(fmt.Sprintf("🏆 Ranked Nodes: %d\n", userRankData.Stats.RankedNodes))
		}
		verificationSection.WriteString("\n")
	} else {
		verificationSection.WriteString("❌ <b>Not Verified</b>\n")
		verificationSection.WriteString("🔗 Get verified in Discord to see rank data!\n")
		verificationSection.WriteString("💬 Use <code>/link-telegram</code> in the gensyn discord channel\n\n")
	}

	// Build per-peer breakdown with delta calculations
	t.messageMutex.Lock()
	var peerBreakdown strings.Builder
	if previousData.Peers == nil {
		previousData.Peers = make(map[string]PeerPreviousData)
	}
	for i, data := range peerData {
		// Truncate the peer ID for better readability
		peerID := data.PeerID
		if len(peerID) > 20 {
			peerID = peerID[:3] + "..." + peerID[len(peerID)-3:]
		}

		// Get previous values for this peer (if any)
		prevPeer := previousData.Peers[data.PeerID]
		prevVotes := new(big.Int)
		if prevPeer.Votes != "" {
			prevVotes.SetString(prevPeer.Votes, 10)
		}
		prevRewards := new(big.Int)
		if prevPeer.Rewards != "" {
			prevRewards.SetString(prevPeer.Rewards, 10)
		}

		// Compute deltas - ALWAYS use Alchemy API data for blockchain calculations
		votesDelta := getChangeIndicator(prevVotes, data.Votes)
		rewardsDelta := getChangeIndicator(prevRewards, data.Rewards)

		peerBreakdown.WriteString(fmt.Sprintf("🔹 <b>Peer %d:</b> %s\n", i+1, peerID))
		peerBreakdown.WriteString(fmt.Sprintf("   📈 Votes: %s %s\n", data.Votes.String(), votesDelta))
		peerBreakdown.WriteString(fmt.Sprintf("   💰 Rewards: %s %s\n", data.Rewards.String(), rewardsDelta))

		// Add rank information if available
		if data.Rank != nil {
			peerBreakdown.WriteString(fmt.Sprintf("   🏆 Rank: #%d\n", data.Rank.Rank))
		}
		peerBreakdown.WriteString("\n")
	}
	t.messageMutex.Unlock()

	// Prepare stats message
	message := fmt.Sprintf(`📊 <b>G-Swarm Stats Report</b>

%s👤 <b>EOA Address:</b> <code>%s</code>
🔍 <b>Peer IDs Monitored:</b> %d

📈 <b>Total Votes:</b> %s %s
💰 <b>Total Rewards:</b> %s %s

📋 <b>Per-Peer Breakdown:</b>
%s⏰ <b>Last Updated:</b> %s

💡 <i>Stats are fetched in real-time. Use this command anytime to check your current status!</i>`,
		verificationSection.String(),
		t.UserEOAAddress,
		len(t.PeerIDs),
		totalVotes.String(),
		getChangeIndicator(previousData.Votes, totalVotes),
		totalRewards.String(),
		getChangeIndicator(previousData.Rewards, totalRewards),
		peerBreakdown.String(),
		time.Now().Format("2006-01-02 15:04:05"))

	// Send stats message
	if err := t.sendTelegramMessageHTML(message); err != nil {
		return fmt.Errorf("failed to send stats message: %w", err)
	}

	fmt.Printf("Stats message sent successfully!\n")
	return nil
}

// sendUnknownCommandMessage sends a message for unknown commands
func (t *TelegramService) sendUnknownCommandMessage() error {
	message := `❓ <b>Unknown Command</b>

Use <code>/help</code> to see available commands.

<b>Quick start:</b>
<code>/link-telegram</code> in Discord to get a code
<code>/verify &lt;code&gt;</code> here in Telegram to link accounts
<code>/stats</code> to check your current stats`

	return t.sendTelegramMessageHTML(message)
}

// sendWelcomeMessageWithInstructions sends a welcome message including the verification code instructions
func (t *TelegramService) sendWelcomeMessageWithInstructions(code string) error {
	message := fmt.Sprintf(`🤖 <b>Welcome to G-Swarm Monitor!</b>

This bot monitors your Gensyn AI node activity and notifies you when your votes or rewards increase.

<b>Features:</b>
• Monitors votes and rewards every hour
• Sends notifications only when there are changes
• Tracks progress across multiple contracts

<b>Get Verified in Discord:</b>
1️⃣ Join the Discord server: <a href="https://discord.gg/gswarm">https://discord.gg/gswarm</a>
2️⃣ Use the verification command:
<b>/verify %s</b>
3️⃣ The bot will automatically assign you the @GSwarm role!

⏰ <b>Code expires in 15 minutes</b>
🔄 <b>Need a new code?</b> Run /verify again

<b>Support Development:</b>
If you find this bot useful, please consider donating to support ongoing development and new features:

ETH: <code>0xA22e20BA3336f5Bd6eCE959F5ac4083C9693e316</code>

Thank you for using G-Swarm Monitor! 🚀`, code)

	return t.sendTelegramMessageHTML(message)
}

// TelegramUpdate represents an incoming Telegram update
type TelegramUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  struct {
		MessageID int64 `json:"message_id"`
		From      struct {
			ID        int64  `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Username  string `json:"username"`
		} `json:"from"`
		Chat struct {
			ID    int64  `json:"id"`
			Type  string `json:"type"`
			Title string `json:"title"`
		} `json:"chat"`
		Date     int64  `json:"date"`
		Text     string `json:"text"`
		Entities []struct {
			Type   string `json:"type"`
			Offset int    `json:"offset"`
			Length int    `json:"length"`
		} `json:"entities"`
	} `json:"message"`
}

// TelegramUpdatesResponse represents the response from getUpdates
type TelegramUpdatesResponse struct {
	OK     bool             `json:"ok"`
	Result []TelegramUpdate `json:"result"`
}

// startMessagePolling starts polling for incoming Telegram messages
func (t *TelegramService) startMessagePolling() {
	fmt.Println("Starting Telegram message polling...")

	var lastUpdateID int64 = 0

	// Poll for updates every 3 seconds
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			updates, err := t.getUpdates(lastUpdateID)
			if err != nil {
				fmt.Printf("Error getting updates: %v\n", err)
				continue
			}

			for _, update := range updates {
				if update.UpdateID > lastUpdateID {
					lastUpdateID = update.UpdateID

					// Handle the message
					if update.Message.Text != "" {
						t.handleIncomingMessage(update.Message.Text, fmt.Sprintf("%d", update.Message.From.ID))
					}
				}
			}
		case <-t.StopChan:
			fmt.Println("Stopping Telegram polling...")
			return
		}
	}
}

// getUpdates fetches updates from Telegram
func (t *TelegramService) getUpdates(offset int64) ([]TelegramUpdate, error) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", t.Config.BotToken)

	// Add offset parameter if provided
	if offset > 0 {
		apiURL += fmt.Sprintf("?offset=%d", offset+1)
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get updates: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response TelegramUpdatesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return nil, fmt.Errorf("Telegram API error: %s", string(body))
	}

	return response.Result, nil
}

// handleIncomingMessage handles incoming messages and routes them to appropriate handlers
func (t *TelegramService) handleIncomingMessage(text string, telegramID string) {
	// Check if it's a command
	if strings.HasPrefix(text, "/") {
		parts := strings.SplitN(text, " ", 2)
		command := parts[0]
		var args string
		if len(parts) > 1 {
			args = strings.TrimSpace(parts[1])
		}

		fmt.Printf("Received command: %s with args: '%s' from Telegram ID: %s\n", command, args, telegramID)

		if err := t.handleTelegramCommand(command, args, telegramID); err != nil {
			fmt.Printf("Error handling command %s: %v\n", command, err)
			// Send error message to user
			errorMsg := fmt.Sprintf("❌ Error processing command: %v", err)
			t.sendTelegramMessageHTML(errorMsg)
		}
	} else {
		// Handle regular messages
		fmt.Printf("Received message: %s from Telegram ID: %s\n", text, telegramID)
	}
}

// handleVerifyCommand handles the /verify command with optional code parameter
func (t *TelegramService) handleVerifyCommand(args string, telegramID string) error {
	if args == "" {
		// No code provided, send instructions
		return t.sendVerificationInstructions("")
	}

	// Code provided, verify it
	return t.verifyLinkingCode(args, telegramID)
}

// verifyLinkingCode verifies a linking code with the API
func (t *TelegramService) verifyLinkingCode(code string, telegramID string) error {
	// Create the request payload
	request := map[string]string{
		"code":       code,
		"telegramId": telegramID,
		"eoa":        t.UserEOAAddress,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Robust API URL construction: ensure only one /api in the final URL
	apiBase := strings.TrimSuffix(t.Config.APIURL, "/")
	apiBase = strings.TrimSuffix(apiBase, "/api")
	url := apiBase + "/api/telegrams/verify-code"
	fmt.Printf("DEBUG: Using API URL: %s\n", url)
	fmt.Printf("DEBUG: Sending payload: %s\n", string(jsonData))

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

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

	// DEBUG: Print raw API response
	fmt.Printf("DEBUG: Raw API response: %s\n", string(body))

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
		// Send error message to user
		errorMsg := fmt.Sprintf("❌ **Verification Failed**\n\n%s", apiResp.Error)
		return t.sendTelegramMessageHTML(errorMsg)
	}

	// Send success message
	successMsg := fmt.Sprintf("✅ **Account Successfully Linked!**\n\n%s\n\n🎉 You can now use both Discord and Telegram to interact with G-Swarm!", apiResp.Message)
	return t.sendTelegramMessageHTML(successMsg)
}

// sendVerificationInstructions sends instructions for verification
func (t *TelegramService) sendVerificationInstructions(code string) error {
	message := `🔗 **Discord-Telegram Account Linking**

To link your Discord and Telegram accounts:

1️⃣ **Get a linking code from Discord:**
   • Join the Discord server: https://discord.gg/gswarm
   • Use the /link-telegram command
   • You'll receive a code via DM

2️⃣ **Verify the code here:**
   • Use /verify <code> with your code
   • Example: /verify a1b2c3

3️⃣ **That's it!** Your accounts will be linked automatically.

⚠️ **Important:**
• Codes expire in 10 minutes
• Each code can only be used once
• Keep your code private`

	return t.sendTelegramMessageHTML(message)
}
