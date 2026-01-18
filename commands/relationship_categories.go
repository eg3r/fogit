package commands

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/pkg/fogit"
)

var categoriesVerbose bool

var relationshipCategoriesCmd = &cobra.Command{
	Use:     "categories",
	Short:   "List relationship categories",
	Long:    `Display all relationship categories defined in the configuration with their settings.`,
	Aliases: []string{"category", "cats"},
	RunE:    runRelationshipCategories,
}

func init() {
	relationshipCategoriesCmd.Flags().BoolVarP(&categoriesVerbose, "verbose", "v", false, "Show detailed settings (cycle detection, history tracking, etc.)")
	rootCmd.AddCommand(relationshipCategoriesCmd)
}

func runRelationshipCategories(cmd *cobra.Command, args []string) error {
	fogitDir, err := getFogitDir()
	if err != nil {
		return fmt.Errorf("failed to get .fogit directory: %w", err)
	}

	// Load config
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get all categories
	categories := make([]string, 0, len(cfg.Relationships.Categories))
	for catName := range cfg.Relationships.Categories {
		categories = append(categories, catName)
	}
	sort.Strings(categories)

	if len(categories) == 0 {
		fmt.Println("No relationship categories defined")
		return nil
	}

	// Display header
	fmt.Println("Relationship Categories:")
	fmt.Println()

	// Display categories
	if categoriesVerbose {
		displayVerboseCategories(categories, cfg)
	} else {
		displayCompactCategories(categories, cfg)
	}

	return nil
}

func displayCompactCategories(categories []string, cfg *fogit.Config) {
	for _, catName := range categories {
		cat := cfg.Relationships.Categories[catName]

		desc := cat.Description
		if desc == "" {
			desc = "No description"
		}

		// Count types in this category
		typeCount := 0
		for _, typeConfig := range cfg.Relationships.Types {
			if typeConfig.Category == catName {
				typeCount++
			}
		}

		fmt.Printf("%-15s %s (%d types)\n", catName, desc, typeCount)
	}
}

func displayVerboseCategories(categories []string, cfg *fogit.Config) {
	for i, catName := range categories {
		if i > 0 {
			fmt.Println()
		}

		cat := cfg.Relationships.Categories[catName]

		fmt.Printf("Category: %s\n", catName)

		if cat.Description != "" {
			fmt.Printf("  Description:       %s\n", cat.Description)
		}

		// Count types in this category
		typeCount := 0
		for _, typeConfig := range cfg.Relationships.Types {
			if typeConfig.Category == catName {
				typeCount++
			}
		}
		fmt.Printf("  Types:             %d relationship types\n", typeCount)

		fmt.Printf("  Allow Cycles:      %t\n", cat.AllowCycles)
		fmt.Printf("  Cycle Detection:   %s\n", cat.CycleDetection)
		fmt.Printf("  Include in Impact: %t\n", cat.IncludeInImpact)
	}
}
