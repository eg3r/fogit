package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/printer"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show repository statistics",
	Long: `Display statistics about features in the repository.

Shows counts by state, priority, type, and other useful metrics.

Examples:
  # Show all statistics
  fogit stats

  # Show statistics with details
  fogit stats --details
`,
	RunE: runStats,
}

var statsDetails bool

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().BoolVar(&statsDetails, "details", false, "Show detailed breakdown")
}

func runStats(cmd *cobra.Command, args []string) error {
	// Get command context
	ctx, err := GetCommandContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// List all features using cross-branch discovery
	featuresList, err := ListFeaturesCrossBranch(cmd.Context(), ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list features: %w", err)
	}

	if len(featuresList) == 0 {
		fmt.Println("No features in repository")
		return nil
	}

	// Calculate statistics
	stats := features.CalculateStats(featuresList, ctx.Config)

	// Display statistics
	return printer.OutputStats(os.Stdout, stats, statsDetails)
}
