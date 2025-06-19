package prompt

import (
	"testing"
)

func TestUser(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		defaultValue string
		validOptions []string
		expected     string
	}{
		{"use default when empty input", "", "0.5", []string{"0.5", "1.5", "7", "32", "72"}, "0.5"},
		{"use valid input", "7", "0.5", []string{"0.5", "1.5", "7", "32", "72"}, "7"},
		{"no validation when no options", "test", "default", []string{}, "test"},
		{"whitespace trimmed", "  7  ", "0.5", []string{"0.5", "1.5", "7", "32", "72"}, "7"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			SetTestInput([]string{c.input})
			result := User("Test prompt", c.defaultValue, c.validOptions)
			if result != c.expected {
				t.Errorf("User() = %v, want %v", result, c.expected)
			}
		})
	}
}

func TestUser_InvalidInputRetry(t *testing.T) {
	// Test that invalid input triggers retry
	SetTestInput([]string{"invalid", "7"})
	expected := "7"

	result := User("Test prompt", "0.5", []string{"0.5", "1.5", "7", "32", "72"})
	if result != expected {
		t.Errorf("User() = %v, want %v", result, expected)
	}
}

func TestYesNo(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		defaultValue string
		expected     bool
	}{
		{"yes input", "y", "N", true},
		{"yes input uppercase", "Y", "N", true},
		{"yes input full", "yes", "N", true},
		{"no input", "n", "Y", false},
		{"no input uppercase", "N", "Y", false},
		{"no input full", "no", "Y", false},
		{"empty input uses default yes", "", "Y", true},
		{"empty input uses default no", "", "N", false},
		{"whitespace trimmed", "  y  ", "N", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			SetTestInput([]string{c.input})
			result := YesNo("Test prompt", c.defaultValue)
			if result != c.expected {
				t.Errorf("YesNo() = %v, want %v", result, c.expected)
			}
		})
	}
}

func TestChoice(t *testing.T) {
	options := map[string]string{
		"A": "Math (small swarm)",
		"B": "Math Hard (big swarm)",
	}

	cases := []struct {
		name         string
		input        string
		defaultValue string
		expected     string
	}{
		{"valid choice A", "A", "A", "Math (small swarm)"},
		{"valid choice B", "B", "A", "Math Hard (big swarm)"},
		{"lowercase choice", "a", "A", "Math (small swarm)"},
		{"empty input uses default", "", "A", "Math (small swarm)"},
		{"whitespace trimmed", "  A  ", "B", "Math (small swarm)"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			SetTestInput([]string{c.input})
			result := Choice("Test prompt", options, c.defaultValue)
			if result != c.expected {
				t.Errorf("Choice() = %v, want %v", result, c.expected)
			}
		})
	}
}

func TestChoice_InvalidInputRetry(t *testing.T) {
	options := map[string]string{
		"A": "Math (small swarm)",
		"B": "Math Hard (big swarm)",
	}

	// Test that invalid input triggers retry
	SetTestInput([]string{"C", "A"})
	expected := "Math (small swarm)"

	result := Choice("Test prompt", options, "B")
	if result != expected {
		t.Errorf("Choice() = %v, want %v", result, expected)
	}
}

func TestHFToken(t *testing.T) {
	cases := []struct {
		name     string
		input    []string
		expected string
	}{
		{"no token", []string{"n"}, "None"},
		{"no token uppercase", []string{"N"}, "None"},
		{"no token full", []string{"no"}, "None"},
		{"empty input defaults to no", []string{""}, "None"},
		{"yes with token", []string{"y", "test-token"}, "test-token"},
		{"yes uppercase with token", []string{"Y", "test-token"}, "test-token"},
		{"yes full with token", []string{"yes", "test-token"}, "test-token"},
		{"token with whitespace", []string{"y", "  test-token  "}, "test-token"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			SetTestInput(c.input)
			result := HFToken()
			if result != c.expected {
				t.Errorf("HFToken() = %v, want %v", result, c.expected)
			}
		})
	}
}

func TestGetKeys(t *testing.T) {
	m := map[string]string{
		"A": "Math (small swarm)",
		"B": "Math Hard (big swarm)",
		"C": "Another option",
	}

	keys := GetKeys(m)

	// Check that all keys are present
	expectedKeys := map[string]bool{"A": true, "B": true, "C": true}
	if len(keys) != len(expectedKeys) {
		t.Errorf("GetKeys() returned %d keys, want %d", len(keys), len(expectedKeys))
	}

	for _, key := range keys {
		if !expectedKeys[key] {
			t.Errorf("GetKeys() returned unexpected key: %s", key)
		}
	}
}

func TestGetKeys_EmptyMap(t *testing.T) {
	m := map[string]string{}
	keys := GetKeys(m)

	if len(keys) != 0 {
		t.Errorf("GetKeys() returned %d keys for empty map, want 0", len(keys))
	}
}
