package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Deep-Commit/gswarm/internal/blockchain"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
)

// Version information
var (
	Version   = "1.0.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

// BlockchainListener is a generic blockchain event listener service
// Currently supports Ethereum HFUploadVerified events
// Future: Can be extended to support other blockchains and event types

// YAMLConfig holds the YAML configuration structure
type YAMLConfig struct {
	Ethereum struct {
		RPCURL          string `yaml:"rpc_url"`
		ContractAddress string `yaml:"contract_address"`
		ContractName    string `yaml:"contract_name"`
	} `yaml:"ethereum"`
	Database struct {
		PostgresDSN string `yaml:"postgres_dsn"`
	} `yaml:"database"`
	ABI struct {
		Directory    string `yaml:"directory"`
		ContractName string `yaml:"contract_name"`
	} `yaml:"abi"`
	Events struct {
		MonitoredEvents []string `yaml:"monitored_events"`
	} `yaml:"events"`
	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
}

// Config holds the application configuration
type Config struct {
	RPCURL          string
	ContractAddress common.Address
	ContractName    string
	PostgresDSN     string
	LogLevel        string
	ABIDirectory    string
	MonitoredEvents []string
}

// EventData struct for HFUploadVerified events
type EventData struct {
	User             string `json:"user"`
	TrainingID       string `json:"trainingID"`
	HuggingFaceID    string `json:"huggingFaceID"`
	NumSessions      uint64 `json:"numSessions"`
	TelemetryEnabled bool   `json:"telemetryEnabled"`
	BlockNumber      uint64 `json:"blockNumber"`
}

func main() {
	// Parse command line flags
	var (
		rpcURL          = flag.String("rpc-url", "", "Ethereum RPC URL (e.g., wss://sepolia.alchemyapi.io/v2/<key>)")
		contractAddress = flag.String("contract-address", "", "Contract address to monitor")
		contractName    = flag.String("contract-name", "blockassist", "Contract name (used for ABI lookup)")
		postgresDSN     = flag.String("postgres-dsn", "", "PostgreSQL connection string")
		logLevel        = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		configFile      = flag.String("config", "config.blockchain-listener.yaml", "Configuration file path")
		abiDirectory    = flag.String("abi-dir", "internal/blockchain/abis", "Directory containing ABI files")
		showVersion     = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("Blockchain Listener version %s\n", Version)
		fmt.Printf("Build date: %s\n", BuildDate)
		fmt.Printf("Git commit: %s\n", GitCommit)
		return
	}

	// Load configuration
	config, err := loadConfig(*configFile, *rpcURL, *contractAddress, *contractName, *postgresDSN, *logLevel, *abiDirectory)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate required configuration
	if config.RPCURL == "" {
		log.Fatal("RPC URL is required (set via --rpc-url flag, config file, or RPC_URL environment variable)")
	}
	if config.ContractAddress == (common.Address{}) {
		log.Fatal("Contract address is required (set via --contract-address flag, config file, or CONTRACT_ADDRESS environment variable)")
	}
	if config.PostgresDSN == "" {
		log.Fatal("PostgreSQL DSN is required (set via --postgres-dsn flag, config file, or POSTGRES_DSN environment variable)")
	}

	// Start the event listener
	if err := runEventListener(config); err != nil {
		log.Fatalf("Failed to run event listener: %v", err)
	}
}

func loadConfig(configFile, rpcURL, contractAddress, contractName, postgresDSN, logLevel, abiDirectory string) (*Config, error) {
	config := &Config{}

	// Try to load from YAML config file first
	if configFile != "" {
		if yamlConfig, err := loadYAMLConfig(configFile); err == nil {
			// Use YAML config values
			config.RPCURL = yamlConfig.Ethereum.RPCURL
			config.ContractName = yamlConfig.Ethereum.ContractName
			config.PostgresDSN = yamlConfig.Database.PostgresDSN
			config.LogLevel = yamlConfig.Logging.Level
			config.ABIDirectory = yamlConfig.ABI.Directory
			config.MonitoredEvents = yamlConfig.Events.MonitoredEvents

			// Parse contract address from YAML
			if yamlConfig.Ethereum.ContractAddress != "" {
				if !common.IsHexAddress(yamlConfig.Ethereum.ContractAddress) {
					return nil, fmt.Errorf("invalid contract address in YAML: %s", yamlConfig.Ethereum.ContractAddress)
				}
				config.ContractAddress = common.HexToAddress(yamlConfig.Ethereum.ContractAddress)
			}
		} else {
			log.Printf("Warning: Failed to load YAML config from %s: %v", configFile, err)
		}
	}

	// Override with environment variables (environment takes precedence)
	config.RPCURL = getEnvOrDefault("RPC_URL", config.RPCURL)
	config.ContractName = getEnvOrDefault("CONTRACT_NAME", config.ContractName)
	config.PostgresDSN = getEnvOrDefault("POSTGRES_DSN", config.PostgresDSN)
	config.LogLevel = getEnvOrDefault("LOG_LEVEL", config.LogLevel)
	config.ABIDirectory = getEnvOrDefault("ABI_DIRECTORY", config.ABIDirectory)

	// Parse contract address from environment or CLI
	if contractAddr := getEnvOrDefault("CONTRACT_ADDRESS", contractAddress); contractAddr != "" {
		if !common.IsHexAddress(contractAddr) {
			return nil, fmt.Errorf("invalid contract address: %s", contractAddr)
		}
		config.ContractAddress = common.HexToAddress(contractAddr)
	}

	// Load monitored events from environment variable (overrides YAML)
	if monitoredEvents := os.Getenv("MONITORED_EVENTS"); monitoredEvents != "" {
		// Split by comma and trim whitespace
		events := strings.Split(monitoredEvents, ",")
		config.MonitoredEvents = nil // Clear YAML events
		for _, event := range events {
			event = strings.TrimSpace(event)
			if event != "" {
				config.MonitoredEvents = append(config.MonitoredEvents, event)
			}
		}
	}

	// Set defaults if not specified anywhere
	if config.RPCURL == "" {
		config.RPCURL = rpcURL
	}
	if config.ContractName == "" {
		config.ContractName = contractName
	}
	if config.PostgresDSN == "" {
		config.PostgresDSN = postgresDSN
	}
	if config.LogLevel == "" {
		config.LogLevel = logLevel
	}
	if config.ABIDirectory == "" {
		config.ABIDirectory = abiDirectory
	}
	if len(config.MonitoredEvents) == 0 {
		config.MonitoredEvents = []string{"HFUploadSubmitted", "HFUploadVerified", "HFUploadRejected"}
	}

	return config, nil
}

func loadYAMLConfig(configFile string) (*YAMLConfig, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var yamlConfig YAMLConfig
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &yamlConfig, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func runEventListener(config *Config) error {
	// Connect to PostgreSQL
	pool, err := pgxpool.New(context.Background(), config.PostgresDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer pool.Close()

	// Create tables if they don't exist
	if err := createTables(pool); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Initialize ABI manager
	abiManager := blockchain.NewABIManager()

	// Load ABIs from directory
	if err := abiManager.LoadABIsFromDirectory(config.ABIDirectory); err != nil {
		return fmt.Errorf("failed to load ABIs: %w", err)
	}

	// List loaded contracts
	contracts := abiManager.ListContracts()
	log.Printf("Loaded contracts: %v", contracts)

	// Initialize event processor
	eventProcessor := blockchain.NewEventProcessor(abiManager, pool)

	// Connect to Ethereum
	client, err := ethclient.Dial(config.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to Ethereum: %w", err)
	}
	defer client.Close()

	log.Printf("Connected to Ethereum network at %s", config.RPCURL)
	log.Printf("Monitoring contract: %s (%s)", config.ContractAddress.Hex(), config.ContractName)

	// Start event listener with reconnection
	go func() {
		for {
			if err := listenEvents(client, pool, config, eventProcessor); err != nil {
				log.Printf("Event listener error: %v", err)
			}
			log.Println("WebSocket disconnected. Reconnecting in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}()

	// Wait for interrupt signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	log.Println("Shutting down...")

	return nil
}

func createTables(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS events (
			user_address TEXT NOT NULL,
			training_id TEXT NOT NULL,
			hugging_face_id TEXT NOT NULL,
			num_sessions BIGINT NOT NULL,
			telemetry_enabled BOOLEAN NOT NULL,
			block_number BIGINT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY (user_address, training_id)
		);
		CREATE TABLE IF NOT EXISTS hf_upload_submitted (
			user_address TEXT NOT NULL,
			training_id TEXT NOT NULL,
			hugging_face_id TEXT NOT NULL,
			num_sessions BIGINT NOT NULL,
			telemetry_enabled BOOLEAN NOT NULL,
			block_number BIGINT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY (user_address, training_id)
		);
		CREATE TABLE IF NOT EXISTS hf_upload_rejected (
			user_address TEXT NOT NULL,
			training_id TEXT NOT NULL,
			hugging_face_id TEXT NOT NULL,
			num_sessions BIGINT NOT NULL,
			telemetry_enabled BOOLEAN NOT NULL,
			block_number BIGINT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY (user_address, training_id)
		);
		CREATE TABLE IF NOT EXISTS training_completions (
			user_address TEXT NOT NULL,
			training_id TEXT NOT NULL,
			result TEXT NOT NULL,
			timestamp BIGINT NOT NULL,
			block_number BIGINT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY (user_address, training_id)
		);
		CREATE TABLE IF NOT EXISTS rewards (
			user_address TEXT NOT NULL,
			amount BIGINT NOT NULL,
			training_id TEXT NOT NULL,
			block_number BIGINT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY (user_address, training_id)
		);
		CREATE TABLE IF NOT EXISTS generic_events (
			id SERIAL PRIMARY KEY,
			event_name TEXT NOT NULL,
			event_data JSONB NOT NULL,
			block_number BIGINT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value BIGINT,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS verified_users (
			discord_id TEXT PRIMARY KEY,
			user_address TEXT NOT NULL,
			training_id TEXT,
			hugging_face_id TEXT,
			verified_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(user_address, training_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Initialize last_processed_block if not exists
	_, err = pool.Exec(context.Background(), `
		INSERT INTO metadata (key, value) VALUES ('last_processed_block', 0)
		ON CONFLICT (key) DO NOTHING;
	`)
	if err != nil {
		return fmt.Errorf("failed to initialize metadata: %w", err)
	}

	log.Println("Database tables initialized successfully")
	return nil
}

func listenEvents(client *ethclient.Client, pool *pgxpool.Pool, config *Config, eventProcessor *blockchain.EventProcessor) error {
	// Backfill missed events on (re)connect
	if err := backfillEvents(client, pool, config, eventProcessor); err != nil {
		log.Printf("Failed to backfill events: %v", err)
	}

	// Listen for configured events
	abiManager := eventProcessor.GetABIManager()

	// Get signatures for all monitored events
	var eventSignatures []common.Hash
	for _, eventName := range config.MonitoredEvents {
		sig, err := abiManager.GetEventSignature(config.ContractName, eventName)
		if err != nil {
			return fmt.Errorf("failed to get %s signature: %w", eventName, err)
		}
		eventSignatures = append(eventSignatures, crypto.Keccak256Hash([]byte(sig)))
	}

	query := ethereum.FilterQuery{
		Addresses: []common.Address{config.ContractAddress},
		Topics:    [][]common.Hash{eventSignatures},
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		return fmt.Errorf("failed to subscribe to logs: %w", err)
	}

	log.Printf("Started listening for events: %v from contract %s", config.MonitoredEvents, config.ContractName)

	for {
		select {
		case err := <-sub.Err():
			return fmt.Errorf("subscription error: %w", err)
		case vLog := <-logs:
			if err := eventProcessor.ProcessLog(vLog, config.ContractName); err != nil {
				log.Printf("Failed to process log: %v", err)
			} else {
				// Update last processed block for live events
				if err := updateLastProcessedBlock(pool, vLog.BlockNumber); err != nil {
					log.Printf("Failed to update last processed block: %v", err)
				}
			}
		}
	}
}

func backfillEvents(client *ethclient.Client, pool *pgxpool.Pool, config *Config, eventProcessor *blockchain.EventProcessor) error {
	var lastBlock uint64
	err := pool.QueryRow(context.Background(), "SELECT value FROM metadata WHERE key = 'last_processed_block'").Scan(&lastBlock)
	if err != nil {
		return fmt.Errorf("failed to get last block: %w", err)
	}

	currentBlock, err := client.BlockNumber(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get current block: %w", err)
	}

	if lastBlock >= currentBlock {
		return nil
	}

	log.Printf("Backfilling from block %d to %d", lastBlock+1, currentBlock)

	// Backfill configured events
	abiManager := eventProcessor.GetABIManager()

	// Get signatures for all monitored events
	var eventSignatures []common.Hash
	for _, eventName := range config.MonitoredEvents {
		sig, err := abiManager.GetEventSignature(config.ContractName, eventName)
		if err != nil {
			return fmt.Errorf("failed to get %s signature: %w", eventName, err)
		}
		eventSignatures = append(eventSignatures, crypto.Keccak256Hash([]byte(sig)))
	}

	topics := [][]common.Hash{eventSignatures}

	// Process in 50-block chunks to avoid overwhelming the API
	const maxBlockRange = 50
	fromBlock := lastBlock + 1

	for fromBlock < currentBlock {
		toBlock := fromBlock + maxBlockRange - 1
		if toBlock > currentBlock {
			toBlock = currentBlock
		}

		log.Printf("Backfilling blocks %d to %d", fromBlock, toBlock)

		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(fromBlock)),
			ToBlock:   big.NewInt(int64(toBlock)),
			Addresses: []common.Address{config.ContractAddress},
			Topics:    topics,
		}

		logs, err := client.FilterLogs(context.Background(), query)
		if err != nil {
			return fmt.Errorf("failed to filter logs for blocks %d-%d: %w", fromBlock, toBlock, err)
		}

		for _, vLog := range logs {
			if err := eventProcessor.ProcessLog(vLog, config.ContractName); err != nil {
				log.Printf("Failed to process backfill log: %v", err)
			}
		}

		log.Printf("Backfill complete for blocks %d-%d: processed %d events", fromBlock, toBlock, len(logs))

		// Update the last processed block after each chunk
		if err := updateLastProcessedBlock(pool, toBlock); err != nil {
			log.Printf("Failed to update last processed block: %v", err)
		}

		fromBlock = toBlock + 1
	}

	return nil
}

func updateLastProcessedBlock(pool *pgxpool.Pool, blockNumber uint64) error {
	tx, err := pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), `
		UPDATE metadata SET value = $1, updated_at = NOW() WHERE key = 'last_processed_block';
	`, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to update last processed block: %w", err)
	}

	if err = tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Updated last processed block to %d", blockNumber)
	return nil
}
