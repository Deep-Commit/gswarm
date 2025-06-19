package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Deep-Commit/gswarm/internal/bootstrap"
	"github.com/Deep-Commit/gswarm/internal/config"
	"github.com/Deep-Commit/gswarm/internal/train"
)

// TestMain_Integration tests the main application flow with mocked dependencies
func TestMain_Integration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	_, oldWd := setupTestEnvironment(t)
	defer cleanupTestEnvironment(t, oldWd)

	// Mock dependencies
	setupMockDependencies(t)

	// Test configuration
	cfg := getTestConfiguration()

	// Run integration tests
	testBootstrap(t)
	testRequirementsInstallation(t, cfg)
	testTrainingProcess(t, cfg)
}

func setupTestEnvironment(t *testing.T) (string, string) {
	// Create temporary directory for test
	tmp := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("os.Chdir(tmp) error = %v", err)
	}

	// Create mock go.mod to simulate being in a repo
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(go.mod) error = %v", err)
	}

	return tmp, oldWd
}

func cleanupTestEnvironment(t *testing.T, oldWd string) {
	if err := os.Chdir(oldWd); err != nil {
		t.Errorf("defer os.Chdir(oldWd) error = %v", err)
	}
}

func setupMockDependencies(t *testing.T) {
	// Mock all external dependencies
	origBootstrapCommandRunner := bootstrap.CommandRunner
	origTrainCommandRunner := train.CommandRunner
	t.Cleanup(func() {
		bootstrap.CommandRunner = origBootstrapCommandRunner
		train.CommandRunner = origTrainCommandRunner
	})

	// Mock bootstrap commands to succeed
	bootstrap.CommandRunner = func(name string, args ...string) *exec.Cmd {
		cmd := strings.Join(append([]string{name}, args...), " ")

		// Mock Python version check
		if strings.Contains(cmd, "python3 --version") {
			return exec.Command("echo", "Python 3.10.0")
		}

		// Mock Node.js and npm checks
		if strings.Contains(cmd, "node --version") || strings.Contains(cmd, "npm --version") {
			return exec.Command("echo", "v18.0.0")
		}

		// Mock Yarn check
		if strings.Contains(cmd, "yarn --version") {
			return exec.Command("echo", "1.22.0")
		}

		// Mock virtual environment creation
		if strings.Contains(cmd, "python3 -m venv") {
			venvPath := strings.Fields(cmd)[len(strings.Fields(cmd))-1]
			if err := os.MkdirAll(filepath.Join(venvPath, "bin"), 0o755); err != nil {
				t.Fatalf("os.MkdirAll(bin) error = %v", err)
			}
			pythonPath := filepath.Join(venvPath, "bin", "python")
			if err := os.WriteFile(pythonPath, []byte("#!/bin/bash\necho 'Python 3.10.0'"), 0o755); err != nil {
				t.Fatalf("os.WriteFile(pythonPath) error = %v", err)
			}
			return exec.Command("echo", "success")
		}

		// Mock pip upgrade
		if strings.Contains(cmd, "pip install --upgrade pip") {
			return exec.Command("echo", "success")
		}

		return exec.Command("echo", "success")
	}

	// Mock training commands
	train.CommandRunner = func(_ string, args ...string) *exec.Cmd {
		cmd := strings.Join(args, " ")

		// Mock requirements installation
		if strings.Contains(cmd, "pip install -r") {
			return exec.Command("echo", "success")
		}

		// Mock training process
		if strings.Contains(cmd, "hivemind_exp.gsm8k.train_single_gpu") {
			return exec.Command("echo", "Training completed successfully")
		}

		return exec.Command("echo", "success")
	}
}

func getTestConfiguration() config.Configuration {
	return config.Configuration{
		ConnectToTestnet: false,
		UseBigSwarm:      false,
		ParamB:           "0.5",
		CPUOnly:          true,
		HFToken:          "test-token",
		OrgID:            "",
		IdentityPath:     "test.pem",
		ContractAddress:  config.SmallSwarmContract,
		Game:             "gsm8k",
		ConfigPath:       "testdata/quick.yaml",
		PublicMaddr:      config.DefaultPublicMaddr,
		PeerMaddr:        config.DefaultPeerMaddr,
		HostMaddr:        config.DefaultHostMaddr,
		RequirementsFile: "",
	}
}

