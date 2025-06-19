package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
)

// Version information
var (
	Version   = "1.0.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

// Constants
const (
	venvName = "gswarm-venv"

	DefaultPublicMaddr = "/ip4/38.101.215.13/tcp/30002/p2p/QmQ2gEXoPJg6iMBSUFWGzAabS2VhnzuS782Y637hGjfsRJ"
	DefaultPeerMaddr   = "/ip4/38.101.215.13/tcp/30002/p2p/QmQ2gEXoPJg6iMBSUFWGzAabS2VhnzuS782Y637hGjfsRJ"
	DefaultHostMaddr   = "/ip4/0.0.0.0/tcp/38331"
	SmallSwarmContract = "0x69C6e1D608ec64885E7b185d39b04B491a71768C"
	BigSwarmContract   = "0x6947c6E196a48B77eFa9331EC1E3e45f3Ee5Fd58"

	// OS constants
	OSDarwin  = "darwin"
	OSLinux   = "linux"
	OSWindows = "windows"

	// Game constants
	GameDapo  = "dapo"
	GameGSM8K = "gsm8k"

	// Response constants
	ResponseNone = "None"
	ResponseYes  = "yes"
)

var errorMarkers = []string{
	">> An error was detected while running rl-swarm.",
	">> Shutting down trainer...",
	"Error:",
	"Exception:",
	"Traceback:",
	"is already taken by another user",
}

// Configuration holds all the settings for the RL Swarm
type Configuration struct {
	ConnectToTestnet bool
	UseBigSwarm      bool
	ParamB           string
	CPUOnly          bool
	HFToken          string
	OrgID            string
	IdentityPath     string
	ContractAddress  string
	Game             string
	ConfigPath       string
	PublicMaddr      string
	PeerMaddr        string
	HostMaddr        string
	RequirementsFile string
}

func printBanner() {
	banner := `
 ██████  ███████ ██     ██  █████  ██████  ███    ███ 
██       ██      ██     ██ ██   ██ ██   ██ ████  ████ 
██   ███ ███████ ██  █  ██ ███████ ██████  ██ ████ ██ 
██    ██      ██ ██ ███ ██ ██   ██ ██   ██ ██  ██  ██ 
 ██████  ███████  ███ ███  ██   ██ ██   ██ ██      ██ 
		G-SWARM Supervisor (Community Project)
`
	fmt.Println("\033[38;5;224m")
	fmt.Println(banner)
	fmt.Println("\033[0m")
}

// ensureRepo ensures we're in the correct repository
func ensureRepo() error {
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		fmt.Println("Not in RL-Swarm repository. Cloning...")

		// Check if git is available
		if err := checkGit(); err != nil {
			if err := installGit(); err != nil {
				return fmt.Errorf("failed to install git: %w", err)
			}
		}

		cmd := exec.Command("git", "clone", "https://github.com/gensyn-ai/rl-swarm.git")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to clone rl-swarm: %w", err)
		}

		if err := os.Chdir("rl-swarm"); err != nil {
			return fmt.Errorf("failed to change to rl-swarm directory: %w", err)
		}
		fmt.Println("Successfully cloned and entered RL-Swarm repository")
	}
	return nil
}

func checkGit() error {
	cmd := exec.Command("git", "--version")
	return cmd.Run()
}

func installGit() error {
	fmt.Println("Git not found. Installing...")

	switch runtime.GOOS {
	case OSDarwin:
		cmd := exec.Command("brew", "install", "git")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case OSLinux:
		cmd := exec.Command("sudo", "apt-get", "update")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to update package list: %w", err)
		}

		cmd = exec.Command("sudo", "apt-get", "install", "-y", "git")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported OS for git installation: %s", runtime.GOOS)
	}
}

// ensureVenv ensures the Python virtual environment exists and is properly set up
func ensureVenv() (string, error) {
	venvPath := venvName

	// Check if venv already exists
	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		fmt.Printf("Creating virtual environment: %s\n", venvPath)

		cmd := exec.Command("python3", "-m", "venv", venvPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to create virtual environment: %w", err)
		}
	}

	// Determine the Python executable path
	venvPython := filepath.Join(venvPath, "bin", "python")
	if runtime.GOOS == OSWindows {
		venvPython = filepath.Join(venvPath, "Scripts", "python.exe")
	}

	// Verify the Python executable exists and works
	if _, err := os.Stat(venvPython); os.IsNotExist(err) {
		return "", fmt.Errorf("virtual environment Python executable not found: %s", venvPython)
	}

	// Upgrade pip in the virtual environment
	fmt.Println("Upgrading pip in virtual environment...")
	cmd := exec.Command(venvPython, "-m", "pip", "install", "--upgrade", "pip")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to upgrade pip: %w", err)
	}

	return venvPath, nil
}

