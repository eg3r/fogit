package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	catDefineDescription   string
	catDefineAllowCycles   bool
	catDefineNoCycles      bool
	catDefineDetection     string
	catDefineIncludeImpact bool
)

var relationshipCategoryDefineCmd = &cobra.Command{
	Use:   "define <name>",
	Short: "Define a new relationship category",
	Long: `Define a new relationship category.

Categories group relationship types with common validation rules,
such as cycle detection behavior and impact analysis inclusion.

Examples:
  # Define a strict category that prevents cycles
  fogit relationship category define security \
    --description "Security-related relationships" \
    --no-cycles \
    --detection strict

  # Define a category that allows cycles
  fogit relationship category define feedback \
    --description "Feedback loops between features" \
    --allow-cycles

  # Define with all options
  fogit relationship category define audit \
    --description "Audit and compliance relationships" \
    --no-cycles \
    --detection strict \
    --include-in-impact`,
	Args: cobra.ExactArgs(1),
	RunE: runRelationshipCategoryDefine,
}

func init() {
	relationshipCategoryDefineCmd.Flags().StringVarP(&catDefineDescription, "description", "d", "", "Human-readable description")
	relationshipCategoryDefineCmd.Flags().BoolVar(&catDefineAllowCycles, "allow-cycles", false, "Allow cycles in this category")
	relationshipCategoryDefineCmd.Flags().BoolVar(&catDefineNoCycles, "no-cycles", false, "Prevent cycles in this category (default)")
	relationshipCategoryDefineCmd.Flags().StringVar(&catDefineDetection, "detection", "strict", "Cycle detection mode: strict, warn, none")
	relationshipCategoryDefineCmd.Flags().BoolVar(&catDefineIncludeImpact, "include-in-impact", true, "Include in impact analysis")

	// Add as subcommand to relationship category
	relationshipCategoriesCmd.AddCommand(relationshipCategoryDefineCmd)
}

func runRelationshipCategoryDefine(cmd *cobra.Command, args []string) error {
	categoryName := args[0]

	fogitDir, err := getFogitDir()
	if err != nil {
		return fmt.Errorf("failed to get .fogit directory: %w", err)
	}

	// Load config
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if category already exists
	if _, exists := cfg.Relationships.Categories[categoryName]; exists {
		return fmt.Errorf("category '%s' already exists", categoryName)
	}

	// Validate detection mode
	switch catDefineDetection {
	case "strict", "warn", "none":
		// Valid
	default:
		return fmt.Errorf("invalid detection mode '%s': must be strict, warn, or none", catDefineDetection)
	}

	// Handle allow-cycles / no-cycles conflict
	allowCycles := false
	if catDefineAllowCycles && catDefineNoCycles {
		return fmt.Errorf("cannot specify both --allow-cycles and --no-cycles")
	}
	if catDefineAllowCycles {
		allowCycles = true
		// If allowing cycles, detection should be none
		if catDefineDetection != "none" && cmd.Flags().Changed("detection") {
			return fmt.Errorf("when --allow-cycles is set, --detection should be 'none'")
		}
		catDefineDetection = "none"
	}

	// Create new category
	newCategory := fogit.RelationshipCategory{
		Description:     catDefineDescription,
		AllowCycles:     allowCycles,
		CycleDetection:  catDefineDetection,
		IncludeInImpact: catDefineIncludeImpact,
	}

	// Add the new category
	if cfg.Relationships.Categories == nil {
		cfg.Relationships.Categories = make(map[string]fogit.RelationshipCategory)
	}
	cfg.Relationships.Categories[categoryName] = newCategory

	// Validate config before saving
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Save config
	if err := config.Save(fogitDir, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Created relationship category: %s\n", categoryName)
	if newCategory.Description != "" {
		fmt.Printf("  Description:       %s\n", newCategory.Description)
	}
	fmt.Printf("  Allow Cycles:      %t\n", newCategory.AllowCycles)
	fmt.Printf("  Cycle Detection:   %s\n", newCategory.CycleDetection)
	fmt.Printf("  Include in Impact: %t\n", newCategory.IncludeInImpact)

	return nil
}
