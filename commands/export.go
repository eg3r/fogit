package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/common"
	"github.com/eg3r/fogit/internal/exchange"
	"github.com/eg3r/fogit/pkg/fogit"
)

var exportCmd = &cobra.Command{
	Use:   "export <format>",
	Short: "Export features to various formats",
	Long: `Export all features to JSON, YAML, or CSV format.

The exported data includes:
- Feature metadata and descriptions
- Version history
- Relationships with target existence marking
- Custom metadata fields

Formats:
  json - Full export with all fields (recommended for import)
  yaml - Full export in YAML format
  csv  - Simplified tabular format (features only, no relationships)

Examples:
  fogit export json                     # Export to stdout
  fogit export json > features.json     # Export to file
  fogit export yaml --output data.yaml  # Export to file
  fogit export csv --state open         # Export only open features
  fogit export json --tag security      # Export features with tag`,
	Args: cobra.ExactArgs(1),
	RunE: runExport,
}

var (
	exportOutput   string
	exportState    string
	exportType     string
	exportCategory string
	exportTags     []string
	exportPretty   bool
)

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file (default: stdout)")
	exportCmd.Flags().StringVar(&exportState, "state", "", "Filter by state (open, in-progress, closed)")
	exportCmd.Flags().StringVar(&exportType, "type", "", "Filter by type")
	exportCmd.Flags().StringVar(&exportCategory, "category", "", "Filter by category")
	exportCmd.Flags().StringSliceVar(&exportTags, "tag", nil, "Filter by tag (can be repeated)")
	exportCmd.Flags().BoolVar(&exportPretty, "pretty", true, "Pretty-print output (default: true)")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	format := args[0]

	// Validate format
	if format != "json" && format != "yaml" && format != "csv" {
		return fmt.Errorf("unsupported format: %s (use json, yaml, or csv)", format)
	}

	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	// Build filter
	filter := &fogit.Filter{
		State:    fogit.State(exportState),
		Type:     exportType,
		Category: exportCategory,
		Tags:     exportTags,
	}

	// Apply timeout for export operation
	ctx, cancel := WithExportTimeout(cmd.Context())
	defer cancel()

	// Get all features using cross-branch discovery
	featuresList, err := ListFeaturesCrossBranch(ctx, cmdCtx, filter)
	if err != nil {
		return fmt.Errorf("failed to list features: %w", err)
	}

	// Export using service with pre-loaded features
	opts := exchange.ExportOptions{
		Format:   format,
		Filter:   filter,
		FogitDir: cmdCtx.FogitDir,
		Pretty:   exportPretty,
	}

	exportData, err := exchange.ExportWithFeatures(featuresList, opts)
	if err != nil {
		return err
	}

	// Write to file or stdout
	if exportOutput != "" {
		// Validate output path to prevent path traversal attacks
		if err := common.ValidateOutputPath(exportOutput); err != nil {
			return err
		}

		// Use atomic write to prevent partial/corrupted files on failure
		return common.AtomicWriteFile(exportOutput, func(f *os.File) error {
			return writeExport(f, format, exportData)
		})
	}

	// Write to stdout (no atomic write needed)
	return writeExport(os.Stdout, format, exportData)
}

// writeExport writes the export data in the specified format
func writeExport(output *os.File, format string, exportData *exchange.ExportData) error {
	switch format {
	case "json":
		return exchange.WriteJSON(output, exportData, exportPretty)
	case "yaml":
		return exchange.WriteYAML(output, exportData)
	case "csv":
		return exchange.WriteCSV(output, exportData.Features)
	}
	return nil
}
