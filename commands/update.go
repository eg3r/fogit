package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/common"
	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/printer"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	updateState       string
	updatePriority    string
	updateDescription string
	updateType        string
	updateCategory    string
	updateDomain      string
	updateTeam        string
	updateEpic        string
	updateModule      string
	updateName        string
	updateMetadata    []string
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update <feature>",
	Short: "Update feature properties",
	Long: `Update properties of an existing feature by name or ID.

Examples:
  # Update state
  fogit update "User Authentication" --state in-progress

  # Update priority
  fogit update "Login Page" --priority critical

  # Update multiple properties
  fogit update "API Endpoint" --domain backend --team api-team

  # Update metadata
  fogit update "Bug Fix" --metadata estimate=8h --metadata sprint=23
`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVar(&updateState, "state", "", "Update state (open, in-progress, closed)")
	updateCmd.Flags().StringVar(&updatePriority, "priority", "", "Update priority (low, medium, high, critical)")
	updateCmd.Flags().StringVar(&updateDescription, "description", "", "Update description")
	updateCmd.Flags().StringVar(&updateType, "type", "", "Update type")
	updateCmd.Flags().StringVar(&updateCategory, "category", "", "Update category")
	updateCmd.Flags().StringVar(&updateDomain, "domain", "", "Update domain")
	updateCmd.Flags().StringVar(&updateTeam, "team", "", "Update team")
	updateCmd.Flags().StringVar(&updateEpic, "epic", "", "Update epic")
	updateCmd.Flags().StringVar(&updateModule, "module", "", "Update module")
	updateCmd.Flags().StringVar(&updateName, "name", "", "Update name (renames file)")
	updateCmd.Flags().StringSliceVar(&updateMetadata, "metadata", []string{}, "Update metadata (key=value, repeatable)")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	// Check if at least one update flag is provided
	if !hasUpdateFlags(cmd) {
		return fmt.Errorf("no updates specified: provide at least one update flag (--state, --priority, etc.)")
	}

	// Get command context
	cmdCtx, err := GetCommandContext()
	if err != nil {
		return fmt.Errorf("failed to get command context: %w", err)
	}

	// Find feature by ID or name
	result, err := features.Find(cmd.Context(), cmdCtx.Repo, identifier, cmdCtx.Config)
	if err != nil {
		if err == fogit.ErrNotFound && result != nil && len(result.Suggestions) > 0 {
			printer.PrintSuggestions(os.Stdout, identifier, result.Suggestions, "fogit update <id> ...")
			return fmt.Errorf("feature not found")
		}
		if err == fogit.ErrNotFound {
			return fmt.Errorf("feature not found: %s", identifier)
		}
		return fmt.Errorf("failed to find feature: %w", err)
	}
	feature := result.Feature

	// Prepare update options
	opts := features.UpdateOptions{}

	if cmd.Flags().Changed("name") {
		if updateName == "" {
			return fmt.Errorf("name cannot be empty")
		}
		opts.Name = &updateName
	}
	if cmd.Flags().Changed("description") {
		opts.Description = &updateDescription
	}
	if cmd.Flags().Changed("state") {
		opts.State = &updateState
	}
	if cmd.Flags().Changed("priority") {
		opts.Priority = &updatePriority
	}
	if cmd.Flags().Changed("type") {
		opts.Type = &updateType
	}
	if cmd.Flags().Changed("category") {
		opts.Category = &updateCategory
	}
	if cmd.Flags().Changed("domain") {
		opts.Domain = &updateDomain
	}
	if cmd.Flags().Changed("team") {
		opts.Team = &updateTeam
	}
	if cmd.Flags().Changed("epic") {
		opts.Epic = &updateEpic
	}
	if cmd.Flags().Changed("module") {
		opts.Module = &updateModule
	}

	if len(updateMetadata) > 0 {
		opts.Metadata = make(map[string]interface{})
		for _, kv := range updateMetadata {
			key, value := common.SplitKeyValueEquals(kv)
			if value == "" {
				return fmt.Errorf("invalid metadata format: %s", kv)
			}
			opts.Metadata[key] = value
		}
	}

	// Apply updates
	changed, err := features.Update(cmd.Context(), cmdCtx.Repo, feature, opts)
	if err != nil {
		return err
	}

	if changed {
		// Output success
		fmt.Printf("Updated feature: %s\n", feature.ID)
		fmt.Printf("  Name: %s\n", feature.Name)
		if cmd.Flags().Changed("state") {
			fmt.Printf("  State: %s\n", feature.DeriveState())
		}
		if cmd.Flags().Changed("priority") {
			fmt.Printf("  Priority: %s\n", feature.GetPriority())
		}
		if cmd.Flags().Changed("description") {
			fmt.Printf("  Description: %s\n", feature.Description)
		}
	} else {
		fmt.Println("No changes made")
	}

	return nil
}

func hasUpdateFlags(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("state") ||
		cmd.Flags().Changed("priority") ||
		cmd.Flags().Changed("description") ||
		cmd.Flags().Changed("type") ||
		cmd.Flags().Changed("category") ||
		cmd.Flags().Changed("domain") ||
		cmd.Flags().Changed("team") ||
		cmd.Flags().Changed("epic") ||
		cmd.Flags().Changed("module") ||
		cmd.Flags().Changed("name") ||
		len(updateMetadata) > 0
}
