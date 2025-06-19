// Package train provides training utilities for GSwarm,
// including model training orchestration and process management.
package train

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

	// Set environment variables like the bash script does
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PUB_MULTI_ADDRS=%s", config.PublicMaddr),
		fmt.Sprintf("PEER_MULTI_ADDRS=%s", config.PeerMaddr),
		fmt.Sprintf("HOST_MULTI_ADDRS=%s", config.HostMaddr),
		fmt.Sprintf("IDENTITY_PATH=%s", config.IdentityPath),
		fmt.Sprintf("CONNECT_TO_TESTNET=%t", config.ConnectToTestnet),
		fmt.Sprintf("ORG_ID=%s", config.OrgID),
		"HF_HUB_DOWNLOAD_TIMEOUT=120",
	)

	// Use direct passthrough to preserve TTY detection and progress bars
	// Note: We lose identity conflict detection with this approach, but gain proper progress bars
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start training process: %w", err)
	}

	err := cmd.Wait()
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
