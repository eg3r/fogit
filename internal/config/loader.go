package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/eg3r/fogit/pkg/fogit"
)

// Load reads the config from .fogit/config.yml, returning defaults if file doesn't exist
func Load(fogitDir string) (*fogit.Config, error) {
	configPath := filepath.Join(fogitDir, "config.yml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return fogit.DefaultConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config fogit.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Merge with defaults for any missing fields
	mergeWithDefaults(&config)

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Save writes the config to .fogit/config.yml
func Save(fogitDir string, config *fogit.Config) error {
	configPath := filepath.Join(fogitDir, "config.yml")

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// mergeWithDefaults fills in any missing fields with defaults
func mergeWithDefaults(config *fogit.Config) {
	defaults := fogit.DefaultConfig()

	// Repository defaults
	if config.Repository.Version == "" {
		config.Repository.Version = defaults.Repository.Version
	}
	if config.Repository.InitializedAt.IsZero() {
		config.Repository.InitializedAt = defaults.Repository.InitializedAt
	}

	// UI defaults
	if config.UI.DefaultGroupBy == "" {
		config.UI.DefaultGroupBy = defaults.UI.DefaultGroupBy
	}
	if config.UI.DefaultLayout == "" {
		config.UI.DefaultLayout = defaults.UI.DefaultLayout
	}

	// Workflow defaults
	if config.Workflow.Mode == "" {
		config.Workflow.Mode = defaults.Workflow.Mode
	}
	if config.Workflow.BaseBranch == "" {
		config.Workflow.BaseBranch = defaults.Workflow.BaseBranch
	}
	if config.Workflow.VersionFormat == "" {
		config.Workflow.VersionFormat = defaults.Workflow.VersionFormat
	}
	// Note: AllowSharedBranches is a boolean, so we can't check if it's "empty"
	// We'll assume false if not set (which is the Go zero value)

	// Commit template default
	if config.CommitTemplate == "" {
		config.CommitTemplate = defaults.CommitTemplate
	}

	// Relationship defaults - types and categories must be merged together
	// since default types reference default categories.
	// Use nil check (not len==0) to respect explicit empty configs
	if config.Relationships.Types == nil {
		config.Relationships.Types = defaults.Relationships.Types
	}
	if config.Relationships.Categories == nil {
		config.Relationships.Categories = defaults.Relationships.Categories
	}
	// Merge relationship defaults if not set, but only if the referenced
	// types/categories exist (to avoid validation errors when user explicitly
	// defines empty types/categories)
	if config.Relationships.Defaults.Category == "" {
		if _, exists := config.Relationships.Categories[defaults.Relationships.Defaults.Category]; exists {
			config.Relationships.Defaults.Category = defaults.Relationships.Defaults.Category
		}
	}
	if config.Relationships.Defaults.TreeType == "" {
		if _, exists := config.Relationships.Types[defaults.Relationships.Defaults.TreeType]; exists {
			config.Relationships.Defaults.TreeType = defaults.Relationships.Defaults.TreeType
		}
	}

	// FeatureSearch defaults
	if config.FeatureSearch.MinSimilarity == 0 {
		config.FeatureSearch = defaults.FeatureSearch
	}
}
