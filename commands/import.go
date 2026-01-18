package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/exchange"
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import features from a file",
	Long: `Import features from a JSON or YAML file.

The import file should match the format produced by 'fogit export'.

Conflict Handling:
  Default:     Error on conflicts (abort import, no changes made)
  --merge:     Skip existing features, import only new ones
  --overwrite: Replace existing features with imported data

The import validates that all relationship targets exist (either in the
repository or in the import file) before making any changes.

Examples:
  fogit import features.json             # Import with error on conflicts
  fogit import features.json --merge     # Skip existing, import new only
  fogit import features.yaml --overwrite # Replace existing features
  fogit import data.json --dry-run       # Preview changes without applying`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

var (
	importMerge     bool
	importOverwrite bool
	importDryRun    bool
)

func init() {
	importCmd.Flags().BoolVar(&importMerge, "merge", false, "Skip existing features, import only new ones")
	importCmd.Flags().BoolVar(&importOverwrite, "overwrite", false, "Replace existing features with imported data")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Preview changes without applying them")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Validate flags
	if importMerge && importOverwrite {
		return fmt.Errorf("cannot use both --merge and --overwrite flags")
	}

	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	// Read and parse import file
	importData, err := exchange.ReadImportFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Apply timeout for import operation
	ctx, cancel := WithImportTimeout(cmd.Context())
	defer cancel()

	// Execute import via service
	opts := exchange.ImportOptions{
		Merge:     importMerge,
		Overwrite: importOverwrite,
		DryRun:    importDryRun,
	}

	result, err := exchange.Import(ctx, cmdCtx.Repo, importData, opts)
	if err != nil {
		return err
	}

	// Print actions for dry-run
	if importDryRun {
		for _, action := range result.Actions {
			switch action.Type {
			case "CREATE":
				fmt.Printf("[CREATE] %s\n", action.FeatureName)
			case "UPDATE":
				fmt.Printf("[UPDATE] %s\n", action.FeatureName)
			case "SKIP":
				fmt.Printf("[SKIP] %s (%s)\n", action.FeatureName, action.Reason)
			}
		}
	}

	// Report results
	if importDryRun {
		fmt.Println("\n--- Dry Run Results ---")
	} else {
		fmt.Println("\n--- Import Results ---")
	}

	fmt.Printf("Created: %d\n", result.Created)
	fmt.Printf("Updated: %d\n", result.Updated)
	fmt.Printf("Skipped: %d\n", result.Skipped)

	if len(result.Errors) > 0 {
		fmt.Printf("Errors:  %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
		if !importMerge && !importOverwrite {
			return fmt.Errorf("import failed: %d conflicts detected (use --merge or --overwrite to handle)", len(result.Errors))
		}
	}

	return nil
}
