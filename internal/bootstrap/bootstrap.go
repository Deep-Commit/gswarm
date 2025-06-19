// Package bootstrap provides utilities for setting up the GSwarm environment,
// including repository management, virtual environment creation, and dependency installation.
package bootstrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// CommandRunner is a package-level variable that can be replaced in tests
var CommandRunner = exec.Command

const (
	venvName = "gswarm-venv"

	// OS constants
	OSDarwin  = "darwin"
	OSLinux   = "linux"
	OSWindows = "windows"
)

// EnsureRepo ensures we're in the correct repository
func EnsureRepo() error {
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		fmt.Println("Not in RL-Swarm repository. Cloning...")

		// Check if git is available
		if err := checkGit(); err != nil {
			if err := installGit(); err != nil {
				return fmt.Errorf("failed to install git: %w", err)
			}
		}

		cmd := CommandRunner("git", "clone", "https://github.com/gensyn-ai/rl-swarm.git")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone rl-swarm: %w", err)
		}

		// Check if the directory was actually created
		if _, err := os.Stat("rl-swarm"); os.IsNotExist(err) {
			return fmt.Errorf("failed to clone rl-swarm: directory not created")
		}

		if err := os.Chdir("rl-swarm"); err != nil {
			return fmt.Errorf("failed to change to rl-swarm directory: %w", err)
		}
		fmt.Println("Successfully cloned and entered RL-Swarm repository")
	}
	return nil
}

func checkGit() error {
	cmd := CommandRunner("git", "--version")
	return cmd.Run()
}

func installGit() error {
	fmt.Println("Git not found. Installing...")

	switch runtime.GOOS {
	case OSDarwin:
		cmd := CommandRunner("brew", "install", "git")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case OSLinux:
		cmd := CommandRunner("sudo", "apt-get", "update")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to update package list: %w", err)
		}

		cmd = CommandRunner("sudo", "apt-get", "install", "-y", "git")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported OS for git installation: %s", runtime.GOOS)
	}
}

// EnsureVenv ensures the Python virtual environment exists and is properly set up
func EnsureVenv() (string, error) {
	venvPath := venvName

	// Check if venv already exists
	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		fmt.Printf("Creating virtual environment: %s\n", venvPath)

		cmd := CommandRunner("python3", "-m", "venv", venvPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to create virtual environment: %w", err)
		}
	}

	// Get absolute path for the venv
	absVenvPath, err := filepath.Abs(venvPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for venv: %w", err)
	}

	// Determine the Python executable path
	venvPython := filepath.Join(absVenvPath, "bin", "python")
	if runtime.GOOS == OSWindows {
		venvPython = filepath.Join(absVenvPath, "Scripts", "python.exe")
	}

	// Verify the Python executable exists and works
	if _, err := os.Stat(venvPython); os.IsNotExist(err) {
		return "", fmt.Errorf("virtual environment Python executable not found: %s", venvPython)
	}

	// Upgrade pip in the virtual environment
	fmt.Println("Upgrading pip in virtual environment...")
	cmd := CommandRunner(venvPython, "-m", "pip", "install", "--upgrade", "pip")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to upgrade pip: %w", err)
	}

	return absVenvPath, nil
}

func CheckPythonVersion() error {
	cmd := CommandRunner("python3", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("python3 not found: %w", err)
	}

	version := strings.TrimSpace(string(output))
	version = strings.TrimPrefix(version, "Python ")

	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return fmt.Errorf("unable to parse Python version: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("unable to parse major version: %w", err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("unable to parse minor version: %w", err)
	}

	if major < 3 || (major == 3 && minor < 10) {
		return fmt.Errorf("python 3.10+ required, found %s", version)
	}

	return nil
}

func checkNpm() error {
	cmd := CommandRunner("npm", "--version")
	return cmd.Run()
}