func testBootstrap(t *testing.T) {
	venvPath, err := bootstrap.Env()
	if err != nil {
		t.Fatalf("Env() error = %v", err)
	}

	if venvPath == "" {
		t.Error("Env() returned empty venv path")
	}
}

func testRequirementsInstallation(t *testing.T, cfg config.Configuration) {
	venvPath, err := bootstrap.Env()
	if err != nil {
		t.Fatalf("Env() error = %v", err)
	}

	logger := config.GetTestLogger()
	err = train.InstallRequirements(venvPath, cfg.RequirementsFile, logger)
	if err != nil {
		t.Fatalf("InstallRequirements() error = %v", err)
	}
}

func testTrainingProcess(t *testing.T, cfg config.Configuration) {
	venvPath, err := bootstrap.Env()
	if err != nil {
		t.Fatalf("Env() error = %v", err)
	}

	logger := config.GetTestLogger()
	err = train.RunPythonTraining(cfg, venvPath, logger)
	if err != nil {
		t.Fatalf("RunPythonTraining() error = %v", err)
	}
}

// TestMain_FlagParsing tests command line flag parsing
func TestMain_FlagParsing(_ *testing.T) {
	// Reset flags to avoid test interference
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	// Test version flag
	*config.ShowVersionFlag = true

	// This would normally call os.Exit(0), so we can't easily test it
	// In a real test, you'd need to capture stdout or use a different approach
}

// TestMain_ConfigurationValidation tests configuration validation
func TestMain_ConfigurationValidation(t *testing.T) {
	cases := []struct {
		name    string
		config  config.Configuration
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: config.Configuration{
				ParamB: "0.5",
				Game:   "gsm8k",
			},
			wantErr: false,
		},
		{
			name: "invalid paramB",
			config: config.Configuration{
				ParamB: "3",
				Game:   "gsm8k",
			},
			wantErr: true,
		},
		{
			name: "invalid game",
			config: config.Configuration{
				ParamB: "0.5",
				Game:   "invalid",
			},
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := config.ValidateConfiguration(c.config)
			if (err != nil) != c.wantErr {
				t.Errorf("ValidateConfiguration() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

// TestMain_ErrorHandling tests error handling scenarios
func TestMain_ErrorHandling(t *testing.T) {
	// Test identity conflict detection
	cfg := config.Configuration{
		HFToken:      "test-token",
		IdentityPath: "test.pem",
		ConfigPath:   "test.yaml",
		Game:         "gsm8k",
	}

	venvPath := t.TempDir()
	logger := config.GetTestLogger()

	origTrainCommandRunner := train.CommandRunner
	defer func() { train.CommandRunner = origTrainCommandRunner }()

	// Mock command that produces identity conflict error
	train.CommandRunner = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", ">> An error was detected while running rl-swarm.")
	}

	err := train.RunPythonTraining(cfg, venvPath, logger)
	if err == nil || !strings.Contains(err.Error(), "identity conflict detected") {
		t.Errorf("Expected identity conflict error, got %v", err)
	}
}

// TestMain_ProcessCleanup tests process cleanup functionality
func TestMain_ProcessCleanup(t *testing.T) {
	logger := config.GetTestLogger()

	origTrainCommandRunner := train.CommandRunner
	defer func() { train.CommandRunner = origTrainCommandRunner }()

	// Mock cleanup commands
	callCount := 0
	train.CommandRunner = func(_ string, _ ...string) *exec.Cmd {
		callCount++
		return exec.Command("echo", "success")
	}

	err := train.CleanupStaleProcesses(logger)
	if err != nil {
		t.Errorf("CleanupStaleProcesses() error = %v", err)
	}

	// Should have called cleanup commands
	if callCount < 3 {
		t.Errorf("Expected at least 3 cleanup commands, got %d", callCount)
	}
}
