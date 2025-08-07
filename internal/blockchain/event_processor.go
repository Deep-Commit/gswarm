package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventProcessor handles processing of blockchain events
type EventProcessor struct {
	abiManager *ABIManager
	pool       *pgxpool.Pool
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(abiManager *ABIManager, pool *pgxpool.Pool) *EventProcessor {
	return &EventProcessor{
		abiManager: abiManager,
		pool:       pool,
	}
}

// ProcessLog processes a single log entry
func (ep *EventProcessor) ProcessLog(vLog types.Log, contractName string) error {
	// Get the ABI for the contract
	contractABI, err := ep.abiManager.GetABI(contractName)
	if err != nil {
		return fmt.Errorf("failed to get ABI for contract %s: %w", contractName, err)
	}

	// Try to identify the event from the log
	eventName, err := ep.identifyEvent(vLog, contractABI)
	if err != nil {
		return fmt.Errorf("failed to identify event: %w", err)
	}

	// Process based on event type
	switch eventName {
	case "HFUploadVerified":
		return ep.processHFUploadVerified(vLog, contractABI)
	case "HFUploadSubmitted":
		return ep.processHFUploadSubmitted(vLog, contractABI)
	case "HFUploadRejected":
		return ep.processHFUploadRejected(vLog, contractABI)
	case "TrainingCompleted":
		return ep.processTrainingCompleted(vLog, contractABI)
	case "RewardDistributed":
		return ep.processRewardDistributed(vLog, contractABI)
	case "Initialized":
		return ep.processInitialized(vLog, contractABI)
	case "Upgraded":
		return ep.processUpgraded(vLog, contractABI)
	default:
		return ep.processGenericEvent(vLog, contractABI, eventName)
	}
}

// identifyEvent identifies which event the log represents
func (ep *EventProcessor) identifyEvent(vLog types.Log, contractABI *abi.ABI) (string, error) {
	// The first topic is the event signature
	if len(vLog.Topics) == 0 {
		return "", fmt.Errorf("no topics in log")
	}

	eventSignature := vLog.Topics[0].Hex()

	// Find the event that matches this signature
	for eventName, event := range contractABI.Events {
		// Calculate the proper event signature hash
		sig := event.Sig
		hash := crypto.Keccak256Hash([]byte(sig))
		hashHex := hash.Hex()

		if hashHex == eventSignature {
			return eventName, nil
		}
	}

	return "", fmt.Errorf("unknown event signature: %s", eventSignature)
}

// processHFUploadVerified processes HFUploadVerified events
func (ep *EventProcessor) processHFUploadVerified(vLog types.Log, contractABI *abi.ABI) error {
	var event struct {
		User             string   `json:"user"`
		TrainingID       string   `json:"trainingID"`
		HuggingFaceID    string   `json:"huggingFaceID"`
		NumSessions      *big.Int `json:"numSessions"`
		TelemetryEnabled bool     `json:"telemetryEnabled"`
		BlockNumber      uint64   `json:"blockNumber"`
	}

	// Unpack the event data
	err := contractABI.UnpackIntoInterface(&event, "HFUploadVerified", vLog.Data)
	if err != nil {
		return fmt.Errorf("failed to unpack HFUploadVerified event: %w", err)
	}

	// Extract user address from indexed topic
	event.User = common.BytesToAddress(vLog.Topics[1].Bytes()).Hex()
	event.BlockNumber = vLog.BlockNumber

	// Insert into database
	_, err = ep.pool.Exec(context.Background(), `
		INSERT INTO events (user_address, training_id, hugging_face_id, num_sessions, telemetry_enabled, block_number)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_address, training_id) DO NOTHING;
	`, strings.ToLower(event.User), event.TrainingID, event.HuggingFaceID, event.NumSessions.Uint64(), event.TelemetryEnabled, event.BlockNumber)
	if err != nil {
		return fmt.Errorf("failed to insert HFUploadVerified event: %w", err)
	}

	// Update last processed block
	if err := ep.updateLastProcessedBlock(event.BlockNumber); err != nil {
		return fmt.Errorf("failed to update last processed block: %w", err)
	}

	// Log the event
	eventJSON, _ := json.Marshal(event)
	log.Printf("Processed HFUploadVerified event: %s", string(eventJSON))

	return nil
}

// processHFUploadSubmitted processes HFUploadSubmitted events
func (ep *EventProcessor) processHFUploadSubmitted(vLog types.Log, contractABI *abi.ABI) error {
	var event struct {
		User             string   `json:"user"`
		TrainingID       string   `json:"trainingID"`
		HuggingFaceID    string   `json:"huggingFaceID"`
		NumSessions      *big.Int `json:"numSessions"`
		TelemetryEnabled bool     `json:"telemetryEnabled"`
		BlockNumber      uint64   `json:"blockNumber"`
	}

	// Unpack the event data
	err := contractABI.UnpackIntoInterface(&event, "HFUploadSubmitted", vLog.Data)
	if err != nil {
		return fmt.Errorf("failed to unpack HFUploadSubmitted event: %w", err)
	}

	// Extract user address from indexed topic
	event.User = common.BytesToAddress(vLog.Topics[1].Bytes()).Hex()
	event.BlockNumber = vLog.BlockNumber

	// Insert into database
	_, err = ep.pool.Exec(context.Background(), `
		INSERT INTO hf_upload_submitted (user_address, training_id, hugging_face_id, num_sessions, telemetry_enabled, block_number)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_address, training_id) DO NOTHING;
	`, strings.ToLower(event.User), event.TrainingID, event.HuggingFaceID, event.NumSessions.Uint64(), event.TelemetryEnabled, event.BlockNumber)
	if err != nil {
		return fmt.Errorf("failed to insert HFUploadSubmitted event: %w", err)
	}

	// Update last processed block
	if err := ep.updateLastProcessedBlock(event.BlockNumber); err != nil {
		return fmt.Errorf("failed to update last processed block: %w", err)
	}

	// Log the event
	eventJSON, _ := json.Marshal(event)
	log.Printf("Processed HFUploadSubmitted event: %s", string(eventJSON))

	return nil
}

// processHFUploadRejected processes HFUploadRejected events
func (ep *EventProcessor) processHFUploadRejected(vLog types.Log, contractABI *abi.ABI) error {
	var event struct {
		User             string   `json:"user"`
		TrainingID       string   `json:"trainingID"`
		HuggingFaceID    string   `json:"huggingFaceID"`
		NumSessions      *big.Int `json:"numSessions"`
		TelemetryEnabled bool     `json:"telemetryEnabled"`
		BlockNumber      uint64   `json:"blockNumber"`
	}

	// Unpack the event data
	err := contractABI.UnpackIntoInterface(&event, "HFUploadRejected", vLog.Data)
	if err != nil {
		return fmt.Errorf("failed to unpack HFUploadRejected event: %w", err)
	}

	// Extract user address from indexed topic
	event.User = common.BytesToAddress(vLog.Topics[1].Bytes()).Hex()
	event.BlockNumber = vLog.BlockNumber

	// Insert into database
	_, err = ep.pool.Exec(context.Background(), `
		INSERT INTO hf_upload_rejected (user_address, training_id, hugging_face_id, num_sessions, telemetry_enabled, block_number)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_address, training_id) DO NOTHING;
	`, strings.ToLower(event.User), event.TrainingID, event.HuggingFaceID, event.NumSessions.Uint64(), event.TelemetryEnabled, event.BlockNumber)
	if err != nil {
		return fmt.Errorf("failed to insert HFUploadRejected event: %w", err)
	}

	// Update last processed block
	if err := ep.updateLastProcessedBlock(event.BlockNumber); err != nil {
		return fmt.Errorf("failed to update last processed block: %w", err)
	}

	// Log the event
	eventJSON, _ := json.Marshal(event)
	log.Printf("Processed HFUploadRejected event: %s", string(eventJSON))

	return nil
}

// processTrainingCompleted processes TrainingCompleted events
func (ep *EventProcessor) processTrainingCompleted(vLog types.Log, contractABI *abi.ABI) error {
	var event struct {
		User        string `json:"user"`
		TrainingID  string `json:"trainingID"`
		Result      string `json:"result"`
		Timestamp   uint64 `json:"timestamp"`
		BlockNumber uint64 `json:"blockNumber"`
	}

	// Unpack the event data
	err := contractABI.UnpackIntoInterface(&event, "TrainingCompleted", vLog.Data)
	if err != nil {
		return fmt.Errorf("failed to unpack TrainingCompleted event: %w", err)
	}

	// Extract user address from indexed topic
	event.User = common.BytesToAddress(vLog.Topics[1].Bytes()).Hex()
	event.BlockNumber = vLog.BlockNumber

	// Insert into database (you might want to create a separate table for training completions)
	_, err = ep.pool.Exec(context.Background(), `
		INSERT INTO training_completions (user_address, training_id, result, timestamp, block_number)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_address, training_id) DO UPDATE SET
			result = EXCLUDED.result,
			timestamp = EXCLUDED.timestamp,
			block_number = EXCLUDED.block_number;
	`, strings.ToLower(event.User), event.TrainingID, event.Result, event.Timestamp, event.BlockNumber)
	if err != nil {
		return fmt.Errorf("failed to insert TrainingCompleted event: %w", err)
	}

	// Update last processed block
	if err := ep.updateLastProcessedBlock(event.BlockNumber); err != nil {
		return fmt.Errorf("failed to update last processed block: %w", err)
	}

	// Log the event
	eventJSON, _ := json.Marshal(event)
	log.Printf("Processed TrainingCompleted event: %s", string(eventJSON))

	return nil
}

// processRewardDistributed processes RewardDistributed events
func (ep *EventProcessor) processRewardDistributed(vLog types.Log, contractABI *abi.ABI) error {
	var event struct {
		User        string `json:"user"`
		Amount      uint64 `json:"amount"`
		TrainingID  string `json:"trainingID"`
		BlockNumber uint64 `json:"blockNumber"`
	}

	// Unpack the event data
	err := contractABI.UnpackIntoInterface(&event, "RewardDistributed", vLog.Data)
	if err != nil {
		return fmt.Errorf("failed to unpack RewardDistributed event: %w", err)
	}

	// Extract user address from indexed topic
	event.User = common.BytesToAddress(vLog.Topics[1].Bytes()).Hex()
	event.BlockNumber = vLog.BlockNumber

	// Insert into database (you might want to create a separate table for rewards)
	_, err = ep.pool.Exec(context.Background(), `
		INSERT INTO rewards (user_address, amount, training_id, block_number)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_address, training_id) DO UPDATE SET
			amount = EXCLUDED.amount,
			block_number = EXCLUDED.block_number;
	`, strings.ToLower(event.User), event.Amount, event.TrainingID, event.BlockNumber)
	if err != nil {
		return fmt.Errorf("failed to insert RewardDistributed event: %w", err)
	}

	// Update last processed block
	if err := ep.updateLastProcessedBlock(event.BlockNumber); err != nil {
		return fmt.Errorf("failed to update last processed block: %w", err)
	}

	// Log the event
	eventJSON, _ := json.Marshal(event)
	log.Printf("Processed RewardDistributed event: %s", string(eventJSON))

	return nil
}

// processInitialized processes Initialized events
func (ep *EventProcessor) processInitialized(vLog types.Log, contractABI *abi.ABI) error {
	var event struct {
		BlockNumber uint64 `json:"blockNumber"`
	}

	// Unpack the event data
	err := contractABI.UnpackIntoInterface(&event, "Initialized", vLog.Data)
	if err != nil {
		return fmt.Errorf("failed to unpack Initialized event: %w", err)
	}

	event.BlockNumber = vLog.BlockNumber

	// Insert into database
	_, err = ep.pool.Exec(context.Background(), `
		INSERT INTO events (block_number)
		VALUES ($1)
		ON CONFLICT (block_number) DO NOTHING;
	`, event.BlockNumber)
	if err != nil {
		return fmt.Errorf("failed to insert Initialized event: %w", err)
	}

	// Update last processed block
	if err := ep.updateLastProcessedBlock(event.BlockNumber); err != nil {
		return fmt.Errorf("failed to update last processed block: %w", err)
	}

	// Log the event
	eventJSON, _ := json.Marshal(event)
	log.Printf("Processed Initialized event: %s", string(eventJSON))

	return nil
}

// processUpgraded processes Upgraded events
func (ep *EventProcessor) processUpgraded(vLog types.Log, contractABI *abi.ABI) error {
	var event struct {
		BlockNumber uint64 `json:"blockNumber"`
	}

	// Unpack the event data
	err := contractABI.UnpackIntoInterface(&event, "Upgraded", vLog.Data)
	if err != nil {
		return fmt.Errorf("failed to unpack Upgraded event: %w", err)
	}

	event.BlockNumber = vLog.BlockNumber

	// Insert into database
	_, err = ep.pool.Exec(context.Background(), `
		INSERT INTO events (block_number)
		VALUES ($1)
		ON CONFLICT (block_number) DO NOTHING;
	`, event.BlockNumber)
	if err != nil {
		return fmt.Errorf("failed to insert Upgraded event: %w", err)
	}

	// Update last processed block
	if err := ep.updateLastProcessedBlock(event.BlockNumber); err != nil {
		return fmt.Errorf("failed to update last processed block: %w", err)
	}

	// Log the event
	eventJSON, _ := json.Marshal(event)
	log.Printf("Processed Upgraded event: %s", string(eventJSON))

	return nil
}

// processGenericEvent processes any event type generically
func (ep *EventProcessor) processGenericEvent(vLog types.Log, contractABI *abi.ABI, eventName string) error {
	// Create a generic map to hold the event data
	eventData := make(map[string]interface{})

	// Try to unpack the event data
	err := contractABI.UnpackIntoInterface(&eventData, eventName, vLog.Data)
	if err != nil {
		return fmt.Errorf("failed to unpack generic event %s: %w", eventName, err)
	}

	// Add metadata
	eventData["blockNumber"] = vLog.BlockNumber
	eventData["eventName"] = eventName

	// Store in a generic events table
	eventJSON, _ := json.Marshal(eventData)
	_, err = ep.pool.Exec(context.Background(), `
		INSERT INTO generic_events (event_name, event_data, block_number)
		VALUES ($1, $2, $3)
	`, eventName, string(eventJSON), vLog.BlockNumber)
	if err != nil {
		return fmt.Errorf("failed to insert generic event: %w", err)
	}

	// Update last processed block
	if err := ep.updateLastProcessedBlock(vLog.BlockNumber); err != nil {
		return fmt.Errorf("failed to update last processed block: %w", err)
	}

	log.Printf("Processed generic event %s: %s", eventName, string(eventJSON))
	return nil
}

// GetABIManager returns the ABI manager
func (ep *EventProcessor) GetABIManager() *ABIManager {
	return ep.abiManager
}

// updateLastProcessedBlock updates the last processed block number
func (ep *EventProcessor) updateLastProcessedBlock(blockNumber uint64) error {
	tx, err := ep.pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), `
		UPDATE metadata SET value = GREATEST(value, $1), updated_at = NOW() WHERE key = 'last_processed_block';
	`, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to update last block: %w", err)
	}

	if err = tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