func checkPythonVersion() error {
	// Check if python3 is available
	cmd := exec.Command("python3", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("python3 not found: %w", err)
	}

	// Get Python version
	cmd = exec.Command("python3", "-c", "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get Python version: %w", err)
	}

	versionStr := strings.TrimSpace(string(output))
	parts := strings.Split(versionStr, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid Python version format: %s", versionStr)
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
		return fmt.Errorf("python 3.10+ required, found %s", versionStr)
	}

	return nil
}

func checkNpm() error {
	cmd := exec.Command("npm", "--version")
	return cmd.Run()
}

func ensureNodeAndNpm() error {
	// Check if both node and npm are available
	nodeErr := checkNodeJS()
	npmErr := checkNpm()

	if nodeErr != nil || npmErr != nil {
		fmt.Println("Node.js or npm not found. Installing via NVM...")

		// Install NVM
		fmt.Println("Installing NVM...")
		cmd := exec.Command("bash", "-c", "curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install NVM: %w", err)
		}

		// Source NVM and install Node.js
		fmt.Println("Installing Node.js via NVM...")
		cmd = exec.Command("bash", "-c", "source ~/.nvm/nvm.sh && nvm install 18 && nvm use 18")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install Node.js via NVM: %w", err)
		}

		// Verify Node.js installation
		cmd = exec.Command("bash", "-c", "source ~/.nvm/nvm.sh && node --version")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("node.js installation verification failed: %w", err)
		}

		// Verify npm installation
		cmd = exec.Command("bash", "-c", "source ~/.nvm/nvm.sh && npm --version")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("npm installation verification failed: %w", err)
		}
	}

	return nil
}

func checkNodeJS() error {
	cmd := exec.Command("node", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("node.js not found: %w", err)
	}
	return nil
}

func checkYarn() error {
	cmd := exec.Command("yarn", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yarn not found: %w", err)
	}
	return nil
}

func installYarn() error {
	fmt.Println("Yarn not found. Installing Yarn...")

	// Try npm install first (with proper NVM sourcing and timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Use npm with network-friendly options and proper shell sourcing
	cmd := exec.CommandContext(ctx, "bash", "-lc", "source ~/.nvm/nvm.sh && npm install -g yarn --silent")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("npm install failed: %v, trying system package managers...\n", err)

		// Fallback to system package manager based on OS
		switch runtime.GOOS {
		case "darwin":
			// macOS - use Homebrew
			fmt.Println("Trying Homebrew installation...")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			cmd = exec.CommandContext(ctx, "bash", "-lc", "brew install yarn --quiet")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				// Try corepack as last resort (available in Node.js 16.10+)
				fmt.Println("Homebrew failed, trying corepack...")
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				cmd = exec.CommandContext(ctx, "bash", "-lc", "source ~/.nvm/nvm.sh && corepack enable")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("failed to install yarn via Homebrew or corepack: %w", err)
				}
			}
		case "linux":
			// Linux - use modern apt approach
			fmt.Println("Trying apt installation...")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
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
			cmd = exec.CommandContext(ctx, "bash", "-c", installScript)
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
	if err := checkYarn(); err != nil {
		return fmt.Errorf("yarn installation verification failed: %w", err)
	}

	return nil
}

