package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Version information
var (
	Version   = "1.0.0"
	BuildDate = "unknown"
	GitCommit = "unknown"
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
}

// Default values
const (
	DefaultPeerMaddr   = "/ip4/38.101.215.13/tcp/30002/p2p/QmQ2gEXoPJg6iMBSUFWGzAabS2VhnzuS782Y637hGjfsRJ"
	DefaultHostMaddr   = "/ip4/0.0.0.0/tcp/38331"
	SmallSwarmContract = "0x69C6e1D608ec64885E7b185d39b04B491a71768C"
	BigSwarmContract   = "0x6947c6E196a48B77eFa9331EC1E3e45f3Ee5Fd58"
)

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

func checkPythonVersion() error {
	cmd := exec.Command("python3", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("python3 not found: %v", err)
	}

	version := strings.TrimSpace(string(output))
	version = strings.TrimPrefix(version, "Python ")

	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return fmt.Errorf("unable to parse Python version: %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("unable to parse major version: %v", err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("unable to parse minor version: %v", err)
	}

	if major < 3 || (major == 3 && minor < 10) {
		return fmt.Errorf("Python 3.10+ required, found %s", version)
	}

	return nil
}

func checkNodeJS() error {
	cmd := exec.Command("node", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Node.js not found: %v", err)
	}
	return nil
}

func installNodeJS() error {
	fmt.Println("Node.js not found. Installing NVM and latest Node.js...")

	// Install NVM
	nvmDir := filepath.Join(os.Getenv("HOME"), ".nvm")
	if _, err := os.Stat(nvmDir); os.IsNotExist(err) {
		cmd := exec.Command("bash", "-c", "curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install NVM: %v", err)
		}
	}

	// Source NVM and install Node.js
	cmd := exec.Command("bash", "-c", "source ~/.nvm/nvm.sh && nvm install node")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func checkYarn() error {
	cmd := exec.Command("yarn", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Yarn not found: %v", err)
	}
	return nil
}

