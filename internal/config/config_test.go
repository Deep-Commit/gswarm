package config

import (
	"flag"
	"testing"
)

func TestValidateConfiguration(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Configuration
		wantErr bool
	}{
		{"valid gsm8k small", Configuration{ParamB: "0.5", Game: "gsm8k"}, false},
		{"valid dapo big", Configuration{ParamB: "32", Game: "dapo"}, false},
		{"valid 7B gsm8k", Configuration{ParamB: "7", Game: "gsm8k"}, false},
		{"valid 1.5B dapo", Configuration{ParamB: "1.5", Game: "dapo"}, false},
		{"valid 72B gsm8k", Configuration{ParamB: "72", Game: "gsm8k"}, false},
		{"bad param", Configuration{ParamB: "3", Game: "gsm8k"}, true},
		{"bad game", Configuration{ParamB: "7", Game: "foo"}, true},
		{"empty param", Configuration{ParamB: "", Game: "gsm8k"}, true},
		{"empty game", Configuration{ParamB: "0.5", Game: ""}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateConfiguration(c.cfg)
			if (err != nil) != c.wantErr {
				t.Errorf("ValidateConfiguration() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	cases := []struct {
		name        string
		paramB      string
		useBigSwarm bool
		want        string
	}{
		{"small swarm 0.5B", "0.5", false, "hivemind_exp/configs/gpu/grpo-qwen-2.5-0.5b-deepseek-r1.yaml"},
		{"big swarm 32B", "32", true, "hivemind_exp/configs/gpu/grpo-qwen-2.5-32b-bnb-4bit-deepseek-r1.yaml"},
		{"small swarm 7B", "7", false, "hivemind_exp/configs/gpu/grpo-qwen-2.5-7b-deepseek-r1.yaml"},
		{"big swarm 72B", "72", true, "hivemind_exp/configs/gpu/grpo-qwen-2.5-72b-bnb-4bit-deepseek-r1.yaml"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := GetConfigPath(c.paramB, c.useBigSwarm)
			if got != c.want {
				t.Errorf("GetConfigPath() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestGetConfiguration_FlagOverrides(t *testing.T) {
	// Temporarily override testable prompt functions
	origTestPromptHFToken := testPromptHFToken
	origTestPromptYesNo := testPromptYesNo
	origTestPromptChoice := testPromptChoice
	origTestPromptUser := testPromptUser

	defer func() {
		testPromptHFToken = origTestPromptHFToken
		testPromptYesNo = origTestPromptYesNo
		testPromptChoice = origTestPromptChoice
		testPromptUser = origTestPromptUser
	}()

	// Test cases
	cases := []struct {
		name    string
		visited map[string]bool
		want    Configuration
	}{
		{
			name:    "no flags set - uses defaults",
			visited: map[string]bool{},
			want: Configuration{
				ConnectToTestnet: false, UseBigSwarm: false, ParamB: "0.5",
				HFToken: "None", OrgID: "", IdentityPath: "swarm.pem",
				ContractAddress: SmallSwarmContract, Game: "gsm8k", ConfigPath: "hivemind_exp/configs/gpu/grpo-qwen-2.5-0.5b-deepseek-r1.yaml",
				CPUOnly: false, RequirementsFile: "",
				PublicMaddr: DefaultPublicMaddr, PeerMaddr: DefaultPeerMaddr, HostMaddr: DefaultHostMaddr,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Reset flags for each test case
			flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

			// Re-declare package-level flags for this test case
			TestnetFlag = flag.Bool("testnet", false, "Connect to the Testnet?")
			BigSwarmFlag = flag.Bool("big-swarm", false, "Use big swarm (Math Hard)?")
			ModelSizeFlag = flag.String("model-size", "0.5", "Parameter count (0.5, 1.5, 7, 32, 72)")
			HFTokenFlag = flag.String("hf-token", "", "HuggingFace access token")
			OrgIDFlag = flag.String("org-id", "", "Modal ORG_ID (for testnet)")
			IdentityPathFlag = flag.String("identity-path", "swarm.pem", "Path to identity PEM")
			ContractAddrFlag = flag.String("contract-address", "", "Override smart‐contract address")
			GameFlag = flag.String("game", "", "Game type ('gsm8k' or 'dapo')")
			ConfigPathFlag = flag.String("config-path", "", "Path to YAML config file")
			CPUOnlyFlag = flag.Bool("cpu-only", false, "Force CPU-only mode")
			RequirementsFlag = flag.String("requirements", "", "Requirements file (overrides default)")

			// Override testable prompt functions to return predictable values
			testPromptHFToken = func() string { return "None" }
			testPromptYesNo = func(_ string, _ string) bool { return false }
			testPromptChoice = func(_ string, _ map[string]string, _ string) string {
				return "Math (small swarm)"
			}
			testPromptUser = func(_ string, defaultValue string, _ []string) string { return defaultValue }

			got := GetConfiguration(c.visited)

			// Compare key fields
			if got.ConnectToTestnet != c.want.ConnectToTestnet {
				t.Errorf("ConnectToTestnet = %v, want %v", got.ConnectToTestnet, c.want.ConnectToTestnet)
			}
			if got.UseBigSwarm != c.want.UseBigSwarm {
				t.Errorf("UseBigSwarm = %v, want %v", got.UseBigSwarm, c.want.UseBigSwarm)
			}
			if got.ParamB != c.want.ParamB {
				t.Errorf("ParamB = %v, want %v", got.ParamB, c.want.ParamB)
			}
			if got.Game != c.want.Game {
				t.Errorf("Game = %v, want %v", got.Game, c.want.Game)
			}
			if got.ContractAddress != c.want.ContractAddress {
				t.Errorf("ContractAddress = %v, want %v", got.ContractAddress, c.want.ContractAddress)
			}
			if got.ConfigPath != c.want.ConfigPath {
				t.Errorf("ConfigPath = %v, want %v", got.ConfigPath, c.want.ConfigPath)
			}
		})
	}
}

func TestGetConfiguration_OrgIDOverridesTestnet(t *testing.T) {
	// Reset flags
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	OrgIDFlag = flag.String("org-id", "", "Modal ORG_ID (for testnet)")

	// Override testable prompt functions
	origTestPromptHFToken := testPromptHFToken
	defer func() { testPromptHFToken = origTestPromptHFToken }()
	testPromptHFToken = func() string { return "None" }

	// Test that setting org-id forces testnet to true
	*OrgIDFlag = "test-org"
	visited := map[string]bool{"org-id": true}

	cfg := GetConfiguration(visited)
	if !cfg.ConnectToTestnet {
		t.Error("Setting org-id should force ConnectToTestnet to true")
	}
	if cfg.OrgID != "test-org" {
		t.Errorf("OrgID = %v, want test-org", cfg.OrgID)
	}
}
