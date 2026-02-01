package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	configGlobal bool
	configList   bool
	configUnset  string
)

var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "View or modify configuration",
	Long: `View or modify FoGit configuration.

Configuration is stored in .fogit/config.yml.

Supported configuration keys:
  workflow.mode                     - Workflow mode ('branch-per-feature' or 'trunk-based')
  workflow.allow_shared_branches    - Allow --same flag (true/false)
  workflow.base_branch              - Base branch for features
  workflow.create_branch_from       - Where to create branches from ('trunk', 'warn', 'current')
  workflow.version_format           - Version format ('simple' or 'semantic')
  feature_search.fuzzy_match        - Enable fuzzy matching (true/false)
  feature_search.min_similarity     - Minimum similarity threshold (0.0-1.0)
  feature_search.max_suggestions    - Maximum suggestions to show
  default_priority                  - Default feature priority

Examples:
  fogit config --list
  fogit config workflow.mode trunk-based
  fogit config set workflow.allow_shared_branches false
  fogit config --unset default_priority`,
	RunE: runConfig,
}

func init() {
	configCmd.Flags().BoolVar(&configGlobal, "global", false, "Set global configuration")
	configCmd.Flags().BoolVar(&configList, "list", false, "List all configuration")
	configCmd.Flags().StringVar(&configUnset, "unset", "", "Remove configuration key")

	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	// Find .fogit directory
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	fogitDir := filepath.Join(cwd, ".fogit")

	// Check if .fogit exists
	if _, statErr := os.Stat(fogitDir); os.IsNotExist(statErr) {
		return fmt.Errorf("not in a FoGit repository (no .fogit directory found)")
	}

	// Load existing config
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Handle --list
	if configList {
		return listConfig(cfg)
	}

	// Handle --unset
	if configUnset != "" {
		return unsetConfig(cfg, fogitDir, configUnset)
	}

	// Handle get/set
	if len(args) == 0 {
		// No args = list all
		return listConfig(cfg)
	}

	// Check for 'set' subcommand
	if args[0] == "set" {
		if len(args) < 3 {
			return fmt.Errorf("usage: fogit config set <key> <value>")
		}
		return setConfig(cfg, fogitDir, args[1], args[2])
	}

	// Direct key [value] syntax
	if len(args) == 1 {
		// Get value
		return getConfig(cfg, args[0])
	}

	if len(args) == 2 {
		// Set value
		return setConfig(cfg, fogitDir, args[0], args[1])
	}

	return fmt.Errorf("usage: fogit config [key] [value] or fogit config set <key> <value>")
}

func listConfig(cfg *fogit.Config) error {
	fmt.Println("Configuration:")
	fmt.Println()

	// Workflow settings
	fmt.Println("[workflow]")
	fmt.Printf("  mode = %s\n", cfg.Workflow.Mode)
	fmt.Printf("  allow_shared_branches = %v\n", cfg.Workflow.AllowSharedBranches)
	if cfg.Workflow.BaseBranch != "" {
		fmt.Printf("  base_branch = %s\n", cfg.Workflow.BaseBranch)
	}
	if cfg.Workflow.CreateBranchFrom != "" {
		fmt.Printf("  create_branch_from = %s\n", cfg.Workflow.CreateBranchFrom)
	}
	fmt.Printf("  version_format = %s\n", cfg.Workflow.VersionFormat)
	fmt.Println()

	// Feature search settings
	fmt.Println("[feature_search]")
	fmt.Printf("  fuzzy_match = %v\n", cfg.FeatureSearch.FuzzyMatch)
	fmt.Printf("  min_similarity = %.2f\n", cfg.FeatureSearch.MinSimilarity)
	fmt.Printf("  max_suggestions = %d\n", cfg.FeatureSearch.MaxSuggestions)
	fmt.Println()

	// Default priority
	if cfg.DefaultPriority != "" {
		fmt.Println("[defaults]")
		fmt.Printf("  default_priority = %s\n", cfg.DefaultPriority)
		fmt.Println()
	}

	return nil
}

func getConfig(cfg *fogit.Config, key string) error {
	value, err := getConfigValue(cfg, key)
	if err != nil {
		return err
	}

	fmt.Println(value)
	return nil
}

