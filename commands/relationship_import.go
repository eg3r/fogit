package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	relImportMerge     bool
	relImportOverwrite bool
)

var relationshipImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import relationship type and category definitions",
	Long: `Import relationship type and category definitions from JSON or YAML file.

This enables sharing configurations across repositories and teams.

Import Conflict Handling:
  (default)    Error - abort import, no changes made
  --merge      Skip - keep existing definition, import only new ones
  --overwrite  Replace - overwrite existing definitions with imported data

Validation:
  - Imported types must reference valid categories (either in import file or already defined)
  - Inverse relationships must be consistent (if A.inverse = B, then B.inverse = A)
  - Category names and type names must be unique within their respective groups
  - cycle_detection must be one of: strict, warn, none

Examples:
  # Import definitions (error on conflict)
  fogit relationship import definitions.json

  # Merge with existing (keep existing on conflict)
  fogit relationship import team-standards.yaml --merge

  # Overwrite existing definitions
  fogit relationship import org-standard.yaml --overwrite`,
	Args: cobra.ExactArgs(1),
	RunE: runRelationshipImport,
}

func init() {
	relationshipImportCmd.Flags().BoolVar(&relImportMerge, "merge", false, "Merge with existing definitions (skip conflicts)")
	relationshipImportCmd.Flags().BoolVar(&relImportOverwrite, "overwrite", false, "Overwrite existing definitions")

	// Add as subcommand to relationship
	relationshipCmd.AddCommand(relationshipImportCmd)
}

