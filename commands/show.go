package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/printer"
)

var (
	showFormat        string
	showVersions      bool
	showRelationships bool
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show <feature>",
	Short: "Show detailed information about a feature",
	Long: `Display detailed information about a specific feature by name or ID.

Examples:
  # Show feature by name
  fogit show "User Authentication"

  # Show feature by ID
  fogit show 550e8400-e29b-41d4-a716-446655440000

  # Show with relationships
  fogit show "Login Page" --relationships

  # Show in JSON format
  fogit show "API Endpoint" --format json

  # Show in YAML format
  fogit show "Database Migration" --format yaml
`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)

	showCmd.Flags().StringVar(&showFormat, "format", "text", "Output format: text, json, yaml")
	showCmd.Flags().BoolVar(&showVersions, "versions", false, "Show version history")
	showCmd.Flags().BoolVar(&showRelationships, "relationships", false, "Show relationships")
}

func runShow(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	// Validate format
	if !printer.IsValidShowFormat(showFormat) {
		return fmt.Errorf("invalid format: must be one of text, json, yaml")
	}

	// Get command context
	cmdCtx, err := GetCommandContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Find feature using cross-branch discovery
	feature, err := FindFeatureCrossBranch(cmd.Context(), cmdCtx, identifier, "fogit show <id>")
	if err != nil {
		return err
	}

	// Format output
	switch showFormat {
	case "json":
		return printer.OutputFeatureJSON(os.Stdout, feature)
	case "yaml":
		return printer.OutputFeatureYAML(os.Stdout, feature)
	default:
		return printer.OutputFeatureText(os.Stdout, feature, showRelationships, showVersions)
	}
}
