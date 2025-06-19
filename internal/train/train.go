// Package train provides training utilities for GSwarm,
// including model training orchestration and process management.
package train

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Deep-Commit/gswarm/internal/config"
)

// CommandRunner is a package-level variable that can be replaced in tests
var CommandRunner = exec.Command

const (
	// OS constants
	OSWindows = "windows"
)

var errorMarkers = []string{
	">> An error was detected while running rl-swarm.",
	">> Shutting down trainer...",
	"Error:",
	"Exception:",
	"Traceback:",
	"is already taken by another user",
}

// InstallRequirements installs Python requirements in the virtual environment
func InstallRequirements(venvPath string, requirementsFile string, _ *log.Logger) error {
	venvPython := filepath.Join(venvPath, "bin", "python")
	if runtime.GOOS == OSWindows {
		venvPython = filepath.Join(venvPath, "Scripts", "python.exe")
	}

	// Determine which requirements file to use
	if requirementsFile == "" {
		// Check for requirements.txt in current directory
		if _, err := os.Stat("requirements.txt"); err == nil {
			requirementsFile = "requirements.txt"
		} else {
			// Use default requirements
			requirementsFile = "requirements.txt"
		}
	}

	fmt.Printf("Installing requirements from %s...\n", requirementsFile)

	// Install requirements
	cmd := CommandRunner(venvPython, "-m", "pip", "install", "-r", requirementsFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install requirements: %w", err)
	}

	return nil
}

// IsCPUOnly checks if CUDA is available
func IsCPUOnly() bool {
	// Check if CUDA is available
	cmd := CommandRunner("nvidia-smi")
	return cmd.Run() != nil
}

// RunPythonTraining runs the Python training process
func RunPythonTraining(config config.Configuration, venvPath string, logger *log.Logger) error {
	venvPython := filepath.Join(venvPath, "bin", "python")
	if runtime.GOOS == OSWindows {
		venvPython = filepath.Join(venvPath, "Scripts", "python.exe")
	}

	args := []string{
		"-m", "hivemind_exp.gsm8k.train_single_gpu",
		"--hf_token", config.HFToken,
		"--identity_path", config.IdentityPath,
		"--config", config.ConfigPath,
		"--game", config.Game,
		"--param_b", config.ParamB,
	}

	if config.ConnectToTestnet && config.OrgID != "" {
		args = append(args, "--modal_org_id", config.OrgID)
		args = append(args, "--contract_address", config.ContractAddress)
	} else {
		args = append(args, "--public_maddr", config.PublicMaddr)
		args = append(args, "--initial_peers", config.PeerMaddr)
		args = append(args, "--host_maddr", config.HostMaddr)
	}

	cmd := CommandRunner(venvPython, args...)

	// Capture stdout and stderr to detect identity conflicts using atomic operations
	var identityConflictDetected atomic.Bool

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	cmd.Stdin = os.Stdin

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start training process: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line) // Still print to console

			// Check for identity conflict patterns
			for _, marker := range errorMarkers {
				if strings.Contains(strings.ToLower(line), strings.ToLower(marker)) {
					identityConflictDetected.Store(true)
					logger.Printf("Identity conflict detected in stdout: %s", line)
					break
				}
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(os.Stderr, "%s\n", line) // Still print to stderr

			// Check for identity conflict patterns in stderr too
			for _, marker := range errorMarkers {
				if strings.Contains(strings.ToLower(line), strings.ToLower(marker)) {
					identityConflictDetected.Store(true)
					logger.Printf("Identity conflict detected in stderr: %s", line)
					break
				}
			}
		}
	}()

	err = cmd.Wait()

	if identityConflictDetected.Load() {
		return fmt.Errorf("identity conflict detected - need cleanup and retry")
	}

	return err
}

// CleanupStaleProcesses kills any existing gensyn and hivemind processes
func CleanupStaleProcesses(logger *log.Logger) error {
	fmt.Println("Cleaning up stale gensyn processes...")
	logger.Printf("Cleaning up stale gensyn processes")

	// Kill any existing gensyn processes
	cmd := CommandRunner("pkill", "-f", "gensyn")
	if err := cmd.Run(); err != nil {
		// pkill returns error if no processes found, which is fine
		fmt.Println("No existing gensyn processes found")
		logger.Printf("No existing gensyn processes found")
	} else {
		fmt.Println("Killed existing gensyn processes")
		logger.Printf("Killed existing gensyn processes")
	}

	// Also try to kill hivemind processes
	cmd = CommandRunner("pkill", "-f", "hivemind")
	if err := cmd.Run(); err != nil {
		fmt.Println("No existing hivemind processes found")
		logger.Printf("No existing hivemind processes found")
	} else {
		fmt.Println("Killed existing hivemind processes")
		logger.Printf("Killed existing hivemind processes")
	}

	// Wait a moment for processes to fully terminate
	time.Sleep(2 * time.Second)

	// Check for any remaining processes
	cmd = CommandRunner("pgrep", "-f", "gensyn")
	if err := cmd.Run(); err == nil {
		// Still have processes, try force kill
		fmt.Println("Force killing remaining gensyn processes...")
		logger.Printf("Force killing remaining gensyn processes")
		if err := CommandRunner("pkill", "-9", "-f", "gensyn").Run(); err != nil {
			logger.Printf("Failed to force kill gensyn processes: %v", err)
		}
		if err := CommandRunner("pkill", "-9", "-f", "hivemind").Run(); err != nil {
			logger.Printf("Failed to force kill hivemind processes: %v", err)
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}
