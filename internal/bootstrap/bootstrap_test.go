package bootstrap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Mock command runner for testing
type mockCommandRunner struct {
	commands []string
	success  bool
}

func (m *mockCommandRunner) Command(name string, args ...string) *exec.Cmd {
	m.commands = append(m.commands, strings.Join(append([]string{name}, args...), " "))

	if m.success {
		// Special handling for venv creation
		if name == "python3" && len(args) >= 3 && args[0] == "-m" && args[1] == "venv" {
			venvPath := args[2]
			// Create the venv directory structure
			os.MkdirAll(filepath.Join(venvPath, "bin"), 0o755)
			pythonPath := filepath.Join(venvPath, "bin", "python")
			os.WriteFile(pythonPath, []byte("#!/bin/bash\necho 'Python 3.10.0'"), 0o755)
		}
		return exec.Command("echo", "success")
	}
	return exec.Command("false")
}

func TestCheckPythonVersion(t *testing.T) {
	cases := []struct {
		name    string
		output  string
		wantErr bool
	}{
		{"valid python 3.10", "Python 3.10.0", false},
		{"valid python 3.11", "Python 3.11.5", false},
		{"valid python 3.12", "Python 3.12.1", false},
		{"invalid python 2.7", "Python 2.7.18", true},
		{"invalid python 3.9", "Python 3.9.9", true},
		{"malformed version", "Python abc", true},
		{"empty version", "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Create a mock command that returns the test output
			origCommandRunner := CommandRunner
			defer func() { CommandRunner = origCommandRunner }()

			CommandRunner = func(name string, args ...string) *exec.Cmd {
				if name == "python3" && len(args) > 0 && args[0] == "--version" {
					return exec.Command("echo", c.output)
				}
				return exec.Command("false")
			}

			err := CheckPythonVersion()
			if (err != nil) != c.wantErr {
				t.Errorf("CheckPythonVersion() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

func TestEnsureNodeAndNpm_AlreadyInstalled(t *testing.T) {
	// Mock that both node and npm are already available
	mock := &mockCommandRunner{success: true}
	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()
	CommandRunner = mock.Command

	err := EnsureNodeAndNpm()
	if err != nil {
		t.Errorf("EnsureNodeAndNpm() error = %v, expected no error when already installed", err)
	}

	// Should only check for node and npm, not install
	expectedCommands := []string{"node --version", "npm --version"}
	if len(mock.commands) != len(expectedCommands) {
		t.Errorf("Expected %d commands, got %d", len(expectedCommands), len(mock.commands))
	}
}

func TestEnsureNodeAndNpm_InstallsWhenMissing(t *testing.T) {
	// Mock that node/npm are missing initially, then available after install
	callCount := 0
	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()

	CommandRunner = func(_ string, _ ...string) *exec.Cmd {
		callCount++

		// First calls to check node/npm fail, then succeed after "installation"
		if callCount <= 2 {
			return exec.Command("false")
		}
		return exec.Command("echo", "success")
	}

	err := EnsureNodeAndNpm()
	if err != nil {
		t.Errorf("EnsureNodeAndNpm() error = %v, expected no error after installation", err)
	}

	// Should have attempted installation commands
	if callCount < 3 {
		t.Errorf("Expected at least 3 command calls, got %d", callCount)
	}
}

func TestEnsureVenv_CreatesNewVenv(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)

	// Mock command runner
	mock := &mockCommandRunner{success: true}
	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()
	CommandRunner = mock.Command

	venvPath, err := EnsureVenv()
	if err != nil {
		t.Fatalf("EnsureVenv() error = %v", err)
	}

	// Check that venv was created
	expectedVenvPath := filepath.Join(tmp, venvName)
	absExpectedVenvPath, _ := filepath.Abs(expectedVenvPath)
	resolvedVenvPath, _ := filepath.EvalSymlinks(venvPath)
	resolvedExpectedVenvPath, _ := filepath.EvalSymlinks(absExpectedVenvPath)
	if resolvedVenvPath != resolvedExpectedVenvPath {
		t.Errorf("Expected venv path %s, got %s", resolvedExpectedVenvPath, resolvedVenvPath)
	}

	// Check that the venv directory exists
	if _, err := os.Stat(resolvedExpectedVenvPath); os.IsNotExist(err) {
		t.Errorf("Virtual environment directory was not created")
	}

	// Should have called python3 -m venv and pip install --upgrade pip
	expectedCommands := []string{
		"python3 -m venv " + venvName,
		filepath.Join(venvName, "bin", "python") + " -m pip install --upgrade pip",
	}

	if len(mock.commands) != len(expectedCommands) {
		t.Errorf("Expected %d commands, got %d", len(expectedCommands), len(mock.commands))
	}
}

func TestEnsureVenv_UsesExistingVenv(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)

	// Create existing venv structure
	venvDir := filepath.Join(tmp, venvName)
	os.MkdirAll(venvDir, 0o755)

	// Create python executable
	pythonPath := filepath.Join(venvDir, "bin", "python")
	os.MkdirAll(filepath.Dir(pythonPath), 0o755)
	os.WriteFile(pythonPath, []byte("#!/bin/bash\necho 'Python 3.10.0'"), 0o755)

	// Mock command runner
	mock := &mockCommandRunner{success: true}
	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()
	CommandRunner = mock.Command

	venvPath, err := EnsureVenv()
	if err != nil {
		t.Fatalf("EnsureVenv() error = %v", err)
	}

	// Should not have created new venv, only upgraded pip
	if len(mock.commands) != 1 {
		t.Errorf("Expected 1 command (pip upgrade), got %d", len(mock.commands))
	}

	expectedVenvPath := filepath.Join(tmp, venvName)
	absExpectedVenvPath, _ := filepath.Abs(expectedVenvPath)
	resolvedVenvPath, _ := filepath.EvalSymlinks(venvPath)
	resolvedExpectedVenvPath, _ := filepath.EvalSymlinks(absExpectedVenvPath)
	if resolvedVenvPath != resolvedExpectedVenvPath {
		t.Errorf("Expected venv path %s, got %s", resolvedExpectedVenvPath, resolvedVenvPath)
	}
}

func TestCheckYarn_Available(t *testing.T) {
	mock := &mockCommandRunner{success: true}
	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()
	CommandRunner = mock.Command

	err := CheckYarn()
	if err != nil {
		t.Errorf("CheckYarn() error = %v, expected no error when yarn is available", err)
	}
}

func TestCheckYarn_NotAvailable(t *testing.T) {
	mock := &mockCommandRunner{success: false}
	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()
	CommandRunner = mock.Command

	err := CheckYarn()
	if err == nil {
		t.Error("CheckYarn() expected error when yarn is not available")
	}
}

func TestInstallYarn_Success(t *testing.T) {
	// Mock successful yarn installation
	callCount := 0
	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()

	CommandRunner = func(_ string, _ ...string) *exec.Cmd {
		callCount++

		// First call fails (yarn not found), subsequent calls succeed
		if callCount == 1 {
			return exec.Command("false")
		}
		return exec.Command("echo", "success")
	}

	err := InstallYarn()
	if err != nil {
		t.Errorf("InstallYarn() error = %v, expected no error on successful installation", err)
	}

	// Should have attempted installation
	if callCount < 2 {
		t.Errorf("Expected at least 2 command calls, got %d", callCount)
	}
}

func TestInstallYarn_Failure(t *testing.T) {
	// Mock failed yarn installation
	mock := &mockCommandRunner{success: false}
	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()
	CommandRunner = mock.Command

	err := InstallYarn()
	if err == nil {
		t.Error("InstallYarn() expected error on failed installation")
	}
}

func TestEnsureRepo_AlreadyInRepo(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)

	// Create go.mod to simulate being in a repo
	os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test"), 0o644)

	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()
	CommandRunner = func(_ string, _ ...string) *exec.Cmd {
		t.Errorf("Command should not be called when already in repo")
		return exec.Command("false")
	}

	err := EnsureRepo()
	if err != nil {
		t.Errorf("EnsureRepo() error = %v, expected no error when already in repo", err)
	}
}

func TestEnsureRepo_ClonesRepo(t *testing.T) {
	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)

	// Mock successful git clone that creates the directory
	callCount := 0
	origCommandRunner := CommandRunner
	defer func() { CommandRunner = origCommandRunner }()

	CommandRunner = func(name string, args ...string) *exec.Cmd {
		callCount++

		// For git clone command, create the directory structure
		if name == "git" && len(args) >= 2 && args[0] == "clone" {
			// Create the rl-swarm directory and go.mod file
			os.MkdirAll("rl-swarm", 0o755)
			os.WriteFile(filepath.Join("rl-swarm", "go.mod"), []byte("module rl-swarm"), 0o644)
		}

		return exec.Command("echo", "success")
	}

	err := EnsureRepo()
	if err != nil {
		t.Errorf("EnsureRepo() error = %v, expected no error on successful clone", err)
	}

	// Should have called git clone
	expectedCommands := []string{"git --version", "git clone https://github.com/gensyn-ai/rl-swarm.git"}
	if callCount != len(expectedCommands) {
		t.Errorf("Expected %d commands, got %d", len(expectedCommands), callCount)
	}
}
