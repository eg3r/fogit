package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/printer"
	"github.com/eg3r/fogit/pkg/fogit"
)

var filesCmd = &cobra.Command{
	Use:   "files [feature-name|file-path]",
	Short: "Show files associated with features",
	Long: `Show files associated with a feature, or features associated with a file.

If a feature name is provided, shows all files in that feature.
If a file path is provided, shows all features that reference that file.
If no argument is provided, shows a summary of all file associations.

Examples:
  # Show files in a specific feature
  fogit files "User Authentication"

  # Show features that modified a specific file
  fogit files src/auth/login.go

  # Show all file associations
  fogit files

  # Show with state filter
  fogit files --state open
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFiles,
}

var filesState string

func init() {
	rootCmd.AddCommand(filesCmd)
	filesCmd.Flags().StringVar(&filesState, "state", "", "Filter by feature state")
}

func runFiles(cmd *cobra.Command, args []string) error {
	// Get command context
	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	// Build filter
	filter := &fogit.Filter{
		State: fogit.State(filesState),
	}

	if validateErr := filter.Validate(); validateErr != nil {
		return validateErr
	}

	// List all features using cross-branch discovery
	allFeatures, listErr := ListFeaturesCrossBranch(cmd.Context(), cmdCtx, filter)
	if listErr != nil {
		return fmt.Errorf("failed to list features: %w", listErr)
	}

	if len(args) == 0 {
		// Show summary of all file associations
		return printer.OutputFilesSummary(os.Stdout, allFeatures)
	}

	arg := args[0]

	// Determine if arg is a file path or feature name
	if strings.Contains(arg, "/") || strings.Contains(arg, "\\") || strings.Contains(arg, ".") {
		// Looks like a file path
		matchingFeatures, findErr := features.FindForFile(cmd.Context(), cmdCtx.Repo, arg)
		if findErr != nil {
			return findErr
		}
		return printer.OutputFeaturesForFile(os.Stdout, arg, matchingFeatures)
	}

	// Looks like a feature name
	result, findErr := features.Find(cmd.Context(), cmdCtx.Repo, arg, &fogit.Config{})
	if findErr != nil {
		return findErr
	}
	return printer.OutputFilesForFeature(os.Stdout, result.Feature)
}
