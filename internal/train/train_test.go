package train

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Deep-Commit/gswarm/internal/config"
)

// Mock command runner for testing
type mockTrainCommandRunner struct {
	commands     []string
	shouldFail   bool
	stdoutOutput string
}

func (m *mockTrainCommandRunner) Command(name string, args ...string) *exec.Cmd {
	cmd := strings.Join(append([]string{name}, args...), " ")
	m.commands = append(m.commands, cmd)

	if m.shouldFail {
		return exec.Command("false")
	}

	// Create a command that will produce the desired output
	return exec.Command("echo", m.stdoutOutput)
}

func TestIsCPUOnly(t *testing.T) {
	cases := []struct {
		name        string
		nvidiaSmi   bool
		expectedCPU bool
	}{
		{"CUDA available", true, false},
		{"CUDA not available", false, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			origCommandRunner := CommandRunner
			defer func() { CommandRunner = origCommandRunner }()

			CommandRunner = func(name string, _ ...string) *exec.Cmd {
				if name == "nvidia-smi" {
					if c.nvidiaSmi {
						return exec.Command("echo", "NVIDIA-SMI")
					}
					return exec.Command("false")
				}
				return exec.Command("false")
			}

			result := IsCPUOnly()
			if result != c.expectedCPU {
				t.Errorf("IsCPUOnly() = %v, want %v", result, c.expectedCPU)
			}
		})
	}
}

func TestInstallRequirements(t *testing.T) {
	tmp := t.TempDir()
	venvPath := filepath.Join(tmp, "venv")

	// Create mock venv structure
	os.MkdirAll(filepath.Join(venvPath, "bin"), 0o755)
	pythonPath := filepath.Join(venvPath, "bin", "python")
	os.WriteFile(pythonPath, []byte("#!/bin/bash\necho 'Python'"), 0o755)

	// Create test requirements file
	requirementsFile := filepath.Join(tmp, "requirements.txt")
	os.WriteFile(requirementsFile, []byte("torch\nnumpy"), 0o644)

	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()

	mock := &mockTrainCommandRunner{shouldFail: false}
	CommandRunner = mock.Command

	logger := log.New(os.Stdout, "", log.LstdFlags)
	err := InstallRequirements(venvPath, requirementsFile, logger)
	if err != nil {
		t.Errorf("InstallRequirements() error = %v", err)
	}

	// Should have called pip install
	expectedCmd := pythonPath + " -m pip install -r " + requirementsFile
	if len(mock.commands) != 1 || mock.commands[0] != expectedCmd {
		t.Errorf("Expected command %s, got %v", expectedCmd, mock.commands)
	}
}

func TestInstallRequirements_DefaultFile(t *testing.T) {
	tmp := t.TempDir()
	venvPath := filepath.Join(tmp, "venv")

	// Create mock venv structure
	os.MkdirAll(filepath.Join(venvPath, "bin"), 0o755)
	pythonPath := filepath.Join(venvPath, "bin", "python")
	os.WriteFile(pythonPath, []byte("#!/bin/bash\necho 'Python'"), 0o755)

	// Create default requirements.txt
	requirementsFile := filepath.Join(tmp, "requirements.txt")
	os.WriteFile(requirementsFile, []byte("torch\nnumpy"), 0o644)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)

	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()

	mock := &mockTrainCommandRunner{shouldFail: false}
	CommandRunner = mock.Command

	logger := log.New(os.Stdout, "", log.LstdFlags)
	err := InstallRequirements(venvPath, "", logger) // Empty requirements file
	if err != nil {
		t.Errorf("InstallRequirements() error = %v", err)
	}

	// Should have used default requirements.txt
	expectedCmd := pythonPath + " -m pip install -r requirements.txt"
	if len(mock.commands) != 1 || mock.commands[0] != expectedCmd {
		t.Errorf("Expected command %s, got %v", expectedCmd, mock.commands)
	}
}

func TestRunPythonTraining_IdentityConflictDetection(t *testing.T) {
	// Note: We no longer detect identity conflicts since switching to direct passthrough
	// to preserve TTY detection and progress bars. This test is kept for future reference.
	cases := []struct {
		name           string
		stdoutOutput   string
		stderrOutput   string
		expectConflict bool
	}{
		{
			name:           "identity conflict in stdout",
			stdoutOutput:   ">> An error was detected while running rl-swarm.",
			stderrOutput:   "",
			expectConflict: false, // No longer detected
		},
		{
			name:           "identity conflict in stderr",
			stdoutOutput:   "",
			stderrOutput:   "Error: is already taken by another user",
			expectConflict: false, // No longer detected
		},
		{
			name:           "no conflict",
			stdoutOutput:   "Training started successfully",
			stderrOutput:   "",
			expectConflict: false,
		},
		{
			name:           "other error not conflict",
			stdoutOutput:   "Some other error occurred",
			stderrOutput:   "",
			expectConflict: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			config := config.Configuration{
				HFToken:      "test-token",
				IdentityPath: "test.pem",
				ConfigPath:   "test.yaml",
				Game:         "gsm8k",
			}

			venvPath := t.TempDir()
			logger := log.New(os.Stdout, "", log.LstdFlags)

			origCommandRunner := CommandRunner
			defer func() { CommandRunner = origCommandRunner }()

			// Create a mock command that produces the test output
			CommandRunner = func(_ string, _ ...string) *exec.Cmd {
				// Create a command that will produce the desired output
				script := fmt.Sprintf(`
					echo "%s" >&1
					echo "%s" >&2
					exit 0
				`, c.stdoutOutput, c.stderrOutput)

				return exec.Command("bash", "-c", script)
			}

			err := RunPythonTraining(config, venvPath, logger)

			// Since we switched to direct passthrough, we no longer detect identity conflicts
			// All errors would be passed through to the user directly
			if err != nil {
				t.Errorf("Expected no error detection (direct passthrough), got %v", err)
			}
		})
	}
}

func TestCleanupStaleProcesses(t *testing.T) {
	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()

	mock := &mockTrainCommandRunner{shouldFail: false}
	CommandRunner = mock.Command

	logger := log.New(os.Stdout, "", log.LstdFlags)
	err := CleanupStaleProcesses(logger)
	if err != nil {
		t.Errorf("CleanupStaleProcesses() error = %v", err)
	}

	// Should have called pkill commands
	expectedCommands := []string{
		"pkill -f gensyn",
		"pkill -f hivemind",
		"pgrep -f gensyn",
	}

	if len(mock.commands) < len(expectedCommands) {
		t.Errorf("Expected at least %d commands, got %d", len(expectedCommands), len(mock.commands))
	}
}

func TestCleanupStaleProcesses_ForceKill(t *testing.T) {
	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()

	callCount := 0
	CommandRunner = func(name string, args ...string) *exec.Cmd {
		callCount++
		cmd := strings.Join(append([]string{name}, args...), " ")

		// pgrep succeeds (processes still exist), triggering force kill
		if strings.Contains(cmd, "pgrep") {
			return exec.Command("echo", "12345") // Simulate process found
		}
		return exec.Command("echo", "success")
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	err := CleanupStaleProcesses(logger)
	if err != nil {
		t.Errorf("CleanupStaleProcesses() error = %v", err)
	}

	// Should have called force kill commands
	if callCount < 5 { // pkill gensyn, pkill hivemind, pgrep, pkill -9 gensyn, pkill -9 hivemind
		t.Errorf("Expected at least 5 command calls for force kill, got %d", callCount)
	}
}
