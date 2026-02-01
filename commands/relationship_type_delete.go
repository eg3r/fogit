package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
)

var (
	typeDeleteMigrateTo string
	typeDeleteCascade   bool
	typeDeleteForce     bool
)

var relationshipTypeDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a relationship type",
	Long: `Delete a relationship type from the configuration.

Without flags, errors if relationships exist using this type.
Use --migrate-to to update existing relationships to another type.
Use --cascade to delete the type and all relationships using it.
Use --force to delete only the type (leaves relationships orphaned).

Examples:
  # Migrate relationships to another type
  fogit relationship type delete needs --migrate-to depends-on

  # Delete type and all its relationships
  fogit relationship type delete old-type --cascade

  # Force delete (leaves orphaned relationships)
  fogit relationship type delete broken-type --force`,
	Args: cobra.ExactArgs(1),
	RunE: runRelationshipTypeDelete,
}

func init() {
	relationshipTypeDeleteCmd.Flags().StringVar(&typeDeleteMigrateTo, "migrate-to", "", "Migrate existing relationships to another type")
	relationshipTypeDeleteCmd.Flags().BoolVar(&typeDeleteCascade, "cascade", false, "Delete type and all relationships using it")
	relationshipTypeDeleteCmd.Flags().BoolVar(&typeDeleteForce, "force", false, "Delete type only, leave relationships orphaned")

	relationshipTypesCmd.AddCommand(relationshipTypeDeleteCmd)
}

func runRelationshipTypeDelete(cmd *cobra.Command, args []string) error {
	typeName := args[0]

	fogitDir, err := getFogitDir()
	if err != nil {
		return fmt.Errorf("failed to get .fogit directory: %w", err)
	}

	// Build options
	opts := features.RelationshipTypeDeleteOptions{
		MigrateTo: typeDeleteMigrateTo,
		Cascade:   typeDeleteCascade,
		Force:     typeDeleteForce,
	}

	// Try delete (may return error or require confirmation)
	result, err := features.DeleteRelationshipType(fogitDir, typeName, opts)
	if err != nil {
		// Check for "in use" error with helpful message
		if inUseErr, ok := err.(*features.RelationshipTypeInUseError); ok {
			fmt.Printf("Error: Cannot delete type '%s' - %d relationships are using this type.\n\n",
				inUseErr.TypeName, len(inUseErr.AffectedRels))
			fmt.Println("Affected relationships:")
			for i, rel := range inUseErr.AffectedRels {
				if i >= 5 {
					fmt.Printf("  ... and %d more\n", len(inUseErr.AffectedRels)-5)
					break
				}
				fmt.Printf("  • %s\n", rel)
			}
			fmt.Println("\nOptions:")
			fmt.Println("  --migrate-to <type>  Migrate relationships to another type")
			fmt.Println("  --cascade            Delete all relationships using this type")
			fmt.Println("  --force              Delete type only, leave relationships orphaned")
		}
		return err
	}

	// Handle confirmation if required
	if result.RequiresConfirm {
		fmt.Printf("WARNING: %s\n", result.ConfirmMessage)
		fmt.Println("This action cannot be undone (except via Git history).")
		if len(result.AffectedRels) > 0 {
			fmt.Println("\nAffected relationships:")
			for i, rel := range result.AffectedRels {
				if i >= 5 {
					fmt.Printf("  ... and %d more\n", len(result.AffectedRels)-5)
					break
				}
				fmt.Printf("  • %s\n", rel)
			}
		}
		fmt.Print("\nType 'yes' to confirm: ")
		reader := bufio.NewReader(os.Stdin)
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(confirm)
		if confirm != "yes" {
			return fmt.Errorf("deletion canceled")
		}

		// Execute after confirmation
		result, err = features.ExecuteRelationshipTypeDelete(fogitDir, typeName, opts)
		if err != nil {
			return err
		}
	}

	// Output results
	fmt.Printf("Deleted relationship type: %s\n", result.TypeName)
	if result.InverseType != "" {
		fmt.Printf("Deleted inverse type: %s\n", result.InverseType)
	}
	if result.MigratedCount > 0 {
		fmt.Printf("Migrated %d relationships to target type\n", result.MigratedCount)
	}
	if result.DeletedRelCount > 0 {
		fmt.Printf("Removed %d relationships from feature files\n", result.DeletedRelCount)
	}
	if opts.Force && len(result.AffectedRels) > 0 {
		fmt.Printf("\nWARNING: %d relationships may now be orphaned. Run 'fogit validate' to see them.\n", len(result.AffectedRels))
	}

	return nil
}
