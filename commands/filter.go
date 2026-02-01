package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/printer"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	filterFormat string
	filterSort   string
)

// filterCmd represents the filter command
var filterCmd = &cobra.Command{
	Use:   "filter <expression>",
	Short: "Advanced filtering with expressions",
	Long: `Filter features using advanced expressions with field matching, logical operators, and comparisons.

Expression syntax:
  Core field matching:    state:open, name:*auth*, tags:security
  Metadata fields:        metadata.priority:high, metadata.category:auth
  Shorthand aliases:      priority:high → metadata.priority:high
  Logical operators:      AND, OR, NOT
  Comparisons:            created:>2025-01-01, priority:>=medium
  Wildcards:              name:*auth* (contains "auth")
  Grouping:               (priority:high OR priority:critical) AND state:open

Shorthand aliases (metadata.*):
  priority, type, category, domain, team, epic, module

Comparison operators:
  :    equals (or contains for arrays/wildcards)
  :>   greater than
  :<   less than
  :>=  greater or equal
  :<=  less or equal

Examples:
  # Filter by priority and state
  fogit filter "priority:high AND state:open"

  # Filter with explicit metadata prefix
  fogit filter "metadata.priority:high AND state:open"

  # Filter by category
  fogit filter "metadata.category:authentication OR metadata.category:security"

  # Filter by date and team
  fogit filter "created:>2025-01-01 AND metadata.team:backend"

  # Complex expression with grouping
  fogit filter "(priority:high OR priority:critical) AND state:open AND category:security"

  # Wildcard matching
  fogit filter "name:*auth*"

  # Negation
  fogit filter "NOT state:closed"

  # Priority comparison
  fogit filter "priority:>=medium AND state:open"
`,
	Args: cobra.ExactArgs(1),
	RunE: runFilter,
}

func init() {
	rootCmd.AddCommand(filterCmd)

	// Output flags
	filterCmd.Flags().StringVar(&filterFormat, "format", "table", "Output format: table, json, csv")
	filterCmd.Flags().StringVar(&filterSort, "sort", "created", "Sort by field: name, priority, created, modified")
}

func runFilter(cmd *cobra.Command, args []string) error {
	expression := args[0]

	// Parse the filter expression
	expr, err := fogit.ParseFilterExpr(expression)
	if err != nil {
		return fmt.Errorf("invalid filter expression: %w", err)
	}

	// Validate format
	if !printer.IsValidFormat(filterFormat) {
		return fmt.Errorf("invalid format: must be one of table, json, csv")
	}

	// Validate sort field
	sortField := fogit.SortField(filterSort)
	if !sortField.IsValid() {
		return fmt.Errorf("invalid sort field: must be one of name, priority, created, modified")
	}

	// Get command context
	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	// List all features using cross-branch discovery (no filter - we'll apply expression filter)
	allFeatures, err := ListFeaturesCrossBranch(cmd.Context(), cmdCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to list features: %w", err)
	}

	// Apply expression filter
	var features []*fogit.Feature
	for _, f := range allFeatures {
		if expr.Matches(f) {
			features = append(features, f)
		}
	}

	// Sort features
	sortFilter := &fogit.Filter{
		SortBy:    sortField,
		SortOrder: fogit.SortDescending,
	}
	fogit.SortFeatures(features, sortFilter)

	// Check if empty
	if len(features) == 0 {
		fmt.Println("No features found matching expression")
		return nil
	}

	// Format output
	switch filterFormat {
	case "json":
		return printer.OutputJSON(os.Stdout, features)
	case "csv":
		return printer.OutputCSV(os.Stdout, features)
	default:
		return printer.OutputTable(os.Stdout, features)
	}
}
