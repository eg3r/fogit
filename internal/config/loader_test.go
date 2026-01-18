package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eg3r/fogit/pkg/fogit"
)

func TestLoad_MissingFile(t *testing.T) {
	// Create temp directory without config file
	tempDir, err := os.MkdirTemp("", "fogit-test-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load returned nil config")
	}

	// Should return defaults
	if !cfg.AutoCommit {
		t.Error("Missing config should return default with auto_commit=true")
	}
	if cfg.Workflow.Mode != "branch-per-feature" {
		t.Errorf("Missing config should return default workflow mode, got %s", cfg.Workflow.Mode)
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fogit-test-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create valid config
	configYAML := `repository:
  name: test-project
  initialized_at: 2025-10-30T10:00:00Z
  version: 1.0.0
ui:
  default_group_by: priority
  default_layout: flat
workflow:
  mode: trunk-based
  base_branch: develop
  allow_shared_branches: false
auto_commit: false
commit_template: "custom: {name}"
auto_push: true
relationships:
  allow_cycles:
    structural: false
    informational: true
  types: {}
`
	configPath := filepath.Join(tempDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	cfg, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify custom values
	if cfg.Repository.Name != "test-project" {
		t.Errorf("Expected repository.name 'test-project', got %s", cfg.Repository.Name)
	}
	if cfg.UI.DefaultGroupBy != "priority" {
		t.Errorf("Expected ui.default_group_by 'priority', got %s", cfg.UI.DefaultGroupBy)
	}
	if cfg.Workflow.Mode != "trunk-based" {
		t.Errorf("Expected workflow.mode 'trunk-based', got %s", cfg.Workflow.Mode)
	}
	if cfg.Workflow.BaseBranch != "develop" {
		t.Errorf("Expected workflow.base_branch 'develop', got %s", cfg.Workflow.BaseBranch)
	}
	if cfg.Workflow.AllowSharedBranches {
		t.Error("Expected workflow.allow_shared_branches false")
	}
	if cfg.AutoCommit {
		t.Error("Expected auto_commit false")
	}
	if cfg.CommitTemplate != "custom: {name}" {
		t.Errorf("Expected commit_template 'custom: {name}', got %s", cfg.CommitTemplate)
	}
	if !cfg.AutoPush {
		t.Error("Expected auto_push true")
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fogit-test-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create partial config (missing several fields)
	configYAML := `auto_commit: false
workflow:
  mode: trunk-based
`
	configPath := filepath.Join(tempDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	cfg, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Failed to load partial config: %v", err)
	}

	// Verify custom values are preserved
	if cfg.AutoCommit {
		t.Error("Expected auto_commit false (from file)")
	}
	if cfg.Workflow.Mode != "trunk-based" {
		t.Errorf("Expected workflow.mode 'trunk-based' (from file), got %s", cfg.Workflow.Mode)
	}

	// Verify defaults are filled in
	if cfg.UI.DefaultGroupBy != "category" {
		t.Errorf("Expected ui.default_group_by filled with default 'category', got %s", cfg.UI.DefaultGroupBy)
	}
	if cfg.Workflow.BaseBranch != "main" {
		t.Errorf("Expected workflow.base_branch filled with default 'main', got %s", cfg.Workflow.BaseBranch)
	}
	if cfg.CommitTemplate != "feat: {title} ({id})" {
		t.Errorf("Expected commit_template filled with default, got %s", cfg.CommitTemplate)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fogit-test-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create invalid YAML
	configYAML := `invalid: [yaml: syntax:
  - broken
    nested: without proper indentation
`
	configPath := filepath.Join(tempDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	_, err = Load(tempDir)
	if err == nil {
		t.Error("Expected error when loading invalid YAML")
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	// Base config with all required categories to avoid merge issues
	baseCategories := `relationships:
  categories:
    structural:
      description: "Dependencies"
      cycle_detection: strict
    informational:
      description: "References"
      allow_cycles: true
      cycle_detection: none
    workflow:
      description: "Process"
      cycle_detection: warn
    compliance:
      description: "Regulatory"
      cycle_detection: strict
`

	tests := []struct {
		name       string
		configYAML string
		wantErrMsg string
	}{
		{
			name: "invalid workflow mode",
			configYAML: baseCategories + `workflow:
  mode: invalid-mode
`,
			wantErrMsg: "workflow.mode",
		},
		{
			name: "invalid version format",
			configYAML: baseCategories + `workflow:
  version_format: invalid-format
`,
			wantErrMsg: "workflow.version_format",
		},
		{
			name: "invalid default priority",
			configYAML: baseCategories + `default_priority: super-high
`,
			wantErrMsg: "default_priority",
		},
		{
			name: "relationship type with unknown category",
			configYAML: `relationships:
  categories:
    structural:
      description: "Deps"
      cycle_detection: strict
    informational:
      description: "Refs"
      allow_cycles: true
      cycle_detection: none
    workflow:
      description: "Process"
      cycle_detection: warn
    compliance:
      description: "Reg"
      cycle_detection: strict
  types:
    depends-on:
      category: structural
    custom-type:
      category: non-existent-category
  defaults:
    relationship_category: structural
    tree_relationship_type: depends-on
`,
			wantErrMsg: "unknown category",
		},
		{
			name: "invalid cycle detection value",
			configYAML: `relationships:
  categories:
    structural:
      description: "Test"
      cycle_detection: invalid-value
    informational:
      description: "Refs"
      allow_cycles: true
      cycle_detection: none
  types:
    depends-on:
      category: structural
  defaults:
    relationship_category: structural
    tree_relationship_type: depends-on
`,
			wantErrMsg: "cycle_detection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "fogit-test-config-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			configPath := filepath.Join(tempDir, "config.yml")
			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			_, err = Load(tempDir)
			if err == nil {
				t.Errorf("Expected validation error for %s", tt.name)
				return
			}

			if tt.wantErrMsg != "" {
				errStr := err.Error()
				if !strings.Contains(errStr, tt.wantErrMsg) {
					t.Errorf("Expected error to contain %q, got: %v", tt.wantErrMsg, err)
				}
			}
		})
	}
}

func TestLoad_VersionFormat(t *testing.T) {
	tests := []struct {
		name           string
		configYAML     string
		expectedFormat string
	}{
		{
			name: "semantic version format",
			configYAML: `workflow:
  version_format: semantic
`,
			expectedFormat: "semantic",
		},
		{
			name: "simple version format",
			configYAML: `workflow:
  version_format: simple
`,
			expectedFormat: "simple",
		},
		{
			name: "default when not specified",
			configYAML: `workflow:
  mode: branch-per-feature
`,
			expectedFormat: "simple", // Default
		},
		{
			name:           "default for empty config",
			configYAML:     ``,
			expectedFormat: "simple", // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "fogit-test-config-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			configPath := filepath.Join(tempDir, "config.yml")
			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			cfg, err := Load(tempDir)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			if cfg.Workflow.VersionFormat != tt.expectedFormat {
				t.Errorf("Expected version_format %q, got %q", tt.expectedFormat, cfg.Workflow.VersionFormat)
			}
		})
	}
}

func TestLoad_VersionFormatPreservedWithOtherSettings(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fogit-test-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Minimal config with just version_format and allow_shared_branches
	// This tests the bug we fixed where isEmpty was true and overwrote everything
	configYAML := `workflow:
  allow_shared_branches: true
  version_format: semantic
`
	configPath := filepath.Join(tempDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	cfg, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Version format should be preserved
	if cfg.Workflow.VersionFormat != "semantic" {
		t.Errorf("Expected version_format 'semantic', got %q", cfg.Workflow.VersionFormat)
	}

	// AllowSharedBranches should be preserved
	if !cfg.Workflow.AllowSharedBranches {
		t.Error("Expected allow_shared_branches true (from file)")
	}

	// But defaults should still be filled for missing fields
	if cfg.Workflow.Mode != "branch-per-feature" {
		t.Errorf("Expected default workflow.mode 'branch-per-feature', got %q", cfg.Workflow.Mode)
	}
	if cfg.Workflow.BaseBranch != "main" {
		t.Errorf("Expected default workflow.base_branch 'main', got %q", cfg.Workflow.BaseBranch)
	}
}

func TestLoad_EmptyConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fogit-test-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create empty config file
	configPath := filepath.Join(tempDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	cfg, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Failed to load empty config: %v", err)
	}

	// Should fill with defaults for string fields
	// Note: Boolean fields like AutoCommit will be false (zero value) if not in YAML
	if cfg.Workflow.Mode != "branch-per-feature" {
		t.Errorf("Empty config should be filled with default workflow mode, got %s", cfg.Workflow.Mode)
	}
	if cfg.CommitTemplate != "feat: {title} ({id})" {
		t.Errorf("Empty config should be filled with default commit template, got %s", cfg.CommitTemplate)
	}
}

func TestSave_CreatesFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fogit-test-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := fogit.DefaultConfig()
	cfg.Repository.Name = "test-save"

	err = Save(tempDir, cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tempDir, "config.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Reload and verify
	reloaded, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Failed to reload saved config: %v", err)
	}

	if reloaded.Repository.Name != "test-save" {
		t.Errorf("Expected repository.name 'test-save', got %s", reloaded.Repository.Name)
	}
}

func TestSave_Overwrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fogit-test-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save initial config
	cfg1 := fogit.DefaultConfig()
	cfg1.Repository.Name = "initial"
	if err := Save(tempDir, cfg1); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Overwrite with new config
	cfg2 := fogit.DefaultConfig()
	cfg2.Repository.Name = "updated"
	cfg2.AutoCommit = false
	if err := Save(tempDir, cfg2); err != nil {
		t.Fatalf("Failed to overwrite config: %v", err)
	}

	// Reload and verify updated values
	reloaded, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	if reloaded.Repository.Name != "updated" {
		t.Errorf("Expected repository.name 'updated', got %s", reloaded.Repository.Name)
	}
	if reloaded.AutoCommit {
		t.Error("Expected auto_commit false after overwrite")
	}
}

func TestSave_InvalidDirectory(t *testing.T) {
	// Try to save to non-existent directory without creating it
	invalidDir := filepath.Join(os.TempDir(), "non-existent-dir-12345")

	cfg := fogit.DefaultConfig()
	err := Save(invalidDir, cfg)

	if err == nil {
		t.Error("Expected error when saving to non-existent directory")
		// Clean up if somehow it was created
		os.RemoveAll(invalidDir)
	}
}

func TestMergeWithDefaults(t *testing.T) {
	tests := []struct {
		name   string
		input  *fogit.Config
		verify func(t *testing.T, cfg *fogit.Config)
	}{
		{
			name: "fills missing UI fields",
			input: &fogit.Config{
				UI: fogit.UIConfig{
					DefaultGroupBy: "", // Empty
					DefaultLayout:  "", // Empty
				},
			},
			verify: func(t *testing.T, cfg *fogit.Config) {
				if cfg.UI.DefaultGroupBy != "category" {
					t.Errorf("Expected default_group_by filled with 'category', got %s", cfg.UI.DefaultGroupBy)
				}
				if cfg.UI.DefaultLayout != "hierarchical" {
					t.Errorf("Expected default_layout filled with 'hierarchical', got %s", cfg.UI.DefaultLayout)
				}
			},
		},
		{
			name: "preserves custom UI fields",
			input: &fogit.Config{
				UI: fogit.UIConfig{
					DefaultGroupBy: "priority",
					DefaultLayout:  "flat",
				},
			},
			verify: func(t *testing.T, cfg *fogit.Config) {
				if cfg.UI.DefaultGroupBy != "priority" {
					t.Errorf("Expected custom default_group_by 'priority', got %s", cfg.UI.DefaultGroupBy)
				}
				if cfg.UI.DefaultLayout != "flat" {
					t.Errorf("Expected custom default_layout 'flat', got %s", cfg.UI.DefaultLayout)
				}
			},
		},
		{
			name: "fills missing workflow fields",
			input: &fogit.Config{
				Workflow: fogit.WorkflowConfig{
					Mode:       "trunk-based", // Custom
					BaseBranch: "",            // Empty - should be filled
				},
			},
			verify: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.Mode != "trunk-based" {
					t.Errorf("Expected custom mode 'trunk-based', got %s", cfg.Workflow.Mode)
				}
				if cfg.Workflow.BaseBranch != "main" {
					t.Errorf("Expected base_branch filled with 'main', got %s", cfg.Workflow.BaseBranch)
				}
			},
		},
		{
			name: "fills missing commit template",
			input: &fogit.Config{
				CommitTemplate: "", // Empty
			},
			verify: func(t *testing.T, cfg *fogit.Config) {
				if cfg.CommitTemplate != "feat: {title} ({id})" {
					t.Errorf("Expected commit_template filled with default, got %s", cfg.CommitTemplate)
				}
			},
		},
		{
			name: "fills nil relationship types",
			input: &fogit.Config{
				Relationships: fogit.RelationshipsConfig{
					Types: nil, // Nil
				},
			},
			verify: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Relationships.Types == nil {
					t.Error("Expected relationship types to be filled")
				}
				if len(cfg.Relationships.Types) == 0 {
					t.Error("Expected default relationship types to be filled")
				}
				if _, ok := cfg.Relationships.Types["depends-on"]; !ok {
					t.Error("Expected 'depends-on' relationship type in defaults")
				}
			},
		},
		{
			name: "fills missing version_format",
			input: &fogit.Config{
				Workflow: fogit.WorkflowConfig{
					Mode:          "branch-per-feature",
					VersionFormat: "", // Empty - should be filled with default
				},
			},
			verify: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.VersionFormat != "simple" {
					t.Errorf("Expected version_format filled with 'simple', got %s", cfg.Workflow.VersionFormat)
				}
			},
		},
		{
			name: "preserves custom version_format semantic",
			input: &fogit.Config{
				Workflow: fogit.WorkflowConfig{
					Mode:          "branch-per-feature",
					VersionFormat: "semantic",
				},
			},
			verify: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.VersionFormat != "semantic" {
					t.Errorf("Expected custom version_format 'semantic', got %s", cfg.Workflow.VersionFormat)
				}
			},
		},
		{
			name: "preserves custom version_format simple",
			input: &fogit.Config{
				Workflow: fogit.WorkflowConfig{
					VersionFormat: "simple",
				},
			},
			verify: func(t *testing.T, cfg *fogit.Config) {
				if cfg.Workflow.VersionFormat != "simple" {
					t.Errorf("Expected custom version_format 'simple', got %s", cfg.Workflow.VersionFormat)
				}
				// Mode should be filled with default
				if cfg.Workflow.Mode != "branch-per-feature" {
					t.Errorf("Expected mode filled with default, got %s", cfg.Workflow.Mode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergeWithDefaults(tt.input)
			tt.verify(t, tt.input)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that Save -> Load preserves all data
	tempDir, err := os.MkdirTemp("", "fogit-test-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	original := fogit.DefaultConfig()
	original.Repository.Name = "roundtrip-test"
	original.Repository.InitializedAt = time.Date(2025, 10, 30, 12, 0, 0, 0, time.UTC)
	original.AutoCommit = false
	original.AutoPush = true
	original.Workflow.Mode = "trunk-based"
	original.Workflow.VersionFormat = "semantic"
	original.UI.DefaultGroupBy = "priority"

	// Save
	if err := Save(tempDir, original); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load
	loaded, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Compare critical fields
	if loaded.Repository.Name != original.Repository.Name {
		t.Error("Repository.Name mismatch")
	}
	if !loaded.Repository.InitializedAt.Equal(original.Repository.InitializedAt) {
		t.Errorf("InitializedAt mismatch: %v != %v", loaded.Repository.InitializedAt, original.Repository.InitializedAt)
	}
	if loaded.AutoCommit != original.AutoCommit {
		t.Error("AutoCommit mismatch")
	}
	if loaded.AutoPush != original.AutoPush {
		t.Error("AutoPush mismatch")
	}
	if loaded.Workflow.Mode != original.Workflow.Mode {
		t.Error("Workflow.Mode mismatch")
	}
	if loaded.Workflow.VersionFormat != original.Workflow.VersionFormat {
		t.Errorf("Workflow.VersionFormat mismatch: got %q, want %q", loaded.Workflow.VersionFormat, original.Workflow.VersionFormat)
	}
	if loaded.UI.DefaultGroupBy != original.UI.DefaultGroupBy {
		t.Error("UI.DefaultGroupBy mismatch")
	}
}