func runRelationshipImport(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Validate flags
	if relImportMerge && relImportOverwrite {
		return fmt.Errorf("cannot specify both --merge and --overwrite")
	}

	// Read input file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Determine format from extension
	ext := strings.ToLower(filepath.Ext(filePath))
	var importData RelationshipExport

	switch ext {
	case ".json":
		if jsonErr := json.Unmarshal(data, &importData); jsonErr != nil {
			return fmt.Errorf("failed to parse JSON: %w", jsonErr)
		}
	case ".yaml", ".yml":
		if yamlErr := yaml.Unmarshal(data, &importData); yamlErr != nil {
			return fmt.Errorf("failed to parse YAML: %w", yamlErr)
		}
	default:
		// Try JSON first, then YAML
		if jsonErr := json.Unmarshal(data, &importData); jsonErr != nil {
			if yamlErr := yaml.Unmarshal(data, &importData); yamlErr != nil {
				return fmt.Errorf("failed to parse file (tried JSON and YAML)")
			}
		}
	}

	// Validate imported data
	if validateErr := validateRelationshipImportData(&importData); validateErr != nil {
		return fmt.Errorf("validation failed: %w", validateErr)
	}

	// Get fogit directory
	fogitDir, err := getFogitDir()
	if err != nil {
		return fmt.Errorf("failed to get .fogit directory: %w", err)
	}

	// Load current config
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Track what will be imported
	var categoriesAdded, categoriesSkipped, categoriesOverwritten int
	var typesAdded, typesSkipped, typesOverwritten int

	// Import categories first (types depend on categories)
	for name, cat := range importData.RelationshipCategories {
		_, exists := cfg.Relationships.Categories[name]
		if exists {
			if relImportMerge {
				categoriesSkipped++
				continue
			}
			if !relImportOverwrite {
				return fmt.Errorf("category '%s' already exists (use --merge to skip or --overwrite to replace)", name)
			}
			categoriesOverwritten++
		} else {
			categoriesAdded++
		}

		if cfg.Relationships.Categories == nil {
			cfg.Relationships.Categories = make(map[string]fogit.RelationshipCategory)
		}
		cfg.Relationships.Categories[name] = fogit.RelationshipCategory{
			Description:     cat.Description,
			AllowCycles:     cat.AllowCycles,
			CycleDetection:  cat.CycleDetection,
			IncludeInImpact: cat.IncludeInImpact,
		}
	}

	// Import types
	for name, typ := range importData.RelationshipTypes {
		// Validate category exists (in config or import)
		if _, exists := cfg.Relationships.Categories[typ.Category]; !exists {
			return fmt.Errorf("type '%s' references unknown category '%s'", name, typ.Category)
		}

		_, exists := cfg.Relationships.Types[name]
		if exists {
			if relImportMerge {
				typesSkipped++
				continue
			}
			if !relImportOverwrite {
				return fmt.Errorf("type '%s' already exists (use --merge to skip or --overwrite to replace)", name)
			}
			typesOverwritten++
		} else {
			typesAdded++
		}

		if cfg.Relationships.Types == nil {
			cfg.Relationships.Types = make(map[string]fogit.RelationshipTypeConfig)
		}
		cfg.Relationships.Types[name] = fogit.RelationshipTypeConfig{
			Category:      typ.Category,
			Description:   typ.Description,
			Inverse:       typ.Inverse,
			Bidirectional: typ.Bidirectional,
			Aliases:       typ.Aliases,
		}
	}

	// Validate inverse consistency after all types are imported
	if err := validateRelationshipInverseConsistency(cfg); err != nil {
		return fmt.Errorf("inverse consistency check failed: %w", err)
	}

	// Validate config before saving
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Save config
	if err := config.Save(fogitDir, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Print summary
	fmt.Println("Import complete:")
	fmt.Printf("  Categories: %d added", categoriesAdded)
	if categoriesSkipped > 0 {
		fmt.Printf(", %d skipped", categoriesSkipped)
	}
	if categoriesOverwritten > 0 {
		fmt.Printf(", %d overwritten", categoriesOverwritten)
	}
	fmt.Println()
	fmt.Printf("  Types:      %d added", typesAdded)
	if typesSkipped > 0 {
		fmt.Printf(", %d skipped", typesSkipped)
	}
	if typesOverwritten > 0 {
		fmt.Printf(", %d overwritten", typesOverwritten)
	}
	fmt.Println()

	return nil
}

// validateRelationshipImportData validates the import data structure for relationship definitions
func validateRelationshipImportData(data *RelationshipExport) error {
	// Validate cycle detection modes
	for name, cat := range data.RelationshipCategories {
		switch cat.CycleDetection {
		case "strict", "warn", "none", "":
			// Valid
		default:
			return fmt.Errorf("category '%s' has invalid cycle_detection '%s' (must be strict, warn, or none)", name, cat.CycleDetection)
		}
	}

	// Validate type category references within the import
	importedCategories := make(map[string]bool)
	for name := range data.RelationshipCategories {
		importedCategories[name] = true
	}

	for name, typ := range data.RelationshipTypes {
		if typ.Category == "" {
			return fmt.Errorf("type '%s' has no category specified", name)
		}
		// Note: We don't check if category exists in import here because it might exist in the config
		// This is validated later during import
	}

	// Validate inverse consistency within the import
	for name, typ := range data.RelationshipTypes {
		if typ.Inverse != "" && !typ.Bidirectional {
			if inverseType, exists := data.RelationshipTypes[typ.Inverse]; exists {
				if inverseType.Inverse != name {
					return fmt.Errorf("inconsistent inverse relationship: '%s' declares inverse '%s', but '%s' declares inverse '%s'",
						name, typ.Inverse, typ.Inverse, inverseType.Inverse)
				}
			}
		}
		if typ.Bidirectional && typ.Inverse != "" {
			return fmt.Errorf("type '%s' is bidirectional but has an inverse (bidirectional types are their own inverse)", name)
		}
	}

	return nil
}

// validateRelationshipInverseConsistency checks that inverse relationships are consistent in the config
func validateRelationshipInverseConsistency(cfg *fogit.Config) error {
	for name, typ := range cfg.Relationships.Types {
		if typ.Inverse != "" && !typ.Bidirectional {
			inverseType, exists := cfg.Relationships.Types[typ.Inverse]
			if exists {
				if inverseType.Inverse != name {
					return fmt.Errorf("inconsistent inverse: '%s' declares inverse '%s', but '%s' declares inverse '%s'",
						name, typ.Inverse, typ.Inverse, inverseType.Inverse)
				}
			}
			// Note: It's OK if the inverse doesn't exist - it might be created later
		}
	}
	return nil
}
