package fogit

import (
	"strings"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Config)
		wantErr string
	}{
		{
			name:    "valid default config",
			setup:   func(c *Config) {},
			wantErr: "",
		},
		{
			name: "invalid tree type default",
			setup: func(c *Config) {
				c.Relationships.Defaults.TreeType = "nonexistent-type"
			},
			wantErr: "not defined in relationship_types",
		},
		{
			name: "invalid category default",
			setup: func(c *Config) {
				c.Relationships.Defaults.Category = "nonexistent-category"
			},
			wantErr: "not defined in relationship_categories",
		},
		{
			name: "type references unknown category",
			setup: func(c *Config) {
				c.Relationships.Types["test"] = RelationshipTypeConfig{
					Category:    "unknown-category",
					Description: "test",
				}
			},
			wantErr: "references unknown category",
		},
		{
			name: "type with missing inverse",
			setup: func(c *Config) {
				c.Relationships.Types["test"] = RelationshipTypeConfig{
					Category:      "structural",
					Inverse:       "nonexistent-inverse",
					Bidirectional: false,
					Description:   "test",
				}
			},
			wantErr: "inverse 'nonexistent-inverse' which is not defined",
		},
		{
			name: "asymmetric inverse relationships",
			setup: func(c *Config) {
				c.Relationships.Types["test-a"] = RelationshipTypeConfig{
					Category:      "structural",
					Inverse:       "test-b",
					Bidirectional: false,
					Description:   "test a",
				}
				c.Relationships.Types["test-b"] = RelationshipTypeConfig{
					Category:      "structural",
					Inverse:       "test-c", // Wrong! Should be "test-a"
					Bidirectional: false,
					Description:   "test b",
				}
				c.Relationships.Types["test-c"] = RelationshipTypeConfig{
					Category:      "structural",
					Inverse:       "test-b",
					Bidirectional: false,
					Description:   "test c",
				}
			},
			wantErr: "inverse is 'test-c' (should be 'test-a')",
		},
		{
			name: "bidirectional with inverse",
			setup: func(c *Config) {
				c.Relationships.Types["test"] = RelationshipTypeConfig{
					Category:      "informational",
					Inverse:       "test-inverse",
					Bidirectional: true,
					Description:   "test",
				}
			},
			wantErr: "bidirectional but also defines inverse",
		},
		{
			name: "invalid cycle detection value",
			setup: func(c *Config) {
				c.Relationships.Categories["structural"] = RelationshipCategory{
					Description:     "test",
					AllowCycles:     false,
					CycleDetection:  "invalid", // Not strict/warn/none
					IncludeInImpact: true,
				}
			},
			wantErr: "invalid cycle_detection 'invalid'",
		},
		{
			name: "allow cycles with non-none detection",
			setup: func(c *Config) {
				c.Relationships.Categories["informational"] = RelationshipCategory{
					Description:     "test",
					AllowCycles:     true,
					CycleDetection:  "strict", // Should be "none" when allow_cycles is true
					IncludeInImpact: false,
				}
			},
			wantErr: "allow_cycles=true but cycle_detection='strict'",
		},
		{
			name: "invalid workflow mode",
			setup: func(c *Config) {
				c.Workflow.Mode = "invalid-mode"
			},
			wantErr: "workflow.mode 'invalid-mode' is invalid",
		},
		{
			name: "invalid version format",
			setup: func(c *Config) {
				c.Workflow.VersionFormat = "invalid-format"
			},
			wantErr: "workflow.version_format 'invalid-format' is invalid",
		},
		{
			name: "invalid default priority",
			setup: func(c *Config) {
				c.DefaultPriority = "urgent"
			},
			wantErr: "default_priority 'urgent' is invalid",
		},
		{
			name: "valid symmetric inverse relationships",
			setup: func(c *Config) {
				c.Relationships.Types["test-a"] = RelationshipTypeConfig{
					Category:      "structural",
					Inverse:       "test-b",
					Bidirectional: false,
					Description:   "test a",
				}
				c.Relationships.Types["test-b"] = RelationshipTypeConfig{
					Category:      "structural",
					Inverse:       "test-a",
					Bidirectional: false,
					Description:   "test b",
				}
			},
			wantErr: "",
		},
		{
			name: "valid bidirectional without inverse",
			setup: func(c *Config) {
				c.Relationships.Types["test"] = RelationshipTypeConfig{
					Category:      "informational",
					Bidirectional: true,
					Description:   "test",
				}
			},
			wantErr: "",
		},
		{
			name: "valid cycle detection modes",
			setup: func(c *Config) {
				c.Relationships.Categories["test1"] = RelationshipCategory{
					Description:     "test1",
					AllowCycles:     false,
					CycleDetection:  "strict",
					IncludeInImpact: true,
				}
				c.Relationships.Categories["test2"] = RelationshipCategory{
					Description:     "test2",
					AllowCycles:     false,
					CycleDetection:  "warn",
					IncludeInImpact: true,
				}
				c.Relationships.Categories["test3"] = RelationshipCategory{
					Description:     "test3",
					AllowCycles:     true,
					CycleDetection:  "none",
					IncludeInImpact: true,
				}
			},
			wantErr: "",
		},
		{
			name: "empty tree type default is valid",
			setup: func(c *Config) {
				c.Relationships.Defaults.TreeType = ""
			},
			wantErr: "",
		},
		{
			name: "empty category default is valid",
			setup: func(c *Config) {
				c.Relationships.Defaults.Category = ""
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.setup(cfg)

			err := cfg.Validate()

			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestConfig_Validate_AllDefaultCategories(t *testing.T) {
	cfg := DefaultConfig()

	// Verify all 4 default categories are valid
	expectedCategories := []string{"structural", "informational", "workflow", "compliance"}

	for _, cat := range expectedCategories {
		if _, exists := cfg.Relationships.Categories[cat]; !exists {
			t.Errorf("Expected category %q to exist in default config", cat)
		}
	}

	// Validate the default config
	if err := cfg.Validate(); err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}
}

func TestConfig_Validate_AllDefaultTypes(t *testing.T) {
	cfg := DefaultConfig()

	// Verify all 9 default types are valid
	expectedTypes := map[string]string{
		"depends-on":     "structural",
		"contains":       "structural",
		"implements":     "structural",
		"replaces":       "structural",
		"references":     "informational",
		"related-to":     "informational",
		"conflicts-with": "informational",
		"tested-by":      "informational",
		"blocks":         "workflow",
	}

	for typeName, expectedCat := range expectedTypes {
		typeDef, exists := cfg.Relationships.Types[typeName]
		if !exists {
			t.Errorf("Expected type %q to exist in default config", typeName)
			continue
		}
		if typeDef.Category != expectedCat {
			t.Errorf("Type %q: expected category %q, got %q", typeName, expectedCat, typeDef.Category)
		}
	}

	// Validate the default config
	if err := cfg.Validate(); err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}
}

func TestConfig_Validate_CategorySettings(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		category       string
		allowCycles    bool
		cycleDetection string
		includeImpact  bool
	}{
		{"structural", false, "strict", true},
		{"informational", true, "none", false},
		{"workflow", false, "warn", true},
		{"compliance", false, "strict", false},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			cat, exists := cfg.Relationships.Categories[tt.category]
			if !exists {
				t.Fatalf("Category %q not found", tt.category)
			}

			if cat.AllowCycles != tt.allowCycles {
				t.Errorf("AllowCycles: expected %v, got %v", tt.allowCycles, cat.AllowCycles)
			}
			if cat.CycleDetection != tt.cycleDetection {
				t.Errorf("CycleDetection: expected %q, got %q", tt.cycleDetection, cat.CycleDetection)
			}
			if cat.IncludeInImpact != tt.includeImpact {
				t.Errorf("IncludeInImpact: expected %v, got %v", tt.includeImpact, cat.IncludeInImpact)
			}
		})
	}
}
