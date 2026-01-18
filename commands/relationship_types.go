package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	typesCategory string
	typesVerbose  bool
)

var relationshipTypesCmd = &cobra.Command{
	Use:     "types",
	Short:   "List available relationship types",
	Long:    `Display all relationship types defined in the configuration, optionally filtered by category.`,
	Aliases: []string{"type"},
	RunE:    runRelationshipTypes,
}

func init() {
	relationshipTypesCmd.Flags().StringVarP(&typesCategory, "category", "c", "", "Filter by category (structural, informational, workflow, compliance)")
	relationshipTypesCmd.Flags().BoolVarP(&typesVerbose, "verbose", "v", false, "Show detailed information (inverse, aliases, bidirectional)")
	rootCmd.AddCommand(relationshipTypesCmd)
}

func runRelationshipTypes(cmd *cobra.Command, args []string) error {
	fogitDir, err := getFogitDir()
	if err != nil {
		return fmt.Errorf("failed to get .fogit directory: %w", err)
	}

	// Load config
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get all types
	types := make([]string, 0, len(cfg.Relationships.Types))
	for typeName := range cfg.Relationships.Types {
		types = append(types, typeName)
	}
	sort.Strings(types)

	// Filter by category if specified
	if typesCategory != "" {
		filtered := make([]string, 0)
		for _, typeName := range types {
			typeConfig := cfg.Relationships.Types[typeName]
			if strings.EqualFold(typeConfig.Category, typesCategory) {
				filtered = append(filtered, typeName)
			}
		}
		types = filtered
	}

	if len(types) == 0 {
		if typesCategory != "" {
			fmt.Printf("No relationship types found in category '%s'\n", typesCategory)
		} else {
			fmt.Println("No relationship types defined")
		}
		return nil
	}

	// Display header
	if typesCategory != "" {
		fmt.Printf("Relationship Types in '%s' category:\n\n", typesCategory)
	} else {
		fmt.Println("Relationship Types:")
		fmt.Println()
	}

	// Display types
	if typesVerbose {
		displayVerboseTypes(types, cfg)
	} else {
		displayCompactTypes(types, cfg)
	}

	return nil
}

func displayCompactTypes(types []string, cfg *fogit.Config) {
	// Group by category
	byCategory := make(map[string][]string)
	for _, typeName := range types {
		typeConfig := cfg.Relationships.Types[typeName]
		cat := typeConfig.Category
		if cat == "" {
			cat = "uncategorized"
		}
		byCategory[cat] = append(byCategory[cat], typeName)
	}

	// Get sorted categories
	categories := make([]string, 0, len(byCategory))
	for cat := range byCategory {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	// Display by category
	for i, cat := range categories {
		if i > 0 {
			fmt.Println()
		}

		categoryName := cat
		if cfg.Relationships.Categories[cat].Description != "" {
			categoryName += " (" + cfg.Relationships.Categories[cat].Description + ")"
		}
		fmt.Printf("%s:\n", categoryName)

		typeList := byCategory[cat]
		sort.Strings(typeList)

		for _, typeName := range typeList {
			typeConfig := cfg.Relationships.Types[typeName]
			desc := typeConfig.Description
			if desc == "" {
				desc = "No description"
			}

			bidirectional := ""
			if typeConfig.Bidirectional {
				bidirectional = " [bidirectional]"
			}

			fmt.Printf("  %-20s %s%s\n", typeName, desc, bidirectional)
		}
	}
}

func displayVerboseTypes(types []string, cfg *fogit.Config) {
	for i, typeName := range types {
		if i > 0 {
			fmt.Println()
		}

		typeConfig := cfg.Relationships.Types[typeName]

		fmt.Printf("Type: %s\n", typeName)

		if typeConfig.Description != "" {
			fmt.Printf("  Description:    %s\n", typeConfig.Description)
		}

		fmt.Printf("  Category:       %s\n", typeConfig.Category)

		if typeConfig.Inverse != "" {
			fmt.Printf("  Inverse:        %s\n", typeConfig.Inverse)
		} else {
			fmt.Printf("  Inverse:        (none)\n")
		}

		if typeConfig.Bidirectional {
			fmt.Printf("  Bidirectional:  yes\n")
		}

		if len(typeConfig.Aliases) > 0 {
			fmt.Printf("  Aliases:        %s\n", strings.Join(typeConfig.Aliases, ", "))
		}
	}
}
