package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/printer"
	"github.com/eg3r/fogit/pkg/fogit"
)

var unlinkByID bool
var clearConfirm bool

var unlinkCmd = &cobra.Command{
	Use:   "unlink <source> <target|relationship-id> [type]",
	Short: "Remove a relationship between features",
	Long: `Remove a relationship between two features.

If type is not specified, the first matching relationship will be removed.
Use --by-id flag to remove by relationship ID instead.`,
	Args: cobra.RangeArgs(1, 3),
	RunE: runUnlink,
}

// relationshipCmd is the parent command for relationship subcommands
var relationshipCmd = &cobra.Command{
	Use:   "relationship",
	Short: "Manage feature relationships",
	Long:  `Commands for managing feature relationships.`,
}

var clearRelationshipsCmd = &cobra.Command{
	Use:   "clear <feature>",
	Short: "Remove all relationships from a feature",
	Long: `Remove all outgoing relationships from a feature.

This command clears all relationships that originate from the specified feature.
If auto-create-inverse is enabled, the corresponding inverse relationships will
also be removed from target features.`,
	Args: cobra.ExactArgs(1),
	RunE: runClearRelationships,
}

func init() {
	unlinkCmd.Flags().BoolVar(&unlinkByID, "by-id", false, "Remove relationship by ID (2nd argument is relationship ID)")
	rootCmd.AddCommand(unlinkCmd)

	// Add relationship subcommand
	clearRelationshipsCmd.Flags().BoolVarP(&clearConfirm, "yes", "y", false, "Skip confirmation prompt")
	relationshipCmd.AddCommand(clearRelationshipsCmd)
	rootCmd.AddCommand(relationshipCmd)
}

func runUnlink(cmd *cobra.Command, args []string) error {
	sourceIdentifier := args[0]

	cmdCtx, err := GetCommandContext()
	if err != nil {
		return fmt.Errorf("failed to get command context: %w", err)
	}

	ctx := cmd.Context()

	// Find source feature using cross-branch discovery
	source, err := FindFeatureCrossBranch(ctx, cmdCtx, sourceIdentifier, "fogit unlink <id> ...")
	if err != nil {
		return fmt.Errorf("source feature not found: %w", err)
	}

	// Handle removal by relationship ID
	if unlinkByID {
		if len(args) < 2 {
			return fmt.Errorf("relationship ID required when using --by-id")
		}
		relID := args[1]

		// Remove by ID
		foundRel, unlinkErr := features.Unlink(ctx, cmdCtx.Repo, source, relID, cmdCtx.FogitDir, cmdCtx.Config)
		if unlinkErr != nil {
			return unlinkErr
		}

		fmt.Printf("Removed relationship [%s]: %s -> %s (%s)\n",
			foundRel.ID[:8], source.Name, foundRel.TargetName, foundRel.Type)
		return nil
	}

	// Handle removal by target and type (original behavior)
	if len(args) < 2 {
		return fmt.Errorf("target feature required")
	}

	targetIdentifier := args[1]
	var relType fogit.RelationshipType
	if len(args) == 3 {
		relType = fogit.RelationshipType(args[2])
	}

	// Find target feature using cross-branch discovery
	target, err := FindFeatureCrossBranch(ctx, cmdCtx, targetIdentifier, "fogit unlink ... <id> ...")
	if err != nil {
		return fmt.Errorf("target feature not found: %w", err)
	}

	// Remove by target
	removedRel, err := features.UnlinkByTarget(ctx, cmdCtx.Repo, source, target, relType, cmdCtx.FogitDir, cmdCtx.Config)
	if err != nil {
		return err
	}

	fmt.Printf("Removed relationship: %s -> %s (%s)\n", source.Name, target.Name, removedRel.Type)
	return nil
}

func runClearRelationships(cmd *cobra.Command, args []string) error {
	featureIdentifier := args[0]

	cmdCtx, err := GetCommandContext()
	if err != nil {
		return fmt.Errorf("failed to get command context: %w", err)
	}

	ctx := cmd.Context()

	// Find feature
	result, err := features.Find(ctx, cmdCtx.Repo, featureIdentifier, cmdCtx.Config)
	if err != nil {
		if err == fogit.ErrNotFound && result != nil && len(result.Suggestions) > 0 {
			printer.PrintSuggestions(os.Stdout, featureIdentifier, result.Suggestions, "fogit relationship clear <id>")
			return fmt.Errorf("feature not found")
		}
		return fmt.Errorf("feature not found: %w", err)
	}
	feature := result.Feature

	// Check if there are any relationships to clear
	if len(feature.Relationships) == 0 {
		fmt.Printf("No relationships to clear for '%s'\n", feature.Name)
		return nil
	}

	// Show confirmation unless --yes flag is set
	if !clearConfirm {
		fmt.Printf("About to remove %d relationship(s) from '%s':\n", len(feature.Relationships), feature.Name)
		for _, rel := range feature.Relationships {
			targetName := rel.TargetName
			if targetName == "" {
				targetName = rel.TargetID[:8]
			}
			fmt.Printf("  - [%s] -> %s\n", rel.Type, targetName)
		}
		fmt.Print("\nProceed? [y/N]: ")

		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Canceled")
			return nil
		}
	}

	// Clear all relationships
	removedRels, err := features.ClearAllRelationships(ctx, cmdCtx.Repo, feature, cmdCtx.FogitDir, cmdCtx.Config)
	if err != nil {
		return fmt.Errorf("failed to clear relationships: %w", err)
	}

	// Report results
	fmt.Printf("\nCleared %d relationship(s) from '%s':\n", len(removedRels), feature.Name)
	for _, rel := range removedRels {
		targetName := rel.TargetName
		if targetName == "" {
			targetName = rel.TargetID[:8]
		}
		fmt.Printf("  - [%s] %s -> %s\n", rel.Type, feature.Name, targetName)
	}

	return nil
}
