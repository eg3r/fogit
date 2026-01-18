package fogit

import (
	"fmt"
	"time"
)

// Config represents the FoGit repository configuration (.fogit/config.yml)
type Config struct {
	Repository      RepositoryConfig    `yaml:"repository"`
	UI              UIConfig            `yaml:"ui"`
	Workflow        WorkflowConfig      `yaml:"workflow"`
	AutoCommit      bool                `yaml:"auto_commit"`
	CommitTemplate  string              `yaml:"commit_template"`
	AutoPush        bool                `yaml:"auto_push"`
	Relationships   RelationshipsConfig `yaml:"relationships"`
	FeatureSearch   FeatureSearchConfig `yaml:"feature_search"`
	DefaultPriority string              `yaml:"default_priority,omitempty"` // Optional default priority for new features
}

// RepositoryConfig contains repository metadata
type RepositoryConfig struct {
	Name          string    `yaml:"name"`
	InitializedAt time.Time `yaml:"initialized_at"`
	Version       string    `yaml:"version"`
}

// UIConfig contains user interface defaults
type UIConfig struct {
	DefaultGroupBy string `yaml:"default_group_by"`
	DefaultLayout  string `yaml:"default_layout"`
}

// WorkflowConfig contains Git workflow settings
type WorkflowConfig struct {
	Mode                string `yaml:"mode"` // "branch-per-feature" or "trunk-based"
	BaseBranch          string `yaml:"base_branch"`
	AllowSharedBranches bool   `yaml:"allow_shared_branches"`
	VersionFormat       string `yaml:"version_format"` // "simple" (1, 2, 3) or "semantic" (1.0.0, 1.1.0, 2.0.0)
}

// RelationshipsConfig contains relationship system configuration
type RelationshipsConfig struct {
	System     RelationshipSystem                `yaml:"system"`
	Categories map[string]RelationshipCategory   `yaml:"categories"`
	Types      map[string]RelationshipTypeConfig `yaml:"types"`
	Defaults   RelationshipDefaults              `yaml:"defaults"`
}

// RelationshipSystem contains system-wide relationship settings
type RelationshipSystem struct {
	AllowCustomTypes      bool `yaml:"allow_custom_types"`
	AllowCustomCategories bool `yaml:"allow_custom_categories"`
	AutoCreateInverse     bool `yaml:"auto_create_inverse"`
}

// RelationshipCategory defines a category of relationships with validation rules
// Per spec, relationship history is tracked via Git (git log -p .fogit/features/)
type RelationshipCategory struct {
	Description     string                 `yaml:"description"`
	AllowCycles     bool                   `yaml:"allow_cycles"`
	CycleDetection  string                 `yaml:"cycle_detection"` // "strict", "warn", "none"
	IncludeInImpact bool                   `yaml:"include_in_impact"`
	Metadata        map[string]interface{} `yaml:"metadata,omitempty"`
}

// RelationshipTypeConfig defines a relationship type configuration
type RelationshipTypeConfig struct {
	Category      string   `yaml:"category"`
	Inverse       string   `yaml:"inverse,omitempty"`
	Bidirectional bool     `yaml:"bidirectional"`
	Description   string   `yaml:"description,omitempty"`
	Aliases       []string `yaml:"aliases,omitempty"`
}

// RelationshipDefaults contains default values for relationships
type RelationshipDefaults struct {
	Category string `yaml:"relationship_category"`  // Default category for undefined types
	TreeType string `yaml:"tree_relationship_type"` // Default type for tree command
}

// FeatureSearchConfig contains fuzzy search configuration
type FeatureSearchConfig struct {
	FuzzyMatch     bool    `yaml:"fuzzy_match"`
	MinSimilarity  float64 `yaml:"min_similarity"` // 0-100
	MaxSuggestions int     `yaml:"max_suggestions"`
}