func setupModalLogin(_ Configuration) (string, error) {
	fmt.Println("\n=== Modal Login Setup ===")
	fmt.Println("To connect to the testnet, you need to authenticate with Modal.")
	fmt.Println("This will open your browser to complete the login process.")

	// Check if modal CLI is available
	cmd := exec.Command("modal", "token", "new")
	if err := cmd.Run(); err != nil {
		fmt.Println("Modal CLI not found. Installing...")

		// Install modal CLI
		installCmd := exec.Command("pip", "install", "modal")
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return "", fmt.Errorf("failed to install modal CLI: %w", err)
		}
	}

	// Get the login URL
	cmd = exec.Command("modal", "token", "new", "--json")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get modal login URL: %w", err)
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse modal response: %w", err)
	}

	fmt.Printf("Opening browser to: %s\n", result.URL)
	openBrowser(result.URL)

	// Wait for user to complete login
	fmt.Println("Please complete the login in your browser, then press Enter here...")
	if _, err := fmt.Scanln(); err != nil {
		return "", fmt.Errorf("failed to read user input: %w", err)
	}

	// Get the token
	cmd = exec.Command("modal", "token", "current", "--json")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current modal token: %w", err)
	}

	var tokenResult struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(output, &tokenResult); err != nil {
		return "", fmt.Errorf("failed to parse modal token response: %w", err)
	}

	if tokenResult.Token == "" {
		return "", fmt.Errorf("no modal token found after login")
	}

	// Get the org ID
	cmd = exec.Command("modal", "org", "current", "--json")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current modal org: %w", err)
	}

	var orgResult struct {
		OrgID string `json:"org_id"`
	}
	if err := json.Unmarshal(output, &orgResult); err != nil {
		return "", fmt.Errorf("failed to parse modal org response: %w", err)
	}

	if orgResult.OrgID == "" {
		return "", fmt.Errorf("no modal org ID found after login")
	}

	fmt.Printf("Successfully authenticated with Modal (Org ID: %s)\n", orgResult.OrgID)
	return orgResult.OrgID, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		fmt.Printf("Please open this URL in your browser: %s\n", url)
		return
	}
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
		fmt.Printf("Please open this URL manually: %s\n", url)
	}
}

func installRequirements(venvPath string, requirementsFile string, _ *log.Logger) error {
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
	cmd := exec.Command(venvPython, "-m", "pip", "install", "-r", requirementsFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install requirements: %w", err)
	}

	return nil
}

func isCPUOnly() bool {
	// Check if CUDA is available
	cmd := exec.Command("nvidia-smi")
	return cmd.Run() != nil
}

func getConfigPath(paramB string, useBigSwarm bool) string {
	baseDir := "configs"
	if useBigSwarm {
		baseDir = filepath.Join(baseDir, "big_swarm")
	} else {
		baseDir = filepath.Join(baseDir, "small_swarm")
	}
	return filepath.Join(baseDir, fmt.Sprintf("%sB.yaml", paramB))
}

func promptUser(prompt string, defaultValue string, validOptions []string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\033[32m%s [%s]: \033[0m", prompt, defaultValue)
	input, err := reader.ReadString('\n')
	if err != nil {
		// If we can't read from stdin, return default value
		return defaultValue
	}
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
			return promptUser(prompt, defaultValue, validOptions)
		}
	}

	return input
}