func installYarn() error {
	fmt.Println("Yarn not found. Installing Yarn...")

	// Try npm install first (with proper NVM sourcing and timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Use npm with network-friendly options
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
					return fmt.Errorf("failed to install Yarn via Homebrew or corepack: %v", err)
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
				echo "deb [signed-by=/usr/share/keyrings/yarnkey.gpg] https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
				echo "Updating package list..."
				sudo apt update -qq
				echo "Installing Yarn..."
				sudo apt install -y yarn -qq
			`
			cmd = exec.CommandContext(ctx, "bash", "-c", installScript)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				// Try corepack as fallback
				fmt.Println("apt failed, trying corepack...")
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				cmd = exec.CommandContext(ctx, "bash", "-lc", "source ~/.nvm/nvm.sh && corepack enable")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("failed to install Yarn via apt or corepack: %v", err)
				}
			}
		case "windows":
			// Windows - try chocolatey first, then winget
			fmt.Println("Trying Chocolatey installation...")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			cmd = exec.CommandContext(ctx, "powershell", "-Command", "choco install yarn -y")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println("Chocolatey failed, trying winget...")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()
				cmd = exec.CommandContext(ctx, "winget", "install", "Yarn.Yarn", "--silent")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					// Try corepack as last resort
					fmt.Println("winget failed, trying corepack...")
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					defer cancel()
					cmd = exec.CommandContext(ctx, "powershell", "-Command", "corepack enable")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Run(); err != nil {
						return fmt.Errorf("failed to install Yarn via Windows package managers or corepack: %v", err)
					}
				}
			}
		default:
			return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
		}
	}

	// Verify installation
	fmt.Println("Verifying Yarn installation...")
	if err := checkYarn(); err != nil {
		return fmt.Errorf("Yarn installation verification failed: %v", err)
	}

	fmt.Println("Yarn installed successfully!")
	return nil
}

func setupModalLogin(config Configuration) (string, error) {
	fmt.Println("Please login to create an Ethereum Server Wallet")

	// Check and install Node.js if needed
	if err := checkNodeJS(); err != nil {
		if err := installNodeJS(); err != nil {
			return "", fmt.Errorf("failed to install Node.js: %v", err)
		}
	} else {
		cmd := exec.Command("node", "--version")
		output, _ := cmd.Output()
		fmt.Printf("Node.js is already installed: %s", string(output))
	}

	// Check and install Yarn if needed
	if err := checkYarn(); err != nil {
		if err := installYarn(); err != nil {
			return "", fmt.Errorf("failed to install Yarn: %v", err)
		}
	}

	// Change to modal-login directory
	if err := os.Chdir("modal-login"); err != nil {
		return "", fmt.Errorf("failed to change to modal-login directory: %v", err)
	}
	defer os.Chdir("..")

	// Update .env file with contract address
	envFile := ".env"
	envContent, err := os.ReadFile(envFile)
	if err != nil {
		return "", fmt.Errorf("failed to read .env file: %v", err)
	}

	lines := strings.Split(string(envContent), "\n")
	if len(lines) >= 3 {
		lines[2] = fmt.Sprintf("SMART_CONTRACT_ADDRESS=%s", config.ContractAddress)
	}

	if err := os.WriteFile(envFile, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return "", fmt.Errorf("failed to update .env file: %v", err)
	}

	// Install dependencies
	fmt.Println("Installing dependencies...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "yarn", "install", "--immutable")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to install dependencies: %v", err)
	}

	// Build the project
	fmt.Println("Building server...")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd = exec.CommandContext(ctx, "yarn", "build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to build server: %v", err)
	}

	// Start the server in background
	fmt.Println("Starting modal login server...")
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, "yarn", "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start server: %v", err)
	}

	serverPID := cmd.Process.Pid
	fmt.Printf("Started server process: %d\n", serverPID)

	// Wait a bit for server to start
	time.Sleep(5 * time.Second)

	// Try to open browser
	fmt.Println("Opening browser to http://localhost:3000...")
	openBrowser("http://localhost:3000")

	// Wait for userData.json to be created
	fmt.Println("Waiting for modal userData.json to be created...")
	userDataPath := "temp-data/userData.json"
	for {
		if _, err := os.Stat(userDataPath); err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	fmt.Println("Found userData.json. Proceeding...")

	// Wait for userData.json to contain actual data (not just empty braces)
	fmt.Println("Waiting for userData.json to contain login data...")
	var orgID string
	maxWaitTime := 10 * time.Minute // Wait up to 10 minutes for user to complete login
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		userData, err := os.ReadFile(userDataPath)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		// Try to parse as JSON first
		var userDataMap map[string]interface{}
		if err := json.Unmarshal(userData, &userDataMap); err == nil {
			// Look for org_id, orgId, organization_id, etc.
			for key, value := range userDataMap {
				keyLower := strings.ToLower(key)
				if strings.Contains(keyLower, "org") && strings.Contains(keyLower, "id") {
					if str, ok := value.(string); ok && str != "" {
						orgID = str
						break
					}
				}
			}
		}

		// If JSON parsing didn't work, try the awk-like approach
		if orgID == "" {
			lines := strings.Split(string(userData), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				// Skip empty lines and lines that are just braces
				if line == "" || line == "{" || line == "}" || strings.HasPrefix(line, "{") || strings.HasPrefix(line, "}") {
					continue
				}

				// Split by quotes and look for potential org ID
				parts := strings.Split(line, "\"")
				if len(parts) >= 3 {
					// The bash script uses $(NF - 1) which is the second-to-last field
					potentialID := parts[len(parts)-2]
					if potentialID != "" && !strings.Contains(potentialID, "{") && !strings.Contains(potentialID, "}") {
						// Basic validation - should be a reasonable length and not contain obvious non-ID characters
						if len(potentialID) > 5 && len(potentialID) < 100 {
							orgID = potentialID
							break
						}
					}
				}
			}
		}

		if orgID != "" {
			break
		}

		fmt.Println("Waiting for user to complete login...")
		time.Sleep(5 * time.Second)
	}

	if orgID == "" {
		return "", fmt.Errorf("failed to extract ORG_ID from userData.json after waiting for user login")
	}

	fmt.Printf("Your ORG_ID is set to: %s\n", orgID)

	// Wait for API key to be activated
	fmt.Println("Waiting for API key to become activated...")
	for {
		resp, err := http.Get(fmt.Sprintf("http://localhost:3000/api/get-api-key-status?orgId=%s", orgID))
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if strings.TrimSpace(string(body)) == "activated" {
				fmt.Println("API key is activated! Proceeding...")
				break
			}
		}
		fmt.Println("Waiting for API key to be activated...")
		time.Sleep(5 * time.Second)
	}

	// Kill the server
	if cmd.Process != nil {
		cmd.Process.Kill()
	}

	return orgID, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		fmt.Printf("Please open %s manually in your browser\n", url)
		return
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to open %s. Please open it manually.\n", url)
	} else {
		fmt.Printf("Successfully opened %s in your default browser.\n", url)
	}
}

func installRequirements(venvPath string, requirementsFile string, logger *log.Logger) error {
	venvPython := filepath.Join(venvPath, "bin", "python")
	if runtime.GOOS == "windows" {
		venvPython = filepath.Join(venvPath, "Scripts", "python.exe")
	}

	// Create virtual environment if it doesn't exist
	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		fmt.Println("Creating virtual environment...")
		cmd := exec.Command("python3", "-m", "venv", venvPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create virtual environment: %v", err)
		}
	}

	// Upgrade pip
	fmt.Println("Upgrading pip...")
	cmd := exec.Command(venvPython, "-m", "pip", "install", "--upgrade", "pip")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to upgrade pip: %v", err)
	}

	// Determine requirements file based on CPU/GPU
	reqFile := "requirements.txt"
	if requirementsFile != "" {
		reqFile = requirementsFile
	} else {
		if isCPUOnly() {
			reqFile = "requirements-cpu.txt"
		} else {
			reqFile = "requirements-gpu.txt"
		}
	}

	fmt.Printf("Installing requirements from %s...\n", reqFile)
	cmd = exec.Command(venvPython, "-m", "pip", "install", "-r", reqFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install requirements: %v", err)
	}

	// Install flash-attn if not CPU-only
	if !isCPUOnly() {
		fmt.Println("Installing flash-attn...")
		cmd = exec.Command(venvPython, "-m", "pip", "install", "flash-attn", "--no-build-isolation")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			logger.Printf("Warning: failed to install flash-attn: %v", err)
		}
	}

	return nil
}

func isCPUOnly() bool {
	// Check if CUDA is available
	cmd := exec.Command("nvidia-smi")
	return cmd.Run() != nil
}

func getConfigPath(paramB string, useBigSwarm bool) string {
	basePath := "hivemind_exp/configs"

	if isCPUOnly() {
		// CPU-only mode
		return filepath.Join(basePath, "mac", "grpo-qwen-2.5-0.5b-deepseek-r1.yaml")
	}

	// GPU mode
	if useBigSwarm {
		basePath = filepath.Join(basePath, "gpu")
		switch paramB {
		case "32", "72":
			return filepath.Join(basePath, fmt.Sprintf("grpo-qwen-2.5-%sb-bnb-4bit-deepseek-r1.yaml", paramB))
		case "0.5", "1.5", "7":
			return filepath.Join(basePath, fmt.Sprintf("grpo-qwen-2.5-%sb-deepseek-r1.yaml", paramB))
		}
	} else {
		switch paramB {
		case "32", "72":
			return filepath.Join(basePath, "gpu", fmt.Sprintf("grpo-qwen-2.5-%sb-bnb-4bit-deepseek-r1.yaml", paramB))
		case "0.5", "1.5", "7":
			return filepath.Join(basePath, "gpu", fmt.Sprintf("grpo-qwen-2.5-%sb-deepseek-r1.yaml", paramB))
		}
	}

	return filepath.Join(basePath, "gpu", "grpo-qwen-2.5-0.5b-deepseek-r1.yaml") // fallback
}

func promptUser(prompt string, defaultValue string, validOptions []string) string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\033[32m%s [%s]: \033[0m", prompt, defaultValue)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return defaultValue
		}

		// Check if input is valid
		for _, option := range validOptions {
			if strings.EqualFold(input, option) {
				return input
			}
		}

		fmt.Printf("Please enter one of: %s\n", strings.Join(validOptions, ", "))
	}
}

func promptYesNo(prompt string, defaultValue string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\033[32m%s [Y/n]: \033[0m", prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "" {
			input = strings.ToLower(defaultValue)
		}

		if input == "y" || input == "yes" {
			return true
		}
		if input == "n" || input == "no" {
			return false
		}

		fmt.Println("Please answer yes or no.")
	}
}

func promptChoice(prompt string, options map[string]string, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\033[32m%s [%s]: \033[0m", prompt, defaultValue)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToUpper(input))

		if input == "" {
			input = strings.ToUpper(defaultValue)
		}

		if option, exists := options[input]; exists {
			return option
		}

		fmt.Printf("Please enter one of: %s\n", strings.Join(getKeys(options), ", "))
	}
}

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getConfiguration() Configuration {
	fmt.Println("\n=== RL Swarm Configuration ===")

	config := Configuration{
		PeerMaddr:    DefaultPeerMaddr,
		HostMaddr:    DefaultHostMaddr,
		IdentityPath: "swarm.pem",
	}

	// Testnet connection
	config.ConnectToTestnet = promptYesNo("Would you like to connect to the Testnet?", "Y")

	// Swarm selection
	swarmOptions := map[string]string{
		"A": "Math (small swarm)",
		"B": "Math Hard (big swarm)",
	}
	swarmChoice := promptChoice("Which swarm would you like to join (Math (A) or Math Hard (B))?", swarmOptions, "A")
	config.UseBigSwarm = (swarmChoice == "Math Hard (big swarm)")

	// Parameter size
	paramOptions := []string{"0.5", "1.5", "7", "32", "72"}
	config.ParamB = promptUser("How many parameters (in billions)? [0.5, 1.5, 7, 32, 72]", "0.5", paramOptions)

	// Set contract address
	if config.UseBigSwarm {
		config.ContractAddress = BigSwarmContract
	} else {
		config.ContractAddress = SmallSwarmContract
	}

	// Set game type
	if config.UseBigSwarm {
		config.Game = "dapo"
	} else {
		config.Game = "gsm8k"
	}

	// Set config path
	config.ConfigPath = getConfigPath(config.ParamB, config.UseBigSwarm)

	// CPU only mode
	config.CPUOnly = isCPUOnly()

	fmt.Println("=== Configuration Complete ===\n")
	return config
}

func promptHFToken() string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\033[32mWould you like to push models you train in the RL swarm to the Hugging Face Hub? [y/N]: \033[0m")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		input = "n"
	}

	if input == "y" || input == "yes" {
		fmt.Print("Enter your HuggingFace access token: ")
		token, _ := reader.ReadString('\n')
		return strings.TrimSpace(token)
	}

	return "None"
}

func runPythonTraining(config Configuration, venvPath string, logger *log.Logger) error {
	venvPython := filepath.Join(venvPath, "bin", "python")
	if runtime.GOOS == "windows" {
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

	// Capture stdout and stderr to detect identity conflicts
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	cmd.Stdin = os.Stdin

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start training process: %v", err)
	}

	// Monitor output for identity conflicts
	identityConflictDetected := false
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line) // Still print to console

			// Check for specific identity conflict pattern
			if strings.Contains(strings.ToLower(line), "identity") &&
				strings.Contains(strings.ToLower(line), "is already taken by another user") {
				identityConflictDetected = true
				logger.Printf("Identity conflict detected in output: %s", line)
				break
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(os.Stderr, "%s\n", line) // Still print to stderr

			// Check for specific identity conflict pattern in stderr too
			if strings.Contains(strings.ToLower(line), "identity") &&
				strings.Contains(strings.ToLower(line), "is already taken by another user") {
				identityConflictDetected = true
				logger.Printf("Identity conflict detected in stderr: %s", line)
				break
			}
		}
	}()

	err = cmd.Wait()

	if identityConflictDetected {
		return fmt.Errorf("identity conflict detected - need cleanup and retry")
	}

	return err
}

func cleanupStaleProcesses(logger *log.Logger) error {
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
		exec.Command("pkill", "-9", "-f", "gensyn").Run()
		exec.Command("pkill", "-9", "-f", "hivemind").Run()
		time.Sleep(1 * time.Second)
	}

	return nil
}

func main() {
	// Parse command line flags
	var hfToken = flag.String("hf_token", "", "HuggingFace token")
	var orgID = flag.String("org_id", "", "Organization ID")
	var identityPath = flag.String("identity_path", "", "Identity PEM path")
	var contractAddress = flag.String("contract_address", "", "Contract address")
	var game = flag.String("game", "", "Game type")
	var configPath = flag.String("config", "", "Config file path")
	var requirementsFile = flag.String("requirements", "", "Requirements file path")
	var modelSize = flag.String("model-size", "0.5", "Model size in billions")
	var bigSwarm = flag.Bool("big-swarm", false, "Use big swarm (Math Hard)")
	var cpuOnly = flag.Bool("cpu-only", true, "Force CPU-only mode")
	var showVersion = flag.Bool("version", false, "Show version information")

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("GSwarm version %s\n", Version)
		fmt.Printf("Build date: %s\n", BuildDate)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	fmt.Println("Starting RL Swarm Supervisor...")

	// Check Python version
	fmt.Println("Checking Python version...")
	if err := checkPythonVersion(); err != nil {
		fmt.Printf("Python version check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Python version OK")

	// Check for Yarn and install if missing
	fmt.Println("Checking for Yarn...")
	if err := checkYarn(); err != nil {
		fmt.Println("Yarn not found. Attempting to install with 'npm install -g yarn'...")
		cmd := exec.Command("npm", "install", "-g", "yarn")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to install Yarn: %v\n", err)
			os.Exit(1)
		}
		// Double-check Yarn is now available
		if err := checkYarn(); err != nil {
			fmt.Printf("Yarn installation verification failed: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println("Yarn is available.")

	// Print banner
	printBanner()

	// Get configuration from user or command line flags
	var config Configuration
	if flag.NFlag() > 0 {
		// Use command line flags
		config = Configuration{
			PeerMaddr:       DefaultPeerMaddr,
			HostMaddr:       DefaultHostMaddr,
			IdentityPath:    *identityPath,
			HFToken:         *hfToken,
			OrgID:           *orgID,
			ContractAddress: *contractAddress,
			Game:            *game,
			ConfigPath:      *configPath,
			UseBigSwarm:     *bigSwarm,
			ParamB:          *modelSize,
			CPUOnly:         *cpuOnly,
		}

		// Set defaults if not provided
		if config.IdentityPath == "" {
			config.IdentityPath = "swarm.pem"
		}
		if config.Game == "" {
			if config.UseBigSwarm {
				config.Game = "dapo"
			} else {
				config.Game = "gsm8k"
			}
		}
		if config.ContractAddress == "" {
			if config.UseBigSwarm {
				config.ContractAddress = BigSwarmContract
			} else {
				config.ContractAddress = SmallSwarmContract
			}
		}
		if config.ConfigPath == "" {
			config.ConfigPath = getConfigPath(config.ParamB, config.UseBigSwarm)
		}
		if config.OrgID != "" {
			config.ConnectToTestnet = true
		}
	} else {
		// Use interactive mode
		config = getConfiguration()
	}

	// Handle modal login if connecting to testnet
	if config.ConnectToTestnet && config.OrgID == "" {
		orgID, err := setupModalLogin(config)
		if err != nil {
			fmt.Printf("Modal login failed: %v\n", err)
			os.Exit(1)
		}
		config.OrgID = orgID
	}

	// Setup logging
	os.MkdirAll("logs", 0755)
	logFile, err := os.OpenFile("logs/gensyn_rl_swarm_go.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("open log: %v", err)
	}
	defer logFile.Close()
	logger := log.New(logFile, "", log.LstdFlags|log.Lmicroseconds)

	// Install requirements
	fmt.Println("Getting requirements...")
	venvPath := "venv"
	if err := installRequirements(venvPath, *requirementsFile, logger); err != nil {
		logger.Fatalf("Failed to install requirements: %v", err)
	}
	fmt.Println("Done!")

	// Prompt for HuggingFace token only if not provided via command line
	if config.HFToken == "" {
		config.HFToken = promptHFToken()
	}

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
					if cleanupErr := cleanupStaleProcesses(logger); cleanupErr != nil {
						logger.Printf("Failed to cleanup stale processes: %v", cleanupErr)
						fmt.Printf("Warning: Failed to cleanup stale processes: %v\n", cleanupErr)
					}

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
