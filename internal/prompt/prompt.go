// Package prompt provides user interaction utilities for GSwarm,
// including input validation and user prompting functionality.
package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// For testing purposes
var (
	testInput      []string
	testInputIndex int
)

// SetTestInput sets input for testing
func SetTestInput(input []string) {
	testInput = input
	testInputIndex = 0
}

// getTestInput gets the next test input or reads from stdin
func getTestInput() string {
	if testInput != nil && testInputIndex < len(testInput) {
		input := testInput[testInputIndex]
		testInputIndex++
		return input
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		// If we can't read from stdin, return empty string
		return ""
	}
	return input
}

// User prompts the user for input with validation
func User(prompt string, defaultValue string, validOptions []string) string {
	fmt.Printf("\033[32m%s [%s]: \033[0m", prompt, defaultValue)
	input := getTestInput()
	input = strings.TrimSpace(input)

	if input == "" {
		input = defaultValue
	}

	// Validate against valid options if provided
	if len(validOptions) > 0 {
		valid := false
		for _, option := range validOptions {
			if input == option {
				valid = true
				break
			}
		}
		if !valid {
			fmt.Printf("Invalid option. Please choose from: %v\n", validOptions)
			return User(prompt, defaultValue, validOptions)
		}
	}

	return input
}

// YesNo prompts the user for a yes/no response
func YesNo(prompt string, defaultValue string) bool {
	fmt.Printf("\033[32m%s [%s]: \033[0m", prompt, defaultValue)
	input := getTestInput()
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		input = strings.ToLower(defaultValue)
	}

	return input == "y" || input == "yes"
}

// Choice prompts the user to choose from a set of options
func Choice(prompt string, options map[string]string, defaultValue string) string {
	fmt.Printf("\033[32m%s\n", prompt)
	for key, value := range options {
		fmt.Printf("  %s: %s\n", key, value)
	}
	fmt.Printf("Choice [%s]: \033[0m", defaultValue)

	input := getTestInput()
	input = strings.TrimSpace(strings.ToUpper(input))

	if input == "" {
		input = strings.ToUpper(defaultValue)
	}

	if value, exists := options[input]; exists {
		return value
	}

	fmt.Println("Invalid choice. Please try again.")
	return Choice(prompt, options, defaultValue)
}

// HFToken prompts the user for a HuggingFace token
func HFToken() string {
	fmt.Printf("\033[32mWould you like to push models you train in the RL swarm to the Hugging Face Hub? [y/N]: \033[0m")
	input := getTestInput()
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		input = "n"
	}

	if input == "y" || input == "yes" {
		fmt.Print("Enter your HuggingFace access token: ")
		token := getTestInput()
		return strings.TrimSpace(token)
	}

	return "None"
}

// GetKeys returns the keys from a map as a slice
func GetKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
