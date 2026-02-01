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
	categoryDeleteMoveTypesTo string
	categoryDeleteCascade     bool
	categoryDeleteForce       bool
)

var relationshipCategoryDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a relationship category",
	Long: `Delete a relationship category from the configuration.

Without flags, errors if relationship types exist in this category.
Use --move-types-to to move existing types to another category.
Use --cascade to delete the category and all its types (and their relationships).
Use --force to delete only the category (types become uncategorized).

Examples:
  # Move types to another category first
  fogit relationship category delete old-category --move-types-to dependency

  # Delete category and all its types
  fogit relationship category delete obsolete --cascade

  # Force delete (types become uncategorized)
  fogit relationship category delete unused --force`,
	Args: cobra.ExactArgs(1),
	RunE: runRelationshipCategoryDelete,
}

func init() {
	relationshipCategoryDeleteCmd.Flags().StringVar(&categoryDeleteMoveTypesTo, "move-types-to", "", "Move types to another category before deletion")
	relationshipCategoryDeleteCmd.Flags().BoolVar(&categoryDeleteCascade, "cascade", false, "Delete category and all its types (and relationships)")
	relationshipCategoryDeleteCmd.Flags().BoolVar(&categoryDeleteForce, "force", false, "Delete category only, leave types uncategorized")

	relationshipCategoriesCmd.AddCommand(relationshipCategoryDeleteCmd)
}

func runRelationshipCategoryDelete(cmd *cobra.Command, args []string) error {
	categoryName := args[0]

	fogitDir, err := getFogitDir()
	if err != nil {
		return fmt.Errorf("failed to get .fogit directory: %w", err)
	}

	// Build options
	opts := features.RelationshipCategoryDeleteOptions{
		MoveTypesTo: categoryDeleteMoveTypesTo,
		Cascade:     categoryDeleteCascade,
		Force:       categoryDeleteForce,
	}

	// Try delete (may return error or require confirmation)
	result, err := features.DeleteRelationshipCategory(fogitDir, categoryName, opts)
	if err != nil {
		// Check for "in use" error with helpful message
		if inUseErr, ok := err.(*features.RelationshipCategoryInUseError); ok {
			fmt.Printf("Error: Cannot delete category '%s' - %d relationship types belong to it.\n\n",
				inUseErr.CategoryName, len(inUseErr.AffectedTypes))
			fmt.Println("Types in this category:")
			for i, t := range inUseErr.AffectedTypes {
				if i >= 10 {
					fmt.Printf("  ... and %d more\n", len(inUseErr.AffectedTypes)-10)
					break
				}
				fmt.Printf("  • %s\n", t)
			}
			fmt.Println("\nOptions:")
			fmt.Println("  --move-types-to <category>  Move types to another category")
			fmt.Println("  --cascade                   Delete category and all its types")
			fmt.Println("  --force                     Delete category only, types become uncategorized")
		}
		return err
	}

	// Handle confirmation if required
	if result.RequiresConfirm {
		fmt.Printf("WARNING: %s\n", result.ConfirmMessage)
		fmt.Println("This action cannot be undone (except via Git history).")
		if len(result.AffectedTypes) > 0 {
			fmt.Println("\nAffected types:")
			for i, t := range result.AffectedTypes {
				if i >= 10 {
					fmt.Printf("  ... and %d more\n", len(result.AffectedTypes)-10)
					break
				}
				fmt.Printf("  • %s\n", t)
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
		result, err = features.ExecuteRelationshipCategoryDelete(fogitDir, categoryName, opts)
		if err != nil {
			return err
		}
	}

	// Output results
	fmt.Printf("Deleted category: %s\n", result.CategoryName)
	if result.MovedTypesCount > 0 {
		fmt.Printf("Moved %d types to target category\n", result.MovedTypesCount)
	}
	if result.DeletedTypes > 0 {
		fmt.Printf("Deleted %d relationship types\n", result.DeletedTypes)
	}
	if result.DeletedRelCount > 0 {
		fmt.Printf("Removed %d relationships from feature files\n", result.DeletedRelCount)
	}
	if opts.Force && len(result.AffectedTypes) > 0 {
		fmt.Printf("\nWARNING: %d types are now uncategorized. Run 'fogit types' to see them.\n", len(result.AffectedTypes))
	}

	return nil
}