func setConfig(cfg *fogit.Config, fogitDir, key, value string) error {
	// Set the value
	if err := setConfigValue(cfg, key, value); err != nil {
		return err
	}

	// Save config
	if err := config.Save(fogitDir, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Set %s = %s\n", key, value)
	return nil
}

func unsetConfig(cfg *fogit.Config, fogitDir, key string) error {
	// Unset the value
	if err := unsetConfigValue(cfg, key); err != nil {
		return err
	}

	// Save config
	if err := config.Save(fogitDir, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Unset %s\n", key)
	return nil
}

func getConfigValue(cfg *fogit.Config, key string) (string, error) {
	switch key {
	case "workflow.mode":
		return cfg.Workflow.Mode, nil
	case "workflow.allow_shared_branches":
		return fmt.Sprintf("%v", cfg.Workflow.AllowSharedBranches), nil
	case "workflow.base_branch":
		return cfg.Workflow.BaseBranch, nil
	case "workflow.create_branch_from":
		return cfg.Workflow.CreateBranchFrom, nil
	case "workflow.version_format":
		return cfg.Workflow.VersionFormat, nil
	case "feature_search.fuzzy_match":
		return fmt.Sprintf("%v", cfg.FeatureSearch.FuzzyMatch), nil
	case "feature_search.min_similarity":
		return fmt.Sprintf("%.2f", cfg.FeatureSearch.MinSimilarity), nil
	case "feature_search.max_suggestions":
		return fmt.Sprintf("%d", cfg.FeatureSearch.MaxSuggestions), nil
	case "default_priority":
		if cfg.DefaultPriority == "" {
			return "", fmt.Errorf("key not set: %s", key)
		}
		return cfg.DefaultPriority, nil
	default:
		return "", fmt.Errorf("unknown configuration key: %s", key)
	}
}

func setConfigValue(cfg *fogit.Config, key, value string) error {
	switch key {
	case "workflow.mode":
		if value != "branch-per-feature" && value != "trunk-based" {
			return fmt.Errorf("invalid workflow.mode: %s (must be 'branch-per-feature' or 'trunk-based')", value)
		}
		cfg.Workflow.Mode = value
	case "workflow.allow_shared_branches":
		cfg.Workflow.AllowSharedBranches = parseBool(value)
	case "workflow.base_branch":
		cfg.Workflow.BaseBranch = value
	case "workflow.create_branch_from":
		if value != "trunk" && value != "warn" && value != "current" {
			return fmt.Errorf("invalid workflow.create_branch_from: %s (must be 'trunk', 'warn', or 'current')", value)
		}
		cfg.Workflow.CreateBranchFrom = value
	case "workflow.version_format":
		if value != "simple" && value != "semantic" {
			return fmt.Errorf("invalid workflow.version_format: %s (must be 'simple' or 'semantic')", value)
		}
		cfg.Workflow.VersionFormat = value
	case "feature_search.fuzzy_match":
		cfg.FeatureSearch.FuzzyMatch = parseBool(value)
	case "feature_search.min_similarity":
		var similarity float64
		if _, err := fmt.Sscanf(value, "%f", &similarity); err != nil {
			return fmt.Errorf("invalid min_similarity: %s (must be a number between 0.0 and 1.0)", value)
		}
		if similarity < 0.0 || similarity > 1.0 {
			return fmt.Errorf("min_similarity must be between 0.0 and 1.0, got %f", similarity)
		}
		cfg.FeatureSearch.MinSimilarity = similarity
	case "feature_search.max_suggestions":
		var max int
		if _, err := fmt.Sscanf(value, "%d", &max); err != nil {
			return fmt.Errorf("invalid max_suggestions: %s (must be a positive integer)", value)
		}
		if max < 0 {
			return fmt.Errorf("max_suggestions must be positive, got %d", max)
		}
		cfg.FeatureSearch.MaxSuggestions = max
	case "default_priority":
		// Validate priority
		validPriorities := []string{"low", "medium", "high", "critical"}
		valid := false
		for _, p := range validPriorities {
			if value == p {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid priority: %s (must be one of: low, medium, high, critical)", value)
		}
		cfg.DefaultPriority = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return nil
}

func unsetConfigValue(cfg *fogit.Config, key string) error {
	switch key {
	case "workflow.mode":
		cfg.Workflow.Mode = "branch-per-feature" // Reset to default
	case "workflow.allow_shared_branches":
		cfg.Workflow.AllowSharedBranches = false // Reset to default
	case "workflow.base_branch":
		cfg.Workflow.BaseBranch = ""
	case "workflow.create_branch_from":
		cfg.Workflow.CreateBranchFrom = "trunk" // Reset to default
	case "workflow.version_format":
		cfg.Workflow.VersionFormat = "simple" // Reset to default
	case "feature_search.fuzzy_match":
		cfg.FeatureSearch.FuzzyMatch = true // Reset to default
	case "feature_search.min_similarity":
		cfg.FeatureSearch.MinSimilarity = 0.6 // Reset to default
	case "feature_search.max_suggestions":
		cfg.FeatureSearch.MaxSuggestions = 5 // Reset to default
	case "default_priority":
		cfg.DefaultPriority = ""
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return nil
}

func parseBool(value string) bool {
	value = strings.ToLower(value)
	return value == "true" || value == "1" || value == "yes" || value == "on"
}
