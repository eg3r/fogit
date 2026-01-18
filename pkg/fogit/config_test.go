package fogit

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Test Repository defaults
	t.Run("Repository", func(t *testing.T) {
		if cfg.Repository.Version != "1.0.0" {
			t.Errorf("Expected version 1.0.0, got %s", cfg.Repository.Version)
		}
		if cfg.Repository.InitializedAt.IsZero() {
			t.Error("InitializedAt should not be zero")
		}
		if time.Since(cfg.Repository.InitializedAt) > time.Minute {
			t.Error("InitializedAt should be recent")
		}
	})

	// Test UI defaults
	t.Run("UI", func(t *testing.T) {
		if cfg.UI.DefaultGroupBy != "category" {
			t.Errorf("Expected default_group_by 'category', got %s", cfg.UI.DefaultGroupBy)
		}
		if cfg.UI.DefaultLayout != "hierarchical" {
			t.Errorf("Expected default_layout 'hierarchical', got %s", cfg.UI.DefaultLayout)
		}
	})

	// Test Workflow defaults
	t.Run("Workflow", func(t *testing.T) {
		if cfg.Workflow.Mode != "branch-per-feature" {
			t.Errorf("Expected mode 'branch-per-feature', got %s", cfg.Workflow.Mode)
		}
		if cfg.Workflow.BaseBranch != "main" {
			t.Errorf("Expected base_branch 'main', got %s", cfg.Workflow.BaseBranch)
		}
		if !cfg.Workflow.AllowSharedBranches {
			t.Error("Expected allow_shared_branches to be true")
		}
	})

	// Test Git integration defaults
	t.Run("Git Integration", func(t *testing.T) {
		if !cfg.AutoCommit {
			t.Error("Expected auto_commit to be true by default")
		}
		if cfg.CommitTemplate != "feat: {title} ({id})" {
			t.Errorf("Expected commit_template 'feat: {title} ({id})', got %s", cfg.CommitTemplate)
		}
		if cfg.AutoPush {
			t.Error("Expected auto_push to be false by default")
		}
	})

	// Test Relationships defaults
	t.Run("Relationships", func(t *testing.T) {
		// Test system defaults
		if !cfg.Relationships.System.AllowCustomTypes {
			t.Error("Expected allow_custom_types to be true")
		}
		if !cfg.Relationships.System.AllowCustomCategories {
			t.Error("Expected allow_custom_categories to be true")
		}
		if !cfg.Relationships.System.AutoCreateInverse {
			t.Error("Expected auto_create_inverse to be true")
		}

		// Test categories
		expectedCategories := []string{"structural", "informational", "workflow", "compliance"}
		for _, catName := range expectedCategories {
			if _, ok := cfg.Relationships.Categories[catName]; !ok {
				t.Errorf("Expected category %q to exist in defaults", catName)
			}
		}

		// Test relationship types
		expectedTypes := []string{"depends-on", "contains", "implements", "replaces", "references", "related-to", "conflicts-with", "tested-by", "blocks"}
		for _, typeName := range expectedTypes {
			if _, ok := cfg.Relationships.Types[typeName]; !ok {
				t.Errorf("Expected relationship type %q to exist in defaults", typeName)
			}
		}

		// Test defaults
		if cfg.Relationships.Defaults.Category != "informational" {
			t.Errorf("Expected default category 'informational', got %s", cfg.Relationships.Defaults.Category)
		}
		if cfg.Relationships.Defaults.TreeType != "depends-on" {
			t.Errorf("Expected default tree type 'depends-on', got %s", cfg.Relationships.Defaults.TreeType)
		}

		// Test specific relationship type properties
		dependsOn := cfg.Relationships.Types["depends-on"]
		if dependsOn.Category != "structural" {
			t.Errorf("depends-on: expected category 'structural', got %s", dependsOn.Category)
		}
		if dependsOn.Inverse != "required-by" {
			t.Errorf("depends-on: expected inverse 'required-by', got %s", dependsOn.Inverse)
		}
		if dependsOn.Bidirectional {
			t.Error("depends-on: expected bidirectional to be false")
		}

		conflictsWith := cfg.Relationships.Types["conflicts-with"]
		if conflictsWith.Category != "informational" {
			t.Errorf("conflicts-with: expected category 'informational', got %s", conflictsWith.Category)
		}
		if !conflictsWith.Bidirectional {
			t.Error("conflicts-with: expected bidirectional to be true")
		}
	})

	// Test FeatureSearch defaults
	t.Run("FeatureSearch", func(t *testing.T) {
		if !cfg.FeatureSearch.FuzzyMatch {
			t.Error("Expected fuzzy_match to be true by default")
		}
		if cfg.FeatureSearch.MinSimilarity != 60.0 {
			t.Errorf("Expected min_similarity 60.0, got %.2f", cfg.FeatureSearch.MinSimilarity)
		}
		if cfg.FeatureSearch.MaxSuggestions != 5 {
			t.Errorf("Expected max_suggestions 5, got %d", cfg.FeatureSearch.MaxSuggestions)
		}
	})
}

