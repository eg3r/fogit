package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	defineCategory      string
	defineInverse       string
	defineDescription   string
	defineBidirectional bool
	defineAliases       []string
	defineFrom          string
)

var relationshipDefineCmd = &cobra.Command{
	Use:   "define <name>",
	Short: "Define a new relationship type",
	Long: `Define a new custom relationship type.

The type will be added to the repository configuration and can then be used
when creating relationships between features.

Examples:
  # Define a new relationship type
  fogit relationship define approves \
    --category workflow \
    --inverse approved-by \
    --description "Feature approval in review process"

  # Define with aliases
  fogit relationship define supersedes \
    --category structural \
    --inverse superseded-by \
    --alias replaces \
    --alias obsoletes

  # Define a bidirectional relationship
  fogit relationship define collaborates-with \
    --category informational \
    --bidirectional \
    --description "Features that collaborate"

  # Copy from existing type
  fogit relationship define custom-depends \
    --from depends-on \
    --description "Custom dependency with different semantics"`,
	Args: cobra.ExactArgs(1),
	RunE: runRelationshipDefine,
}

func init() {
	relationshipDefineCmd.Flags().StringVarP(&defineCategory, "category", "c", "", "Category name (required)")
	relationshipDefineCmd.Flags().StringVar(&defineInverse, "inverse", "", "Inverse relationship name")
	relationshipDefineCmd.Flags().StringVarP(&defineDescription, "description", "d", "", "Human-readable description")
	relationshipDefineCmd.Flags().BoolVar(&defineBidirectional, "bidirectional", false, "Make relationship symmetric")
	relationshipDefineCmd.Flags().StringArrayVar(&defineAliases, "alias", []string{}, "Add alias name (can be used multiple times)")
	relationshipDefineCmd.Flags().StringVar(&defineFrom, "from", "", "Copy from existing type")
	rootCmd.AddCommand(relationshipDefineCmd)
}

func runRelationshipDefine(cmd *cobra.Command, args []string) error {
	typeName := args[0]

	fogitDir, err := getFogitDir()
	if err != nil {
		return fmt.Errorf("failed to get .fogit directory: %w", err)
	}

	// Load config
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if type already exists
	if _, exists := cfg.Relationships.Types[typeName]; exists {
		return fmt.Errorf("relationship type '%s' already exists", typeName)
	}

	var newType fogit.RelationshipTypeConfig

	// Copy from existing type if specified
	if defineFrom != "" {
		fromType, exists := cfg.Relationships.Types[defineFrom]
		if !exists {
			return fmt.Errorf("source type '%s' not found", defineFrom)
		}
		newType = fromType
		// Clear inverse since it won't be valid for the new type
		newType.Inverse = ""
	}

	// Apply flags (override copied values)
	if defineCategory != "" {
		newType.Category = defineCategory
	}
	if defineInverse != "" {
		newType.Inverse = defineInverse
	}
	if defineDescription != "" {
		newType.Description = defineDescription
	}
	if cmd.Flags().Changed("bidirectional") {
		newType.Bidirectional = defineBidirectional
	}
	if len(defineAliases) > 0 {
		newType.Aliases = defineAliases
	}

	// Validate category is required
	if newType.Category == "" {
		return fmt.Errorf("--category is required")
	}

	// Validate category exists
	if _, exists := cfg.Relationships.Categories[newType.Category]; !exists {
		availableCategories := make([]string, 0, len(cfg.Relationships.Categories))
		for cat := range cfg.Relationships.Categories {
			availableCategories = append(availableCategories, cat)
		}
		return fmt.Errorf("category '%s' not found. Available categories: %s",
			newType.Category, strings.Join(availableCategories, ", "))
	}

	// Validate bidirectional + inverse conflict
	if newType.Bidirectional && newType.Inverse != "" {
		return fmt.Errorf("bidirectional types cannot have an inverse (they are their own inverse)")
	}

	// Add the new type
	if cfg.Relationships.Types == nil {
		cfg.Relationships.Types = make(map[string]fogit.RelationshipTypeConfig)
	}
	cfg.Relationships.Types[typeName] = newType

	// Create inverse type if specified
	if newType.Inverse != "" && !newType.Bidirectional {
		// Check if inverse already exists
		if existingInverse, exists := cfg.Relationships.Types[newType.Inverse]; exists {
			// Update existing inverse to point back to this type
			existingInverse.Inverse = typeName
			cfg.Relationships.Types[newType.Inverse] = existingInverse
		} else {
			// Create new inverse type
			inverseType := fogit.RelationshipTypeConfig{
				Category:      newType.Category,
				Inverse:       typeName,
				Bidirectional: false,
				Description:   fmt.Sprintf("Inverse of %s", typeName),
			}
			cfg.Relationships.Types[newType.Inverse] = inverseType
			fmt.Printf("Created inverse type: %s\n", newType.Inverse)
		}
	}

	// Validate config before saving
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Save config
	if err := config.Save(fogitDir, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Created relationship type: %s\n", typeName)
	fmt.Printf("  Category:      %s\n", newType.Category)
	if newType.Description != "" {
		fmt.Printf("  Description:   %s\n", newType.Description)
	}
	if newType.Inverse != "" {
		fmt.Printf("  Inverse:       %s\n", newType.Inverse)
	}
	if newType.Bidirectional {
		fmt.Printf("  Bidirectional: yes\n")
	}
	if len(newType.Aliases) > 0 {
		fmt.Printf("  Aliases:       %s\n", strings.Join(newType.Aliases, ", "))
	}

	return nil
}
