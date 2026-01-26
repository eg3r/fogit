package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/printer"
	"github.com/eg3r/fogit/pkg/fogit"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show FoGit repository status",
	Long: `Show FoGit repository status including feature counts by state,
recent changes, and relationship statistics.

Per spec 08-interface.md: Shows current feature counts by state,
recent changes, and relationship statistics.

Examples:
  # Show repository status
  fogit status

  # Show status with JSON output
  fogit status --format json
`,
	RunE: runStatus,
}

var statusFormat string

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringVar(&statusFormat, "format", "text", "Output format: text, json")
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Get command context
	cmdCtx, err := GetCommandContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// List all features using cross-branch discovery in branch-per-feature mode
	featuresList, err := ListFeaturesCrossBranch(cmd.Context(), cmdCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to list features: %w", err)
	}

	// Build status report using service
	opts := features.DefaultStatusOptions()
	report := features.BuildStatusReport(featuresList, cmdCtx.Config, opts)

	// Try to get current branch using GitIntegration
	if cmdCtx.Git != nil && cmdCtx.Git.IsAvailable() {
		branch, err := cmdCtx.Git.GetCurrentBranch()
		if err == nil {
			report.CurrentBranch = branch
			// Find features on this branch
			report.FeaturesOnBranch = features.FindFeaturesOnBranch(featuresList, branch)
		}
	}

	// Output based on format
	textFn := func(w io.Writer) error {
		return outputStatusText(w, report, featuresList)
	}
	return printer.OutputFormatted(os.Stdout, statusFormat, report, textFn)
}

func outputStatusText(w io.Writer, report *features.StatusReport, featuresList []*fogit.Feature) error {
	fmt.Fprintf(w, "FoGit Repository Status\n")
	fmt.Fprintf(w, "========================\n\n")

	// Repository info
	repoName := report.Repository.Name
	if repoName == "" {
		repoName = "(unnamed)"
	}
	fmt.Fprintf(w, "Repository: %s\n", repoName)
	if report.CurrentBranch != "" {
		fmt.Fprintf(w, "Branch:     %s\n", report.CurrentBranch)
	}
	fmt.Fprintf(w, "\n")

	// Feature counts
	fmt.Fprintf(w, "Features: %d total\n", report.Repository.TotalFeatures)
	fmt.Fprintf(w, "  Open:        %d\n", report.FeatureCounts.Open)
	fmt.Fprintf(w, "  In Progress: %d\n", report.FeatureCounts.InProgress)
	fmt.Fprintf(w, "  Closed:      %d\n", report.FeatureCounts.Closed)
	fmt.Fprintf(w, "\n")

	// Features on current branch (if any)
	if len(report.FeaturesOnBranch) > 0 {
		fmt.Fprintf(w, "Features on branch '%s':\n", report.CurrentBranch)
		for _, name := range report.FeaturesOnBranch {
			// Find the feature to get its state
			for _, f := range featuresList {
				if f.Name == name {
					fmt.Fprintf(w, "  - %s (%s)\n", name, f.DeriveState())
					break
				}
			}
		}
		fmt.Fprintf(w, "\n")
	}

	// Recent changes
	if len(report.RecentChanges) > 0 {
		fmt.Fprintf(w, "Recent Changes (last 7 days):\n")
		for _, change := range report.RecentChanges {
			ago := features.FormatTimeAgo(change.ModifiedAt)
			fmt.Fprintf(w, "  - %s (%s) - %s\n", change.FeatureName, change.State, ago)
		}
		fmt.Fprintf(w, "\n")
	}

	// Relationship statistics
	fmt.Fprintf(w, "Relationships: %d total\n", report.RelationshipStats.TotalRelationships)
	if len(report.RelationshipStats.ByCategory) > 0 {
		fmt.Fprintf(w, "  By Category:\n")
		for cat, count := range report.RelationshipStats.ByCategory {
			fmt.Fprintf(w, "    %-15s %d\n", cat+":", count)
		}
	}

	return nil
}
