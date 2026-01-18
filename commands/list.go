package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/printer"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	listState       string
	listPriority    string
	listType        string
	listCategory    string
	listDomain      string
	listTeam        string
	listEpic        string
	listParent      string
	listTags        []string
	listContributor string
	listFormat      string
	listSort        string
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List features with optional filters",
	Long: `List all features in the repository with optional filtering and sorting.

Examples:
  # List all features
  fogit list

  # Filter by state and priority
  fogit list --state open --priority high

  # Filter by organization
  fogit list --category authentication --domain backend

  # Filter by tags (multiple tags = AND logic)
  fogit list --tag security --tag auth

  # Filter by contributor
  fogit list --contributor alice@example.com

  # Sort and format
  fogit list --sort priority --format json

  # Multiple filters
  fogit list --state open --team security-team --epic user-management
`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Filter flags
	listCmd.Flags().StringVar(&listState, "state", "", "Filter by state (open, in-progress, closed)")
	listCmd.Flags().StringVar(&listPriority, "priority", "", "Filter by priority (low, medium, high, critical)")
	listCmd.Flags().StringVar(&listType, "type", "", "Filter by type")
	listCmd.Flags().StringVar(&listCategory, "category", "", "Filter by category")
	listCmd.Flags().StringVar(&listDomain, "domain", "", "Filter by domain")
	listCmd.Flags().StringVar(&listTeam, "team", "", "Filter by team")
	listCmd.Flags().StringVar(&listEpic, "epic", "", "Filter by epic")
	listCmd.Flags().StringVar(&listParent, "parent", "", "Show children of feature")
	listCmd.Flags().StringSliceVar(&listTags, "tag", []string{}, "Filter by tag (can be used multiple times, AND logic)")
	listCmd.Flags().StringVar(&listContributor, "contributor", "", "Filter by contributor email")

	// Output flags
	listCmd.Flags().StringVar(&listFormat, "format", "table", "Output format: table, json, csv")
	listCmd.Flags().StringVar(&listSort, "sort", "created", "Sort by field: name, priority, created, modified")
}

func runList(cmd *cobra.Command, args []string) error {
	// Build filter from flags
	filter := &fogit.Filter{
		State:       fogit.State(listState),
		Priority:    fogit.Priority(listPriority),
		Type:        listType,
		Category:    listCategory,
		Domain:      listDomain,
		Team:        listTeam,
		Epic:        listEpic,
		Parent:      listParent,
		Tags:        listTags,
		Contributor: listContributor,
		SortBy:      fogit.SortField(listSort),
	}

	// Validate filter
	if err := filter.Validate(); err != nil {
		return err
	}

	// Validate format
	if !printer.IsValidFormat(listFormat) {
		return fmt.Errorf("invalid format: must be one of table, json, csv")
	}

	// Get command context
	cmdCtx, err := GetCommandContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Apply timeout to prevent hanging on slow filesystems
	ctx, cancel := WithListTimeout(cmd.Context())
	defer cancel()

	// List features
	features, err := cmdCtx.Repo.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list features: %w", err)
	}

	// Sort features
	fogit.SortFeatures(features, filter)

	// Check if empty
	if len(features) == 0 {
		if printer.HasActiveFilters(filter) {
			fmt.Println("No features found matching filters")
		} else {
			fmt.Println("No features found")
		}
		return nil
	}

	// Format output
	switch listFormat {
	case "json":
		return printer.OutputJSON(os.Stdout, features)
	case "csv":
		return printer.OutputCSV(os.Stdout, features)
	default:
		return printer.OutputTable(os.Stdout, features)
	}
}