func promptYesNo(prompt string, defaultValue string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\033[32m%s [%s]: \033[0m", prompt, defaultValue)
	input, err := reader.ReadString('\n')
	if err != nil {
		// If we can't read from stdin, return default value
		return strings.ToLower(defaultValue) == "y" || strings.ToLower(defaultValue) == ResponseYes
	}
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		input = strings.ToLower(defaultValue)
	}

	return input == "y" || input == "yes"
}

func promptChoice(prompt string, options map[string]string, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\033[32m%s\n", prompt)
	for key, value := range options {
		fmt.Printf("  %s: %s\n", key, value)
	}
	fmt.Printf("Choice [%s]: \033[0m", defaultValue)

	input, err := reader.ReadString('\n')
	if err != nil {
		// If we can't read from stdin, return default value
		return defaultValue
	}
	input = strings.TrimSpace(strings.ToUpper(input))

	if input == "" {
		input = strings.ToUpper(defaultValue)
	}

	if value, exists := options[input]; exists {
		return value
	}

	fmt.Println("Invalid choice. Please try again.")
	return promptChoice(prompt, options, defaultValue)
}

// getConfiguration builds a Configuration from CLI context
func getConfiguration(c *cli.Context) Configuration {
	cfg := Configuration{
		PublicMaddr: DefaultPublicMaddr,
		PeerMaddr:   DefaultPeerMaddr,
		HostMaddr:   DefaultHostMaddr,
	}

	// Get values from CLI context
	cfg.ConnectToTestnet = c.Bool("testnet")
	cfg.UseBigSwarm = c.Bool("big-swarm")
	cfg.ParamB = c.String("model-size")
	cfg.HFToken = c.String("hf-token")
	cfg.OrgID = c.String("org-id")
	cfg.IdentityPath = c.String("identity-path")
	cfg.ContractAddress = c.String("contract-address")
	cfg.Game = c.String("game")
	cfg.ConfigPath = c.String("config-path")
	cfg.CPUOnly = c.Bool("cpu-only")
	cfg.RequirementsFile = c.String("requirements")

	// Set defaults for unset values
	if cfg.IdentityPath == "" {
		cfg.IdentityPath = "swarm.pem"
	}

	// Set CPUOnly based on flag or detection
	if !cfg.CPUOnly {
		cfg.CPUOnly = isCPUOnly()
	}

	// Set contract address based on swarm type if not provided
	if cfg.ContractAddress == "" {
		if cfg.UseBigSwarm {
			cfg.ContractAddress = BigSwarmContract
		} else {
			cfg.ContractAddress = SmallSwarmContract
		}
	}

	// Set game type based on swarm type if not provided
	if cfg.Game == "" {
		if cfg.UseBigSwarm {
			cfg.Game = GameDapo
		} else {
			cfg.Game = GameGSM8K
		}
	}

	// Set config path if not provided
	if cfg.ConfigPath == "" {
		cfg.ConfigPath = getConfigPath(cfg.ParamB, cfg.UseBigSwarm)
	}

	// Force testnet if org-id is provided
	if cfg.OrgID != "" {
		cfg.ConnectToTestnet = true
	}

	return cfg
}

// promptForMissingConfiguration prompts for any missing required configuration
func promptForMissingConfiguration(cfg Configuration) Configuration {
	// Prompt for testnet if not set
	if !cfg.ConnectToTestnet {
		cfg.ConnectToTestnet = promptYesNo("Would you like to connect to the Testnet?", "Y")
	}

	// Prompt for big swarm if not set
	if !cfg.UseBigSwarm {
		choice := promptChoice(
			"Which swarm would you like to join (Math (A) or Math Hard (B))?",
			map[string]string{"A": "Math (small swarm)", "B": "Math Hard (big swarm)"},
			"A",
		)
		cfg.UseBigSwarm = (choice == "Math Hard (big swarm)")
	}

	// Prompt for model size if not set
	if cfg.ParamB == "" {
		cfg.ParamB = promptUser(
			"How many parameters (in billions)? [0.5,1.5,7,32,72]",
			"0.5",
			[]string{"0.5", "1.5", "7", "32", "72"},
		)
	}

	// Prompt for HuggingFace token if not set
	if cfg.HFToken == "" {
		cfg.HFToken = promptHFToken()
	}

	// Update derived values based on prompts
	if cfg.ContractAddress == "" {
		if cfg.UseBigSwarm {
			cfg.ContractAddress = BigSwarmContract
		} else {
			cfg.ContractAddress = SmallSwarmContract
		}
	}

	if cfg.Game == "" {
		if cfg.UseBigSwarm {
			cfg.Game = GameDapo
		} else {
			cfg.Game = GameGSM8K
		}
	}

	if cfg.ConfigPath == "" {
		cfg.ConfigPath = getConfigPath(cfg.ParamB, cfg.UseBigSwarm)
	}

	return cfg
}

func promptHFToken() string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\033[32mWould you like to push models you train in the RL swarm to the Hugging Face Hub? [y/N]: \033[0m")
	input, err := reader.ReadString('\n')
	if err != nil {
		// If we can't read from stdin, return "None"
		return ResponseNone
	}
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		input = "n"
	}

	if input == "y" || input == "yes" {
		fmt.Print("Enter your HuggingFace access token: ")
		token, err := reader.ReadString('\n')
		if err != nil {
			// If we can't read the token, return "None"
			return ResponseNone
		}
		return strings.TrimSpace(token)
	}

	return ResponseNone
}

func runPythonTraining(config Configuration, venvPath string, logger *log.Logger) error {
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

	cmd := exec.Command(venvPython, args...)

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

func cleanupStaleProcesses(logger *log.Logger) {
	fmt.Println("Cleaning up stale gensyn processes...")
	logger.Printf("Cleaning up stale gensyn processes")

	// Kill any existing gensyn processes
	cmd := exec.Command("pkill", "-f", "gensyn")
	if err := cmd.Run(); err != nil {
		// pkill returns error if no processes found, which is fine
		fmt.Println("No existing gensyn processes found")
		logger.Printf("No existing gensyn processes found")
	} else {
		fmt.Println("Killed existing gensyn processes")
		logger.Printf("Killed existing gensyn processes")
	}

	// Also try to kill hivemind processes
	cmd = exec.Command("pkill", "-f", "hivemind")
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
	cmd = exec.Command("pgrep", "-f", "gensyn")
	if err := cmd.Run(); err == nil {
		// Still have processes, try force kill
		fmt.Println("Force killing remaining gensyn processes...")
		logger.Printf("Force killing remaining gensyn processes")
		if err := exec.Command("pkill", "-9", "-f", "gensyn").Run(); err != nil {
			logger.Printf("Failed to force kill gensyn processes: %v", err)
		}
		if err := exec.Command("pkill", "-9", "-f", "hivemind").Run(); err != nil {
			logger.Printf("Failed to force kill hivemind processes: %v", err)
		}
		time.Sleep(1 * time.Second)
	}
}

// bootstrapEnv handles all environment setup
func bootstrapEnv() (string, error) {
	// Ensure we're in the correct repository
	if err := ensureRepo(); err != nil {
		return "", fmt.Errorf("failed to ensure repository: %w", err)
	}

	// Check Python version
	fmt.Println("Checking Python version...")
	if err := checkPythonVersion(); err != nil {
		return "", fmt.Errorf("python version check failed: %w", err)
	}
	fmt.Println("Python version OK")

	// Ensure Node.js and npm are available
	fmt.Println("Checking Node.js and npm...")
	if err := ensureNodeAndNpm(); err != nil {
		return "", fmt.Errorf("node.js/npm setup failed: %w", err)
	}
	fmt.Println("Node.js and npm OK")

	// Check for Yarn and install if missing
	fmt.Println("Checking for Yarn...")
	if err := checkYarn(); err != nil {
		if err := installYarn(); err != nil {
			return "", fmt.Errorf("yarn installation failed: %w", err)
		}
	}
	fmt.Println("Yarn is available.")

	// Ensure virtual environment
	venvPath, err := ensureVenv()
	if err != nil {
		return "", fmt.Errorf("virtual environment setup failed: %w", err)
	}

	return venvPath, nil
}

// configure handles CLI parsing and interactive configuration
func configure(c *cli.Context) (Configuration, error) {
	// Build configuration from CLI context
	config := getConfiguration(c)

	// Prompt for missing configuration if in interactive mode
	if c.Bool("interactive") || !c.IsSet("testnet") {
		config = promptForMissingConfiguration(config)
	}

	// Validate configuration
	if err := validateConfiguration(config); err != nil {
		return Configuration{}, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Handle modal login if connecting to testnet but no org-id
	if config.ConnectToTestnet && config.OrgID == "" {
		orgID, err := setupModalLogin(config)
		if err != nil {
			return Configuration{}, fmt.Errorf("modal login failed: %w", err)
		}
		config.OrgID = orgID
	}

	return config, nil
}

// validateConfiguration validates the configuration
func validateConfiguration(config Configuration) error {
	// Validate ParamB
	validParamBs := []string{"0.5", "1.5", "7", "32", "72"}
	valid := false
	for _, validParam := range validParamBs {
		if config.ParamB == validParam {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid paramB: %s (must be one of %v)", config.ParamB, validParamBs)
	}

	// Validate Game
	if config.Game != "gsm8k" && config.Game != "dapo" {
		return fmt.Errorf("invalid game: %s (must be 'gsm8k' or 'dapo')", config.Game)
	}

	return nil
}

// runSupervisor handles the main training loop
func runSupervisor(config Configuration, venvPath string) error {
	// Setup logging
	if err := os.MkdirAll("logs", 0o755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}
	logFile, err := os.OpenFile("logs/gensyn_rl_swarm_go.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()
	logger := log.New(logFile, "", log.LstdFlags|log.Lmicroseconds)

	// Install requirements
	fmt.Println("Getting requirements...")
	if err := installRequirements(venvPath, config.RequirementsFile, logger); err != nil {
		return fmt.Errorf("failed to install requirements: %w", err)
	}
	fmt.Println("Done!")

	fmt.Println("Good luck in the swarm!")
	fmt.Println("Post about rl-swarm on X/twitter! --> https://tinyurl.com/swarmtweet")
	fmt.Println("And remember to star the repo on GitHub! --> https://github.com/gensyn-ai/rl-swarm")

	// Setup signal handling
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	restartCh := make(chan struct{}, 1)
	restartCh <- struct{}{}

	initialBackoff := 5 * time.Second
	maxBackoff := 5 * time.Minute
	backoff := initialBackoff

runloop:
	for {
		select {
		case <-ctx.Done():
			logger.Println("Shutdown signal; exiting.")
			break runloop

		case <-restartCh:
			logger.Println("Starting Python training process...")
			fmt.Println("Starting RL Swarm training...")

			err := runPythonTraining(config, venvPath, logger)
			if err != nil {
				logger.Printf("Training process exited with error: %v", err)
				fmt.Printf("Training process exited with error: %v\n", err)

				// Check if this is an identity conflict
				if strings.Contains(err.Error(), "identity conflict detected") {
					fmt.Println("Identity conflict detected! Cleaning up stale processes and retrying...")
					logger.Printf("Identity conflict detected, cleaning up stale processes")

					// Clean up stale processes
					cleanupStaleProcesses(logger)

					// Wait a bit longer before retry for identity conflicts
					fmt.Println("Waiting 10 seconds before retry...")
					time.Sleep(10 * time.Second)

					// Reset backoff for identity conflicts since we cleaned up
					backoff = initialBackoff
				} else {
					// Regular error, use exponential backoff
					time.Sleep(backoff)
					backoff = minDuration(backoff*2, maxBackoff)
				}

				nonBlockingSend(restartCh)
			} else {
				logger.Println("Training process exited cleanly.")
				backoff = initialBackoff // reset on clean exit
			}
		}
	}

	return nil
}

func main() {
	app := createCLIApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createCLIApp() *cli.App {
	app := &cli.App{
		Name:    "gswarm",
		Usage:   "Gensyn RL Swarm Supervisor - A robust supervisor for Gensyn RL Swarm",
		Version: Version,
		Authors: []*cli.Author{
			{
				Name:  "Deep-Commit Community",
				Email: "community@deep-commit.com",
			},
		},
		Copyright:   "© 2024 Deep-Commit Community. This is a third-party application not affiliated with Gensyn.",
		Description: getAppDescription(),
		Flags:       getAppFlags(),
		Action:      getMainAction(),
		Commands:    getAppCommands(),
		Before:      getBeforeFunc(),
	}
	return app
}

func getAppDescription() string {
	return `GSwarm is a robust Go-based supervisor for Gensyn RL Swarm that provides 
automatic restart capabilities, dependency management, and comprehensive logging.

Features:
• Auto-restart on errors with exponential backoff
• Comprehensive logging with timestamps
• Python environment management
• Interactive CLI with fallback prompts
• Performance monitoring and error detection
• Graceful shutdown handling
• Support for both testnet and mainnet

This is a community-developed tool designed to enhance the user experience of running Gensyn RL Swarm.`
}

func getAppFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "testnet",
			Usage:   "Connect to the Testnet",
			EnvVars: []string{"GSWARM_TESTNET"},
		},
		&cli.BoolFlag{
			Name:    "big-swarm",
			Usage:   "Use big swarm (Math Hard) instead of small swarm (Math)",
			EnvVars: []string{"GSWARM_BIG_SWARM"},
		},
		&cli.StringFlag{
			Name:    "model-size",
			Usage:   "Parameter count in billions",
			Value:   "0.5",
			EnvVars: []string{"GSWARM_MODEL_SIZE"},
			Action:  validateModelSize,
		},
		&cli.StringFlag{
			Name:    "hf-token",
			Usage:   "HuggingFace access token for model pushing",
			EnvVars: []string{"HUGGINGFACE_ACCESS_TOKEN", "GSWARM_HF_TOKEN"},
		},
		&cli.StringFlag{
			Name:    "org-id",
			Usage:   "Modal ORG_ID (required for testnet)",
			EnvVars: []string{"GSWARM_ORG_ID"},
		},
		&cli.StringFlag{
			Name:    "identity-path",
			Usage:   "Path to identity PEM file",
			Value:   "swarm.pem",
			EnvVars: []string{"GSWARM_IDENTITY_PATH"},
		},
		&cli.StringFlag{
			Name:    "contract-address",
			Usage:   "Override smart contract address",
			EnvVars: []string{"GSWARM_CONTRACT_ADDRESS"},
		},
		&cli.StringFlag{
			Name:    "game",
			Usage:   "Game type ('gsm8k' or 'dapo')",
			EnvVars: []string{"GSWARM_GAME"},
			Action:  validateGame,
		},
		&cli.StringFlag{
			Name:    "config-path",
			Usage:   "Path to YAML config file",
			EnvVars: []string{"GSWARM_CONFIG_PATH"},
		},
		&cli.BoolFlag{
			Name:    "cpu-only",
			Usage:   "Force CPU-only mode",
			EnvVars: []string{"GSWARM_CPU_ONLY"},
		},
		&cli.StringFlag{
			Name:    "requirements",
			Usage:   "Requirements file path (overrides default)",
			EnvVars: []string{"GSWARM_REQUIREMENTS"},
		},
		&cli.BoolFlag{
			Name:    "interactive",
			Usage:   "Force interactive mode (prompt for all options)",
			EnvVars: []string{"GSWARM_INTERACTIVE"},
		},
	}
}

func validateModelSize(c *cli.Context, v string) error {
	validSizes := []string{"0.5", "1.5", "7", "32", "72"}
	for _, size := range validSizes {
		if v == size {
			return nil
		}
	}
	return fmt.Errorf("model-size must be one of: %v", validSizes)
}

func validateGame(c *cli.Context, v string) error {
	if v != "gsm8k" && v != "dapo" {
		return fmt.Errorf("game must be 'gsm8k' or 'dapo'")
	}
	return nil
}

func getMainAction() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		fmt.Println("Starting RL Swarm Supervisor...")

		// Print banner
		printBanner()

		// Bootstrap environment
		venvPath, err := bootstrapEnv()
		if err != nil {
			return cli.Exit(fmt.Sprintf("Environment bootstrap failed: %v", err), 1)
		}

		// Configure
		config, err := configure(c)
		if err != nil {
			return cli.Exit(fmt.Sprintf("Configuration failed: %v", err), 1)
		}

		// Run supervisor
		if err := runSupervisor(config, venvPath); err != nil {
			return cli.Exit(fmt.Sprintf("Supervisor failed: %v", err), 1)
		}

		return nil
	}
}

func getAppCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Show detailed version information",
			Action:  getVersionAction(),
		},
	}
}

