package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/pkg/fogit"
)

func TestConfigCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		setFlags   func() // Function to set flags before execution
		wantErr    bool
		checkValue func(*testing.T, *fogit.Config)
	}{
		{
			name: "list all configuration",
			setFlags: func() {
				configList = true
			},
			wantErr: false,
		},
		{
			name:     "list with no args",
			args:     []string{},
			wantErr:  false,
			setFlags: func() {}, // No flags
		},
		{
			name:     "set workflow.mode",
			args:     []string{"workflow.mode", "trunk-based"},
			wantErr:  false,
			setFlags: func() {},
			checkValue: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.Mode != "trunk-based" {
					t.Errorf("workflow.mode = %s, want trunk-based", cfg.Workflow.Mode)
				}
			},
		},
		{
			name:     "set with set subcommand",
			args:     []string{"set", "workflow.allow_shared_branches", "false"},
			wantErr:  false,
			setFlags: func() {},
			checkValue: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.AllowSharedBranches != false {
					t.Errorf("allow_shared_branches = %v, want false", cfg.Workflow.AllowSharedBranches)
				}
			},
		},
		{
			name:     "set default_priority",
			args:     []string{"default_priority", "high"},
			wantErr:  false,
			setFlags: func() {},
			checkValue: func(t *testing.T, cfg *fogit.Config) {
				if cfg.DefaultPriority != "high" {
					t.Errorf("default_priority = %s, want high", cfg.DefaultPriority)
				}
			},
		},
		{
			name:     "invalid workflow.mode",
			args:     []string{"workflow.mode", "invalid"},
			wantErr:  true,
			setFlags: func() {},
		},
		{
			name:     "invalid priority",
			args:     []string{"default_priority", "super-high"},
			wantErr:  true,
			setFlags: func() {},
		},
		{
			name:     "unknown key",
			args:     []string{"unknown.key", "value"},
			wantErr:  true,
			setFlags: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			configList = false
			configUnset = ""

			// Create temp directory
			tmpDir := t.TempDir()

			// Initialize .fogit directory
			fogitDir := filepath.Join(tmpDir, ".fogit")
			if err := os.MkdirAll(fogitDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Save default config
			defaultCfg := fogit.DefaultConfig()
			if err := config.Save(fogitDir, defaultCfg); err != nil {
				t.Fatal(err)
			}

			// Set flags
			if tt.setFlags != nil {
				tt.setFlags()
			}

			// Reset flags and run with -C flag
			ResetFlags()
			args := append([]string{"-C", tmpDir, "config"}, tt.args...)
			rootCmd.SetArgs(args)
			cmdErr := ExecuteRootCmd()

			// Reset flags after test
			configList = false
			configUnset = ""

			// Check error
			if (cmdErr != nil) != tt.wantErr {
				t.Errorf("runConfig() error = %v, wantErr %v", cmdErr, tt.wantErr)
				return
			}

			// Check value if specified
			if !tt.wantErr && tt.checkValue != nil {
				cfg, err := config.Load(fogitDir)
				if err != nil {
					t.Fatal(err)
				}
				tt.checkValue(t, cfg)
			}
		})
	}
}

