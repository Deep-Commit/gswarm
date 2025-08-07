package blockchain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// ABIEvent represents a generic event structure
type ABIEvent struct {
	Type      string     `json:"type"`
	Name      string     `json:"name"`
	Inputs    []ABIInput `json:"inputs"`
	Anonymous bool       `json:"anonymous,omitempty"`
}

// ABIInput represents an event input parameter
type ABIInput struct {
	Indexed      bool   `json:"indexed"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	InternalType string `json:"internalType,omitempty"`
}

// ABIManager handles loading and managing multiple contract ABIs
type ABIManager struct {
	abis map[string]*abi.ABI
}

// NewABIManager creates a new ABI manager
func NewABIManager() *ABIManager {
	return &ABIManager{
		abis: make(map[string]*abi.ABI),
	}
}

// LoadABIFromFile loads an ABI from a JSON file
func (am *ABIManager) LoadABIFromFile(contractName, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read ABI file %s: %w", filePath, err)
	}

	// Parse the full ABI JSON directly
	var fullABI []map[string]interface{}
	if err := json.Unmarshal(data, &fullABI); err != nil {
		return fmt.Errorf("failed to parse ABI JSON %s: %w", filePath, err)
	}

	// Filter only events
	var events []map[string]interface{}
	for _, item := range fullABI {
		if itemType, ok := item["type"].(string); ok && itemType == "event" {
			events = append(events, item)
		}
	}

	// Convert to go-ethereum ABI format
	abiData, err := am.convertToEthereumABI(events)
	if err != nil {
		return fmt.Errorf("failed to convert ABI for %s: %w", contractName, err)
	}

	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(abiData))
	if err != nil {
		return fmt.Errorf("failed to parse ABI for %s: %w", contractName, err)
	}

	am.abis[contractName] = &parsedABI
	return nil
}

// LoadABIsFromDirectory loads all ABI files from a directory
func (am *ABIManager) LoadABIsFromDirectory(dirPath string) error {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		contractName := strings.TrimSuffix(file.Name(), ".json")
		filePath := filepath.Join(dirPath, file.Name())

		if err := am.LoadABIFromFile(contractName, filePath); err != nil {
			return fmt.Errorf("failed to load ABI for %s: %w", contractName, err)
		}
	}

	return nil
}

// GetABI returns the ABI for a specific contract
func (am *ABIManager) GetABI(contractName string) (*abi.ABI, error) {
	abi, exists := am.abis[contractName]
	if !exists {
		return nil, fmt.Errorf("ABI not found for contract: %s", contractName)
	}
	return abi, nil
}

// GetEventSignature returns the event signature for a specific contract and event
func (am *ABIManager) GetEventSignature(contractName, eventName string) (string, error) {
	abi, err := am.GetABI(contractName)
	if err != nil {
		return "", err
	}

	event, exists := abi.Events[eventName]
	if !exists {
		return "", fmt.Errorf("event %s not found in contract %s", eventName, contractName)
	}

	return event.Sig, nil
}

// ListContracts returns all loaded contract names
func (am *ABIManager) ListContracts() []string {
	contracts := make([]string, 0, len(am.abis))
	for contract := range am.abis {
		contracts = append(contracts, contract)
	}
	return contracts
}

// ListEvents returns all events for a specific contract
func (am *ABIManager) ListEvents(contractName string) ([]string, error) {
	abi, err := am.GetABI(contractName)
	if err != nil {
		return nil, err
	}

	events := make([]string, 0, len(abi.Events))
	for eventName := range abi.Events {
		events = append(events, eventName)
	}
	return events, nil
}

// convertToEthereumABI converts our JSON format to go-ethereum ABI format
func (am *ABIManager) convertToEthereumABI(events []map[string]interface{}) (string, error) {
	// Convert to go-ethereum ABI format
	ethereumABI := make([]map[string]interface{}, 0, len(events))

	for _, event := range events {
		ethereumEvent := map[string]interface{}{
			"type":   event["type"],
			"name":   event["name"],
			"inputs": make([]map[string]interface{}, 0),
		}

		// Add anonymous field if present
		if anonymous, ok := event["anonymous"].(bool); ok {
			ethereumEvent["anonymous"] = anonymous
		}

		// Convert inputs
		if inputs, ok := event["inputs"].([]interface{}); ok {
			for _, input := range inputs {
				if inputMap, ok := input.(map[string]interface{}); ok {
					ethereumInput := map[string]interface{}{
						"indexed": inputMap["indexed"],
						"name":    inputMap["name"],
						"type":    inputMap["type"],
					}
					ethereumEvent["inputs"] = append(ethereumEvent["inputs"].([]map[string]interface{}), ethereumInput)
				}
			}
		}

		ethereumABI = append(ethereumABI, ethereumEvent)
	}

	// Convert to JSON string
	data, err := json.Marshal(ethereumABI)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ABI: %w", err)
	}

	return string(data), nil
}