func getVersionAction() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		fmt.Printf("GSwarm version %s\n", Version)
		fmt.Printf("Build date: %s\n", BuildDate)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return nil
	}
}

func getBeforeFunc() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		// Set up custom help template
		cli.AppHelpTemplate = getHelpTemplate()
		return nil
	}
}

func getHelpTemplate() string {
	return `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} \
   {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
   {{if len .Authors}}
AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author.Name}}{{if $author.Email}} <{{$author.Email}}>{{end}}{{end}}
   {{end}}{{if .Commands}}
COMMANDS:{{range .CommandCategories}}
   {{.Name}}:{{range .Commands}}
     {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
GLOBAL OPTIONS:
   {{range $index, $option := .VisibleFlags}}{{if $index}}
   {{end}}{{$option}}{{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}{{if .Version}}
VERSION:
   {{.Version}}
   {{end}}
EXAMPLES:
   # Interactive mode (default)
   gswarm

   # Non-interactive mode with all options
   gswarm --testnet --big-swarm --model-size 7 --org-id YOUR_ORG_ID --hf-token YOUR_TOKEN

   # CPU-only mode
   gswarm --cpu-only --model-size 0.5

   # Custom requirements file
   gswarm --requirements requirements-gpu.txt

   # Show version
   gswarm version

ENVIRONMENT VARIABLES:
   All flags can be set via environment variables with the GSWARM_ prefix.
   For example: GSWARM_TESTNET=true, GSWARM_MODEL_SIZE=7, etc.

   Special environment variables:
   • HUGGINGFACE_ACCESS_TOKEN: For HuggingFace token (no prefix needed)
   • GSWARM_ORG_ID: Modal organization ID for testnet access

LEARN MORE:
   • GitHub: https://github.com/Deep-Commit/gswarm
   • Documentation: https://github.com/Deep-Commit/gswarm#readme
   • Community: https://github.com/Deep-Commit/gswarm/discussions
`
}

func nonBlockingSend(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