func TestConfigGet(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "get workflow.mode",
			key:       "workflow.mode",
			wantValue: "branch-per-feature",
			wantErr:   false,
		},
		{
			name:      "get workflow.allow_shared_branches",
			key:       "workflow.allow_shared_branches",
			wantValue: "true",
			wantErr:   false,
		},
		{
			name:      "get workflow.base_branch",
			key:       "workflow.base_branch",
			wantValue: "main",
			wantErr:   false,
		},
		{
			name:      "get workflow.version_format",
			key:       "workflow.version_format",
			wantValue: "simple",
			wantErr:   false,
		},
		{
			name:      "get feature_search.fuzzy_match",
			key:       "feature_search.fuzzy_match",
			wantValue: "true",
			wantErr:   false,
		},
		{
			name:      "get feature_search.min_similarity",
			key:       "feature_search.min_similarity",
			wantValue: "60.00",
			wantErr:   false,
		},
		{
			name:      "get feature_search.max_suggestions",
			key:       "feature_search.max_suggestions",
			wantValue: "5",
			wantErr:   false,
		},
		{
			name:    "get unset default_priority",
			key:     "default_priority",
			wantErr: true, // Not set by default
		},
		{
			name:    "get unknown key",
			key:     "unknown.key",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := fogit.DefaultConfig()

			value, err := getConfigValue(cfg, tt.key)

			if (err != nil) != tt.wantErr {
				t.Errorf("getConfigValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && value != tt.wantValue {
				t.Errorf("getConfigValue() = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

func TestConfigSet(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
		check   func(*testing.T, *fogit.Config)
	}{
		{
			name:    "set workflow.mode to trunk-based",
			key:     "workflow.mode",
			value:   "trunk-based",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.Mode != "trunk-based" {
					t.Errorf("Mode = %s, want trunk-based", cfg.Workflow.Mode)
				}
			},
		},
		{
			name:    "set workflow.mode to branch-per-feature",
			key:     "workflow.mode",
			value:   "branch-per-feature",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.Mode != "branch-per-feature" {
					t.Errorf("Mode = %s, want branch-per-feature", cfg.Workflow.Mode)
				}
			},
		},
		{
			name:    "set workflow.mode to invalid value",
			key:     "workflow.mode",
			value:   "invalid",
			wantErr: true,
		},
		{
			name:    "set workflow.allow_shared_branches true",
			key:     "workflow.allow_shared_branches",
			value:   "true",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if !cfg.Workflow.AllowSharedBranches {
					t.Error("AllowSharedBranches should be true")
				}
			},
		},
		{
			name:    "set workflow.allow_shared_branches false",
			key:     "workflow.allow_shared_branches",
			value:   "false",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.AllowSharedBranches {
					t.Error("AllowSharedBranches should be false")
				}
			},
		},
		{
			name:    "set workflow.base_branch",
			key:     "workflow.base_branch",
			value:   "develop",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.BaseBranch != "develop" {
					t.Errorf("BaseBranch = %s, want develop", cfg.Workflow.BaseBranch)
				}
			},
		},
		{
			name:    "set workflow.version_format to semantic",
			key:     "workflow.version_format",
			value:   "semantic",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.VersionFormat != "semantic" {
					t.Errorf("VersionFormat = %s, want semantic", cfg.Workflow.VersionFormat)
				}
			},
		},
		{
			name:    "set workflow.version_format to invalid value",
			key:     "workflow.version_format",
			value:   "custom",
			wantErr: true,
		},
		{
			name:    "set feature_search.fuzzy_match",
			key:     "feature_search.fuzzy_match",
			value:   "false",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.FeatureSearch.FuzzyMatch {
					t.Error("FuzzyMatch should be false")
				}
			},
		},
		{
			name:    "set feature_search.min_similarity",
			key:     "feature_search.min_similarity",
			value:   "0.8",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.FeatureSearch.MinSimilarity != 0.8 {
					t.Errorf("MinSimilarity = %f, want 0.8", cfg.FeatureSearch.MinSimilarity)
				}
			},
		},
		{
			name:    "set feature_search.min_similarity invalid",
			key:     "feature_search.min_similarity",
			value:   "not-a-number",
			wantErr: true,
		},
		{
			name:    "set feature_search.min_similarity out of range",
			key:     "feature_search.min_similarity",
			value:   "1.5",
			wantErr: true,
		},
		{
			name:    "set feature_search.max_suggestions",
			key:     "feature_search.max_suggestions",
			value:   "10",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.FeatureSearch.MaxSuggestions != 10 {
					t.Errorf("MaxSuggestions = %d, want 10", cfg.FeatureSearch.MaxSuggestions)
				}
			},
		},
		{
			name:    "set feature_search.max_suggestions invalid",
			key:     "feature_search.max_suggestions",
			value:   "not-a-number",
			wantErr: true,
		},
		{
			name:    "set default_priority low",
			key:     "default_priority",
			value:   "low",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.DefaultPriority != "low" {
					t.Errorf("DefaultPriority = %s, want low", cfg.DefaultPriority)
				}
			},
		},
		{
			name:    "set default_priority medium",
			key:     "default_priority",
			value:   "medium",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.DefaultPriority != "medium" {
					t.Errorf("DefaultPriority = %s, want medium", cfg.DefaultPriority)
				}
			},
		},
		{
			name:    "set default_priority high",
			key:     "default_priority",
			value:   "high",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.DefaultPriority != "high" {
					t.Errorf("DefaultPriority = %s, want high", cfg.DefaultPriority)
				}
			},
		},
		{
			name:    "set default_priority critical",
			key:     "default_priority",
			value:   "critical",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.DefaultPriority != "critical" {
					t.Errorf("DefaultPriority = %s, want critical", cfg.DefaultPriority)
				}
			},
		},
		{
			name:    "set default_priority invalid",
			key:     "default_priority",
			value:   "super-high",
			wantErr: true,
		},
		{
			name:    "set unknown key",
			key:     "unknown.key",
			value:   "value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := fogit.DefaultConfig()

			err := setConfigValue(cfg, tt.key, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("setConfigValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestConfigUnset(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
		check   func(*testing.T, *fogit.Config)
	}{
		{
			name:    "unset workflow.mode",
			key:     "workflow.mode",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.Mode != "branch-per-feature" {
					t.Errorf("Mode = %s, want default branch-per-feature", cfg.Workflow.Mode)
				}
			},
		},
		{
			name:    "unset workflow.allow_shared_branches",
			key:     "workflow.allow_shared_branches",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.AllowSharedBranches != false {
					t.Error("AllowSharedBranches should reset to false")
				}
			},
		},
		{
			name:    "unset workflow.base_branch",
			key:     "workflow.base_branch",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.BaseBranch != "" {
					t.Errorf("BaseBranch = %s, want empty", cfg.Workflow.BaseBranch)
				}
			},
		},
		{
			name:    "unset default_priority",
			key:     "default_priority",
			wantErr: false,
			check: func(t *testing.T, cfg *fogit.Config) {
				if cfg.DefaultPriority != "" {
					t.Errorf("DefaultPriority = %s, want empty", cfg.DefaultPriority)
				}
			},
		},
		{
			name:    "unset unknown key",
			key:     "unknown.key",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := fogit.DefaultConfig()

			err := unsetConfigValue(cfg, tt.key)

			if (err != nil) != tt.wantErr {
				t.Errorf("unsetConfigValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestConfigUnsetFlag(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Initialize .fogit directory
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Save config with default_priority set
	cfg := fogit.DefaultConfig()
	cfg.DefaultPriority = "high"
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "config", "--unset", "default_priority"})
	cmdErr := ExecuteRootCmd()

	if cmdErr != nil {
		t.Fatalf("Failed to unset: %v", cmdErr)
	}

	// Verify it was unset
	reloaded, err := config.Load(fogitDir)
	if err != nil {
		t.Fatal(err)
	}

	if reloaded.DefaultPriority != "" {
		t.Errorf("DefaultPriority = %s, want empty after unset", reloaded.DefaultPriority)
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"true", "true", true},
		{"True", "True", true},
		{"TRUE", "TRUE", true},
		{"1", "1", true},
		{"yes", "yes", true},
		{"YES", "YES", true},
		{"on", "on", true},
		{"ON", "ON", true},
		{"false", "false", false},
		{"False", "False", false},
		{"0", "0", false},
		{"no", "no", false},
		{"off", "off", false},
		{"anything", "anything", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBool(tt.value)
			if got != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestConfigNotInRepository(t *testing.T) {
	// Create temp directory WITHOUT .fogit
	tmpDir := t.TempDir()

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "config"})
	cmdErr := ExecuteRootCmd()

	if cmdErr == nil {
		t.Error("Expected error when not in a FoGit repository")
		return
	}

	if !strings.Contains(cmdErr.Error(), "not in a FoGit repository") {
		t.Errorf("Expected 'not in a FoGit repository' error, got: %v", cmdErr)
	}
}