// DefaultConfig returns a Config with sensible defaults per spec
func DefaultConfig() *Config {
	return &Config{
		Repository: RepositoryConfig{
			Name:          "",
			InitializedAt: time.Now().UTC(),
			Version:       "1.0.0",
		},
		UI: UIConfig{
			DefaultGroupBy: "category",
			DefaultLayout:  "hierarchical",
		},
		Workflow: WorkflowConfig{
			Mode:                "branch-per-feature",
			BaseBranch:          "main",
			AllowSharedBranches: true,
			VersionFormat:       "simple", // Default to simple versioning (1, 2, 3)
		},
		AutoCommit:     true,
		CommitTemplate: "feat: {title} ({id})",
		AutoPush:       false,
		FeatureSearch: FeatureSearchConfig{
			FuzzyMatch:     true,
			MinSimilarity:  60.0,
			MaxSuggestions: 5,
		},
		Relationships: RelationshipsConfig{
			System: RelationshipSystem{
				AllowCustomTypes:      true,
				AllowCustomCategories: true,
				AutoCreateInverse:     true,
			},
			Categories: map[string]RelationshipCategory{
				"structural": {
					Description:     "Dependencies that form system architecture",
					AllowCycles:     false,
					CycleDetection:  "strict",
					IncludeInImpact: true,
				},
				"informational": {
					Description:     "References and associations",
					AllowCycles:     true,
					CycleDetection:  "none",
					IncludeInImpact: false,
				},
				"workflow": {
					Description:     "Process and approval relationships",
					AllowCycles:     false,
					CycleDetection:  "warn",
					IncludeInImpact: true,
				},
				"compliance": {
					Description:     "Regulatory and audit trail relationships",
					AllowCycles:     false,
					CycleDetection:  "strict",
					IncludeInImpact: false,
				},
			},
			Types: map[string]RelationshipTypeConfig{
				"depends-on": {
					Category:      "structural",
					Inverse:       "required-by",
					Bidirectional: false,
					Description:   "Feature requires another feature to function",
					Aliases:       []string{"requires", "needs"},
				},
				"required-by": {
					Category:      "structural",
					Inverse:       "depends-on",
					Bidirectional: false,
					Description:   "Feature is required by another feature",
				},
				"contains": {
					Category:      "structural",
					Inverse:       "contained-by",
					Bidirectional: false,
					Description:   "Feature contains another feature as a component",
					Aliases:       []string{"has", "includes"},
				},
				"contained-by": {
					Category:      "structural",
					Inverse:       "contains",
					Bidirectional: false,
					Description:   "Feature is contained by another feature",
				},
				"implements": {
					Category:      "structural",
					Inverse:       "implemented-by",
					Bidirectional: false,
					Description:   "Feature implements a specification or interface",
				},
				"implemented-by": {
					Category:      "structural",
					Inverse:       "implements",
					Bidirectional: false,
					Description:   "Feature is implemented by another feature",
				},
				"replaces": {
					Category:      "structural",
					Inverse:       "replaced-by",
					Bidirectional: false,
					Description:   "Feature supersedes another feature",
				},
				"replaced-by": {
					Category:      "structural",
					Inverse:       "replaces",
					Bidirectional: false,
					Description:   "Feature is superseded by another feature",
				},
				"references": {
					Category:      "informational",
					Inverse:       "referenced-by",
					Bidirectional: false,
					Description:   "Feature references another feature",
					Aliases:       []string{"uses", "mentions"},
				},
				"referenced-by": {
					Category:      "informational",
					Inverse:       "references",
					Bidirectional: false,
					Description:   "Feature is referenced by another feature",
				},
				"related-to": {
					Category:      "informational",
					Bidirectional: true,
					Description:   "Features are related without specific dependency",
					Aliases:       []string{"relates", "associated-with"},
				},
				"conflicts-with": {
					Category:      "informational",
					Bidirectional: true,
					Description:   "Features cannot coexist in the same system",
					Aliases:       []string{"incompatible-with"},
				},
				"tested-by": {
					Category:      "informational",
					Inverse:       "tests",
					Bidirectional: false,
					Description:   "Feature is tested by another feature",
				},
				"tests": {
					Category:      "informational",
					Inverse:       "tested-by",
					Bidirectional: false,
					Description:   "Feature tests another feature",
				},
				"blocks": {
					Category:      "workflow",
					Inverse:       "blocked-by",
					Bidirectional: false,
					Description:   "Feature blocks progress on another feature",
				},
				"blocked-by": {
					Category:      "workflow",
					Inverse:       "blocks",
					Bidirectional: false,
					Description:   "Feature is blocked by another feature",
				},
			},
			Defaults: RelationshipDefaults{
				Category: "informational",
				TreeType: "depends-on",
			},
		},
	}
}

