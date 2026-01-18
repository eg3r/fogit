package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/printer"
)

var diffCmd = &cobra.Command{
	Use:   "diff <feature> [version1] [version2]",
	Short: "Compare feature versions",
	Long: `Compare two versions of a feature to see what changed.

If no versions are specified, compares the current version with the previous one.
If only one version is specified, compares that version with the current version.

The command shows differences in:
- Timestamps (created, modified, closed)
- Branch name
- Authors
- Notes

Examples:
  fogit diff "User Authentication"           # Compare current vs previous version
  fogit diff "User Auth" 1 2                 # Compare version 1 and version 2
  fogit diff "API Core" 1.0.0 2.0.0          # Compare semantic versions
  fogit diff "Feature" --format json         # Output as JSON
  fogit diff "Feature" --format yaml         # Output as YAML`,
	Args: cobra.RangeArgs(1, 3),
	RunE: runDiff,
}

var (
	diffFormat string
)

func init() {
	diffCmd.Flags().StringVar(&diffFormat, "format", "text", "Output format (text, json, yaml)")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	nameOrID := args[0]

	// Validate format
	if diffFormat != "text" && diffFormat != "json" && diffFormat != "yaml" {
		return fmt.Errorf("unsupported format: %s (use text, json, or yaml)", diffFormat)
	}

	// Get command context
	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	// Find feature using the consolidated helper
	feature, err := FindFeatureWithSuggestions(cmd.Context(), cmdCtx.Repo, nameOrID, cmdCtx.Config, "fogit diff <id>")
	if err != nil {
		return err
	}

	// Check if there are any versions
	if len(feature.Versions) == 0 {
		return fmt.Errorf("feature %q has no versions", feature.Name)
	}

	if len(feature.Versions) < 2 && len(args) < 3 {
		return fmt.Errorf("feature %q has only one version; specify two versions to compare", feature.Name)
	}

	// Get sorted version keys using the consolidated method
	versionKeys := feature.GetSortedVersionKeys()

	// Determine which versions to compare
	var version1, version2 string

	switch len(args) {
	case 1:
		// Compare current with previous
		if len(versionKeys) < 2 {
			return fmt.Errorf("feature has only one version; cannot compare with previous")
		}
		version1 = versionKeys[len(versionKeys)-2] // previous
		version2 = versionKeys[len(versionKeys)-1] // current
	case 2:
		// Compare specified version with current
		version1 = args[1]
		version2 = feature.GetCurrentVersionKey()
		if version1 == version2 {
			return fmt.Errorf("cannot compare version %s with itself", version1)
		}
	case 3:
		// Compare two specified versions
		version1 = args[1]
		version2 = args[2]
		if version1 == version2 {
			return fmt.Errorf("cannot compare version %s with itself", version1)
		}
	}

	// Validate versions exist
	v1, ok := feature.Versions[version1]
	if !ok {
		return fmt.Errorf("version %q not found; available versions: %s", version1, strings.Join(versionKeys, ", "))
	}
	v2, ok := feature.Versions[version2]
	if !ok {
		return fmt.Errorf("version %q not found; available versions: %s", version2, strings.Join(versionKeys, ", "))
	}

	// Calculate differences
	diff := features.CalculateVersionDiff(feature, version1, v1, version2, v2)

	// Output based on format
	textFn := func(w io.Writer) error {
		return outputDiffText(w, diff)
	}
	return printer.OutputFormatted(os.Stdout, diffFormat, diff, textFn)
}

func outputDiffText(w io.Writer, diff *features.VersionDiff) error {
	fmt.Fprintf(w, "Diff: %s\n", diff.FeatureName)
	fmt.Fprintf(w, "ID: %s\n", diff.FeatureID)
	fmt.Fprintf(w, "Comparing version %s -> %s\n", diff.Version1, diff.Version2)
	fmt.Fprintln(w, strings.Repeat("─", 60))

	if !diff.HasDifferences {
		fmt.Fprintln(w, "\nNo differences found between versions.")
		return nil
	}

	fmt.Fprintf(w, "\nChanges (%d):\n\n", len(diff.Changes))

	for _, change := range diff.Changes {
		symbol := features.GetChangeSymbol(change.ChangeType)
		fmt.Fprintf(w, "%s %s:\n", symbol, change.Field)

		switch change.ChangeType {
		case "added":
			fmt.Fprintf(w, "    + %s\n", change.NewValue)
		case "removed":
			fmt.Fprintf(w, "    - %s\n", change.OldValue)
		default:
			fmt.Fprintf(w, "    - %s\n", change.OldValue)
			fmt.Fprintf(w, "    + %s\n", change.NewValue)
		}
		fmt.Fprintln(w)
	}

	return nil
}