func TestDefaultConfigImmutability(t *testing.T) {
	// Test that calling DefaultConfig() multiple times returns independent instances
	cfg1 := DefaultConfig()
	cfg2 := DefaultConfig()

	// Modify cfg1
	cfg1.AutoCommit = false
	cfg1.Repository.Name = "test-repo"
	cfg1.Workflow.Mode = "trunk-based"

	// Verify cfg2 is not affected
	if !cfg2.AutoCommit {
		t.Error("Modifying one config instance affected another")
	}
	if cfg2.Repository.Name == "test-repo" {
		t.Error("Repository name change affected another instance")
	}
	if cfg2.Workflow.Mode == "trunk-based" {
		t.Error("Workflow mode change affected another instance")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Config)
		isValid bool
		desc    string
	}{
		{
			name:    "default config is valid",
			setup:   func(c *Config) {},
			isValid: true,
			desc:    "Default config should be valid",
		},
		{
			name: "auto_commit false is valid",
			setup: func(c *Config) {
				c.AutoCommit = false
			},
			isValid: true,
			desc:    "auto_commit can be disabled",
		},
		{
			name: "trunk-based workflow is valid",
			setup: func(c *Config) {
				c.Workflow.Mode = "trunk-based"
			},
			isValid: true,
			desc:    "trunk-based workflow mode should be valid",
		},
		{
			name: "custom commit template is valid",
			setup: func(c *Config) {
				c.CommitTemplate = "[Feature] {name}"
			},
			isValid: true,
			desc:    "Custom commit templates should be allowed",
		},
		{
			name: "empty relationship types map",
			setup: func(c *Config) {
				c.Relationships.Types = make(map[string]RelationshipTypeConfig)
			},
			isValid: true,
			desc:    "Empty relationship types should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.setup(cfg)

			// For now we just verify the config can be created
			// In the future we might add a Validate() method
			if cfg == nil {
				t.Error("Config should not be nil")
			}
		})
	}
}

func TestConfigStructFields(t *testing.T) {
	// Test that all expected fields exist and have correct types
	cfg := &Config{
		Repository: RepositoryConfig{
			Name:          "test",
			InitializedAt: time.Now(),
			Version:       "1.0",
		},
		UI: UIConfig{
			DefaultGroupBy: "priority",
			DefaultLayout:  "flat",
		},
		Workflow: WorkflowConfig{
			Mode:                "branch-per-feature",
			BaseBranch:          "develop",
			AllowSharedBranches: false,
		},
		AutoCommit:     false,
		CommitTemplate: "custom: {id}",
		AutoPush:       true,
		Relationships: RelationshipsConfig{
			System: RelationshipSystem{
				AllowCustomTypes:      true,
				AllowCustomCategories: false,
				AutoCreateInverse:     false,
			},
			Categories: map[string]RelationshipCategory{
				"structural": {
					Description:     "Test structural",
					AllowCycles:     true,
					CycleDetection:  "warn",
					IncludeInImpact: true,
				},
			},
			Types: map[string]RelationshipTypeConfig{
				"test": {
					Category:      "structural",
					Inverse:       "test-inverse",
					Bidirectional: true,
					Description:   "test relationship",
				},
			},
			Defaults: RelationshipDefaults{
				Category: "structural",
				TreeType: "test",
			},
		},
	}

	// Verify all fields are accessible
	if cfg.Repository.Name != "test" {
		t.Error("Repository.Name not accessible")
	}
	if cfg.UI.DefaultGroupBy != "priority" {
		t.Error("UI.DefaultGroupBy not accessible")
	}
	if cfg.Workflow.Mode != "branch-per-feature" {
		t.Error("Workflow.Mode not accessible")
	}
	if cfg.AutoCommit != false {
		t.Error("AutoCommit not accessible")
	}
	if cfg.CommitTemplate != "custom: {id}" {
		t.Error("CommitTemplate not accessible")
	}
	if cfg.AutoPush != true {
		t.Error("AutoPush not accessible")
	}
	if !cfg.Relationships.System.AllowCustomTypes {
		t.Error("Relationships.System.AllowCustomTypes not accessible")
	}
	if len(cfg.Relationships.Categories) != 1 {
		t.Error("Relationships.Categories not accessible")
	}
	if len(cfg.Relationships.Types) != 1 {
		t.Error("Relationships.Types not accessible")
	}
}
