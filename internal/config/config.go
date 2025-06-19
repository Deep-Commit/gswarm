// Package config provides configuration management utilities for GSwarm,
// including loading and validation of application settings.
package config

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Deep-Commit/gswarm/internal/prompt"
)

// Constants
const (
	DefaultPublicMaddr = "/ip4/38.101.215.13/tcp/30002/p2p/QmQ2gEXoPJg6iMBSUFWGzAabS2VhnzuS782Y637hGjfsRJ"
	DefaultPeerMaddr   = "/ip4/38.101.215.13/tcp/30002/p2p/QmQ2gEXoPJg6iMBSUFWGzAabS2VhnzuS782Y637hGjfsRJ"
	DefaultHostMaddr   = "/ip4/0.0.0.0/tcp/38331"
	SmallSwarmContract = "0x69C6e1D608ec64885E7b185d39b04B491a71768C"
	BigSwarmContract   = "0x6947c6E196a48B77eFa9331EC1E3e45f3Ee5Fd58"
)

// Flag definitions - using consistent hyphen naming
var (
	TestnetFlag      = flag.Bool("testnet", false, "Connect to the Testnet?")
	BigSwarmFlag     = flag.Bool("big-swarm", false, "Use big swarm (Math Hard)?")
	ModelSizeFlag    = flag.String("model-size", "0.5", "Parameter count (0.5, 1.5, 7, 32, 72)")
	HFTokenFlag      = flag.String("hf-token", "", "HuggingFace access token")
	OrgIDFlag        = flag.String("org-id", "", "Modal ORG_ID (for testnet)")
	IdentityPathFlag = flag.String("identity-path", "swarm.pem", "Path to identity PEM")
	ContractAddrFlag = flag.String("contract-address", "", "Override smart‐contract address")
	GameFlag         = flag.String("game", "", "Game type ('gsm8k' or 'dapo')")
	ConfigPathFlag   = flag.String("config-path", "", "Path to YAML config file")
	CPUOnlyFlag      = flag.Bool("cpu-only", false, "Force CPU-only mode")
	RequirementsFlag = flag.String("requirements", "", "Requirements file (overrides default)")
	ShowVersionFlag  = flag.Bool("version", false, "Show version information")
)

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

// Testable prompt functions that can be overridden in tests
var (
	testPromptHFToken = prompt.HFToken
	testPromptYesNo   = prompt.YesNo
	testPromptChoice  = prompt.Choice
	testPromptUser    = prompt.User
)

// GetConfigPath returns the appropriate config path based on parameters
func GetConfigPath(paramB string, useBigSwarm bool) string {
	// Use the same logic as the original run_rl_swarm.sh script
	// Note: We can't check isCPUOnly() here since it's not available in this package
	// The CPU-only check should be done in the main package before calling this function

	// For now, assume GPU mode and use the gpu configs
	switch paramB {
	case "32", "72":
		return fmt.Sprintf("hivemind_exp/configs/gpu/grpo-qwen-2.5-%sb-bnb-4bit-deepseek-r1.yaml", paramB)
	case "0.5", "1.5", "7":
		return fmt.Sprintf("hivemind_exp/configs/gpu/grpo-qwen-2.5-%sb-deepseek-r1.yaml", paramB)
	default:
		// Fallback to 0.5B config
		return "hivemind_exp/configs/gpu/grpo-qwen-2.5-0.5b-deepseek-r1.yaml"
	}
}

// GetConfiguration builds a Configuration, prompting only for flags not in `visited`
func GetConfiguration(visited map[string]bool) Configuration {
	cfg := Configuration{
		PublicMaddr: DefaultPublicMaddr,
		PeerMaddr:   DefaultPeerMaddr,
		HostMaddr:   DefaultHostMaddr,
	}

	// -- Testnet?
	if visited["testnet"] {
		cfg.ConnectToTestnet = *TestnetFlag
	} else {
		cfg.ConnectToTestnet = testPromptYesNo("Would you like to connect to the Testnet?", "Y")
	}

	// -- Big swarm?
	if visited["big-swarm"] {
		cfg.UseBigSwarm = *BigSwarmFlag
	} else {
		choice := testPromptChoice(
			"Which swarm would you like to join (Math (A) or Math Hard (B))?",
			map[string]string{"A": "Math (small swarm)", "B": "Math Hard (big swarm)"},
			"A",
		)
		cfg.UseBigSwarm = (choice == "Math Hard (big swarm)")
	}

	// -- Model size?
	if visited["model-size"] {
		cfg.ParamB = *ModelSizeFlag
	} else {
		cfg.ParamB = testPromptUser(
			"How many parameters (in billions)? [0.5,1.5,7,32,72]",
			"0.5",
			[]string{"0.5", "1.5", "7", "32", "72"},
		)
	}

	// -- Identity path
	if visited["identity-path"] {
		cfg.IdentityPath = *IdentityPathFlag
	} else {
		cfg.IdentityPath = "swarm.pem"
	}

	// -- CPU-only?
	if visited["cpu-only"] {
		cfg.CPUOnly = *CPUOnlyFlag
	} else {
		cfg.CPUOnly = false // This will be determined by isCPUOnly() in main
	}

	// -- Contract address
	if visited["contract-address"] {
		cfg.ContractAddress = *ContractAddrFlag
	} else {
		if cfg.UseBigSwarm {
			cfg.ContractAddress = BigSwarmContract
		} else {
			cfg.ContractAddress = SmallSwarmContract
		}
	}

	// -- Game type
	if visited["game"] {
		cfg.Game = *GameFlag
	} else {
		if cfg.UseBigSwarm {
			cfg.Game = "dapo"
		} else {
			cfg.Game = "gsm8k"
		}
	}

	// -- Config path
	if visited["config-path"] {
		cfg.ConfigPath = *ConfigPathFlag
	} else {
		cfg.ConfigPath = GetConfigPath(cfg.ParamB, cfg.UseBigSwarm)
	}

	// -- HuggingFace token
	if visited["hf-token"] {
		cfg.HFToken = *HFTokenFlag
	} else {
		cfg.HFToken = testPromptHFToken()
	}

	// -- Org ID
	if visited["org-id"] {
		cfg.OrgID = *OrgIDFlag
		if cfg.OrgID != "" {
			cfg.ConnectToTestnet = true
		}
	}

	// -- Requirements override
	if visited["requirements"] {
		cfg.RequirementsFile = *RequirementsFlag
	}

	return cfg
}

// ValidateConfiguration validates the configuration
func ValidateConfiguration(config Configuration) error {
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

// GetTestLogger returns a logger for testing purposes
func GetTestLogger() *log.Logger {
	return log.New(os.Stdout, "", log.LstdFlags)
}