// Validate checks the configuration for integrity and consistency
func (c *Config) Validate() error {
	// 1. Validate defaults reference existing types and categories
	if c.Relationships.Defaults.TreeType != "" {
		if _, exists := c.Relationships.Types[c.Relationships.Defaults.TreeType]; !exists {
			return fmt.Errorf("defaults.tree_relationship_type '%s' not defined in relationship_types",
				c.Relationships.Defaults.TreeType)
		}
	}

	if c.Relationships.Defaults.Category != "" {
		if _, exists := c.Relationships.Categories[c.Relationships.Defaults.Category]; !exists {
			return fmt.Errorf("defaults.relationship_category '%s' not defined in relationship_categories",
				c.Relationships.Defaults.Category)
		}
	}

	// 2. Validate all relationship types
	for typeName, typeDef := range c.Relationships.Types {
		// Check category exists
		if _, exists := c.Relationships.Categories[typeDef.Category]; !exists {
			return fmt.Errorf("relationship type '%s' references unknown category '%s'",
				typeName, typeDef.Category)
		}

		// Check inverse type exists (if specified and not bidirectional)
		if !typeDef.Bidirectional && typeDef.Inverse != "" {
			if _, exists := c.Relationships.Types[typeDef.Inverse]; !exists {
				return fmt.Errorf("relationship type '%s' specifies inverse '%s' which is not defined",
					typeName, typeDef.Inverse)
			}

			// Verify inverse points back
			inverseType := c.Relationships.Types[typeDef.Inverse]
			if inverseType.Inverse != typeName {
				return fmt.Errorf("relationship type '%s' has inverse '%s', but '%s' inverse is '%s' (should be '%s')",
					typeName, typeDef.Inverse, typeDef.Inverse, inverseType.Inverse, typeName)
			}
		}

		// Check bidirectional types don't have inverse (semantic conflict)
		if typeDef.Bidirectional && typeDef.Inverse != "" {
			return fmt.Errorf("relationship type '%s' is bidirectional but also defines inverse '%s' (bidirectional types don't need inverses)",
				typeName, typeDef.Inverse)
		}
	}

	// 3. Validate relationship categories
	for catName, catDef := range c.Relationships.Categories {
		// Check cycle_detection values
		switch catDef.CycleDetection {
		case "strict", "warn", "none":
			// Valid
		default:
			return fmt.Errorf("category '%s' has invalid cycle_detection '%s' (must be: strict, warn, none)",
				catName, catDef.CycleDetection)
		}

		// Logical check: if allow_cycles is true, cycle_detection should be "none"
		if catDef.AllowCycles && catDef.CycleDetection != "none" {
			return fmt.Errorf("category '%s' has allow_cycles=true but cycle_detection='%s' (should be 'none' when cycles allowed)",
				catName, catDef.CycleDetection)
		}
	}

	// 4. Validate workflow settings
	switch c.Workflow.Mode {
	case "branch-per-feature", "trunk-based":
		// Valid
	default:
		return fmt.Errorf("workflow.mode '%s' is invalid (must be: branch-per-feature, trunk-based)",
			c.Workflow.Mode)
	}

	switch c.Workflow.VersionFormat {
	case "simple", "semantic":
		// Valid
	default:
		return fmt.Errorf("workflow.version_format '%s' is invalid (must be: simple, semantic)",
			c.Workflow.VersionFormat)
	}

	// 5. Validate default priority if set
	if c.DefaultPriority != "" {
		validPriorities := []string{"low", "medium", "high", "critical"}
		valid := false
		for _, p := range validPriorities {
			if c.DefaultPriority == p {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("default_priority '%s' is invalid (must be: low, medium, high, critical)",
				c.DefaultPriority)
		}
	}

	return nil
}
