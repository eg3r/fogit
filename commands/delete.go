package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/interactive"
)

var (
	deleteForce bool
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete <feature>",
	Short: "Delete a feature",
	Long: `Delete a feature from the repository by name or ID.

By default, prompts for confirmation before deletion.
Use --force to skip the confirmation prompt.

Examples:
  # Delete with confirmation
  fogit delete "Obsolete Feature"

  # Delete without confirmation
  fogit delete "Old Code" --force

  # Delete by ID
  fogit delete feature-id-123 --force
`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Skip confirmation prompt")
}

func runDelete(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	// Get command context
	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	// Find feature by ID or name using the consolidated helper
	feature, err := FindFeatureWithSuggestions(cmd.Context(), cmdCtx.Repo, identifier, cmdCtx.Config, "fogit delete <id>")
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	// Get incoming relationship summary using the service
	incomingSummaries, err := features.GetIncomingRelationshipSummary(ctx, cmdCtx.Repo, feature.ID)
	if err != nil {
		return fmt.Errorf("failed to check relationships: %w", err)
	}

	// Convert to interactive type for display
	incomingRels := make([]interactive.IncomingRelationship, len(incomingSummaries))
	for i, r := range incomingSummaries {
		incomingRels[i] = interactive.IncomingRelationship{
			SourceID:   r.SourceID,
			SourceName: r.SourceName,
			Type:       r.Type,
		}
	}

	// Confirm deletion unless --force is used
	if !deleteForce {
		prompter := interactive.NewPrompter()
		confirmed, confirmErr := prompter.ConfirmDeletion(feature, incomingRels)
		if confirmErr != nil {
			return fmt.Errorf("failed to get confirmation: %w", confirmErr)
		}
		if !confirmed {
			fmt.Println("Deletion canceled")
			return nil
		}
	}

	// Use the Delete service to handle the complete deletion
	deleteResult, err := features.Delete(ctx, cmdCtx.Repo, feature, features.DeleteOptions{})
	if err != nil {
		return err
	}

	// Report cleanup results
	if deleteResult.CleanedUpRelationships > 0 {
		fmt.Printf("Cleaned up %d incoming relationship(s)\n", deleteResult.CleanedUpRelationships)
	}

	fmt.Printf("Deleted feature: %s (%s)\n", feature.Name, feature.ID)
	return nil
}
