package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
)

var (
	categoryUpdateName          string
	categoryUpdateKeepOldAlias  bool
	categoryUpdateDescription   string
	categoryUpdateAllowCycles   bool
	categoryUpdateNoAllowCycles bool
	categoryUpdateDetection     string
	categoryUpdateIncludeImpact bool
	categoryUpdateExcludeImpact bool
)

var relationshipCategoryUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update or rename a relationship category",
	Long: `Update or rename an existing relationship category.

Use --name to rename the category (updates all type references).
Other flags modify properties without renaming.

Examples:
  # Rename a category
  fogit relationship category update dependency --name dependencies

  # Keep old name as alias during rename
  fogit relationship category update dependency --name dependencies --keep-old-as-alias

  # Update properties
  fogit relationship category update dependency --description "Updated" --allow-cycles

  # Change cycle detection mode
  fogit relationship category update structural --detection strict`,
	Args: cobra.ExactArgs(1),
	RunE: runRelationshipCategoryUpdate,
}

func init() {
	relationshipCategoryUpdateCmd.Flags().StringVar(&categoryUpdateName, "name", "", "Rename the category")
	relationshipCategoryUpdateCmd.Flags().BoolVar(&categoryUpdateKeepOldAlias, "keep-old-as-alias", false, "Keep old name as alias when renaming")
	relationshipCategoryUpdateCmd.Flags().StringVar(&categoryUpdateDescription, "description", "", "Update description")
	relationshipCategoryUpdateCmd.Flags().BoolVar(&categoryUpdateAllowCycles, "allow-cycles", false, "Allow cycles in this category")
	relationshipCategoryUpdateCmd.Flags().BoolVar(&categoryUpdateNoAllowCycles, "no-allow-cycles", false, "Disallow cycles in this category")
	relationshipCategoryUpdateCmd.Flags().StringVar(&categoryUpdateDetection, "detection", "", "Cycle detection mode: strict, warn, none")
	relationshipCategoryUpdateCmd.Flags().BoolVar(&categoryUpdateIncludeImpact, "include-in-impact", false, "Include in impact analysis")
	relationshipCategoryUpdateCmd.Flags().BoolVar(&categoryUpdateExcludeImpact, "exclude-from-impact", false, "Exclude from impact analysis")

	relationshipCategoriesCmd.AddCommand(relationshipCategoryUpdateCmd)
}

func runRelationshipCategoryUpdate(cmd *cobra.Command, args []string) error {
	categoryName := args[0]

	fogitDir, err := getFogitDir()
	if err != nil {
		return fmt.Errorf("failed to get .fogit directory: %w", err)
	}

	// Validate conflicting flags
	if categoryUpdateAllowCycles && categoryUpdateNoAllowCycles {
		return fmt.Errorf("cannot use both --allow-cycles and --no-allow-cycles")
	}
	if categoryUpdateIncludeImpact && categoryUpdateExcludeImpact {
		return fmt.Errorf("cannot use both --include-in-impact and --exclude-from-impact")
	}

	// Build options
	opts := features.RelationshipCategoryUpdateOptions{
		NewName:        categoryUpdateName,
		KeepOldAsAlias: categoryUpdateKeepOldAlias,
		Description:    categoryUpdateDescription,
		SetDescription: cmd.Flags().Changed("description"),
		CycleDetection: categoryUpdateDetection,
	}

	// Handle allow cycles flags
	if categoryUpdateAllowCycles {
		b := true
		opts.AllowCycles = &b
	}
	if categoryUpdateNoAllowCycles {
		b := false
		opts.AllowCycles = &b
	}

	// Handle impact flags
	if categoryUpdateIncludeImpact {
		b := true
		opts.IncludeInImpact = &b
	}
	if categoryUpdateExcludeImpact {
		b := false
		opts.IncludeInImpact = &b
	}

	// Execute update
	result, err := features.UpdateRelationshipCategory(fogitDir, categoryName, opts)
	if err != nil {
		return err
	}

	// Output results
	if result.Renamed {
		fmt.Printf("Renamed category '%s' to '%s'\n", result.OldName, result.NewName)
		if result.TypesUpdated > 0 {
			fmt.Printf("Updated %d relationship types to use new category name\n", result.TypesUpdated)
		}
		if result.KeptOldAsAlias {
			fmt.Printf("Added '%s' as alias for '%s'\n", result.OldName, result.NewName)
		}
	} else {
		fmt.Printf("Updated category '%s'\n", result.NewName)
	}

	return nil
}