func EnsureNodeAndNpm() error {
	// Check if both node and npm are available
	nodeErr := checkNodeJS()
	npmErr := checkNpm()

	if nodeErr != nil || npmErr != nil {
		fmt.Println("Node.js or npm not found. Installing via NVM...")

		// Install NVM if not present
		nvmDir := filepath.Join(os.Getenv("HOME"), ".nvm")
		if _, err := os.Stat(nvmDir); os.IsNotExist(err) {
			cmd := CommandRunner("bash", "-c", "curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to install NVM: %w", err)
			}
		}

		// Install Node.js LTS via NVM with proper shell sourcing
		install := `
			source ~/.nvm/nvm.sh || exit 1
			nvm install --lts
			nvm use --lts
		`
		cmd := CommandRunner("bash", "-lc", install)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install node.js via NVM: %w", err)
		}

		// Verify installation
		if err := checkNodeJS(); err != nil {
			return fmt.Errorf("node.js installation verification failed: %w", err)
		}
		if err := checkNpm(); err != nil {
			return fmt.Errorf("npm installation verification failed: %w", err)
		}
	}

	return nil
}

func checkNodeJS() error {
	cmd := CommandRunner("node", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("node.js not found: %w", err)
	}
	return nil
}

func CheckYarn() error {
	cmd := CommandRunner("yarn", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yarn not found: %w", err)
	}
	return nil
}

func InstallYarn() error {
	fmt.Println("Yarn not found. Installing Yarn...")

	// Try npm install first (with proper NVM sourcing and timeout)
	cmd := CommandRunner("bash", "-lc", "source ~/.nvm/nvm.sh && npm install -g yarn --silent")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("npm install failed: %v, trying system package managers...\n", err)

		// Fallback to system package manager based on OS
		switch runtime.GOOS {
		case "darwin":
			// macOS - use Homebrew
			fmt.Println("Trying Homebrew installation...")
			cmd = CommandRunner("bash", "-lc", "brew install yarn --quiet")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				// Try corepack as last resort (available in Node.js 16.10+)
				fmt.Println("Homebrew failed, trying corepack...")
				cmd = CommandRunner("bash", "-lc", "source ~/.nvm/nvm.sh && corepack enable")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("failed to install yarn via Homebrew or corepack: %w", err)
				}
			}
		case "linux":
			// Linux - use modern apt approach
			fmt.Println("Trying apt installation...")
			installScript := `
				set -e
				echo "Adding Yarn repository..."
				curl -fsSL https://dl.yarnpkg.com/debian/pubkey.gpg | sudo gpg --dearmor -o /usr/share/keyrings/yarnkey.gpg
				echo "deb [signed-by=/usr/share/keyrings/yarnkey.gpg] https://dl.yarnpkg.com/debian/ stable main" | \
					sudo tee /etc/apt/sources.list.d/yarn.list
				echo "Updating package list..."
				sudo apt update -qq
				echo "Installing Yarn..."
				sudo apt install -y yarn -qq
			`
			cmd = CommandRunner("bash", "-c", installScript)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to install yarn via apt: %w", err)
			}
		default:
			return fmt.Errorf("unsupported OS for yarn installation: %s", runtime.GOOS)
		}
	}

	// Verify installation
	if err := CheckYarn(); err != nil {
		return fmt.Errorf("yarn installation verification failed: %w", err)
	}

	return nil
}

// Env handles all environment setup
func Env() (string, error) {
	// Ensure we're in the correct repository
	if err := EnsureRepo(); err != nil {
		return "", fmt.Errorf("failed to ensure repository: %w", err)
	}

	// Check Python version
	fmt.Println("Checking Python version...")
	if err := CheckPythonVersion(); err != nil {
		return "", fmt.Errorf("python version check failed: %w", err)
	}
	fmt.Println("Python version OK")

	// Ensure Node.js and npm are available
	fmt.Println("Checking Node.js and npm...")
	if err := EnsureNodeAndNpm(); err != nil {
		return "", fmt.Errorf("node.js/npm setup failed: %w", err)
	}
	fmt.Println("Node.js and npm OK")

	// Check for Yarn and install if missing
	fmt.Println("Checking for Yarn...")
	if err := CheckYarn(); err != nil {
		if err := InstallYarn(); err != nil {
			return "", fmt.Errorf("yarn installation failed: %w", err)
		}
	}
	fmt.Println("Yarn is available.")

	// Ensure virtual environment
	venvPath, err := EnsureVenv()
	if err != nil {
		return "", fmt.Errorf("virtual environment setup failed: %w", err)
	}

	return venvPath, nil
}
