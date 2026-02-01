package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
)

var (
	typeUpdateName          string
	typeUpdateRenameInverse string
	typeUpdateKeepOldAlias  bool
	typeUpdateCategory      string
	typeUpdateInverse       string
	typeUpdateDescription   string
	typeUpdateBidirectional bool
	typeUpdateNoBidirect    bool
	typeUpdateAddAliases    []string
	typeUpdateRemoveAliases []string
)

var relationshipTypeUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update or rename a relationship type",
	Long: `Update or rename an existing relationship type.

Use --name to rename the type (updates all existing relationships).
Other flags modify properties without renaming.

Examples:
  # Rename a type
  fogit relationship type update needs --name depends-on

  # Rename with custom inverse name
  fogit relationship type update needs --name depends-on --rename-inverse required-by

  # Keep old name as alias during rename
  fogit relationship type update needs --name depends-on --keep-old-as-alias

  # Change category
  fogit relationship type update blocks --category structural

  # Update description and aliases
  fogit relationship type update depends-on --description "Updated" --add-alias requires`,
	Args: cobra.ExactArgs(1),
	RunE: runRelationshipTypeUpdate,
}

func init() {
	relationshipTypeUpdateCmd.Flags().StringVar(&typeUpdateName, "name", "", "Rename the type (updates all relationships)")
	relationshipTypeUpdateCmd.Flags().StringVar(&typeUpdateRenameInverse, "rename-inverse", "", "Set new inverse name when renaming")
	relationshipTypeUpdateCmd.Flags().BoolVar(&typeUpdateKeepOldAlias, "keep-old-as-alias", false, "Keep old name as alias when renaming")
	relationshipTypeUpdateCmd.Flags().StringVarP(&typeUpdateCategory, "category", "c", "", "Change category")
	relationshipTypeUpdateCmd.Flags().StringVar(&typeUpdateInverse, "inverse", "", "Change inverse relationship")
	relationshipTypeUpdateCmd.Flags().StringVarP(&typeUpdateDescription, "description", "d", "", "Update description")
	relationshipTypeUpdateCmd.Flags().BoolVar(&typeUpdateBidirectional, "bidirectional", false, "Make relationship symmetric")
	relationshipTypeUpdateCmd.Flags().BoolVar(&typeUpdateNoBidirect, "no-bidirectional", false, "Make relationship directional")
	relationshipTypeUpdateCmd.Flags().StringArrayVar(&typeUpdateAddAliases, "add-alias", []string{}, "Add alias (repeatable)")
	relationshipTypeUpdateCmd.Flags().StringArrayVar(&typeUpdateRemoveAliases, "remove-alias", []string{}, "Remove alias (repeatable)")

	relationshipTypesCmd.AddCommand(relationshipTypeUpdateCmd)
}

func runRelationshipTypeUpdate(cmd *cobra.Command, args []string) error {
	typeName := args[0]

	fogitDir, err := getFogitDir()
	if err != nil {
		return fmt.Errorf("failed to get .fogit directory: %w", err)
	}

	// Handle bidirectional flags
	if typeUpdateBidirectional && typeUpdateNoBidirect {
		return fmt.Errorf("cannot specify both --bidirectional and --no-bidirectional")
	}

	// Build options
	opts := features.RelationshipTypeUpdateOptions{
		NewName:        typeUpdateName,
		RenameInverse:  typeUpdateRenameInverse,
		KeepOldAsAlias: typeUpdateKeepOldAlias,
		Category:       typeUpdateCategory,
		Inverse:        typeUpdateInverse,
		Description:    typeUpdateDescription,
		SetDescription: cmd.Flags().Changed("description"),
		AddAliases:     typeUpdateAddAliases,
		RemoveAliases:  typeUpdateRemoveAliases,
	}

	// Handle bidirectional setting
	if typeUpdateBidirectional {
		b := true
		opts.Bidirectional = &b
	}
	if typeUpdateNoBidirect {
		b := false
		opts.Bidirectional = &b
	}

	// Execute update
	result, err := features.UpdateRelationshipType(fogitDir, typeName, opts)
	if err != nil {
		return err
	}

	// Output results
	if result.Renamed {
		fmt.Printf("Renamed relationship type: %s → %s\n", result.OldName, result.NewName)
		if result.InverseRenamed {
			fmt.Printf("Renamed inverse type: %s → %s\n", result.OldInverse, result.NewInverse)
		}
		if result.UpdatedRelCount > 0 {
			fmt.Printf("Updated %d relationships in feature files\n", result.UpdatedRelCount)
		}
		if result.KeptOldAsAlias {
			fmt.Printf("Note: '%s' kept as alias\n", result.OldName)
		}
	} else {
		fmt.Printf("Updated relationship type: %s\n", result.NewName)
	}

	return nil
}
