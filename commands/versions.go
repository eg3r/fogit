package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/common"
)

var versionsCmd = &cobra.Command{
	Use:   "versions <feature-name-or-id>",
	Short: "Show version history for a feature",
	Long: `Display the version history for a feature, including:
- Version number
- Created and closed timestamps
- Git branch
- Authors
- Notes

Example:
  fogit versions "User Authentication"
  fogit versions user-authentication`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nameOrID := args[0]

		// Get command context
		cmdCtx, err := GetCommandContext()
		if err != nil {
			return err
		}

		// Find feature using the consolidated helper
		feature, err := FindFeatureWithSuggestions(cmd.Context(), cmdCtx.Repo, nameOrID, cmdCtx.Config, "fogit versions <id>")
		if err != nil {
			return err
		}

		// Display feature header
		fmt.Printf("Version History: %s\n", feature.Name)
		fmt.Printf("ID: %s\n", feature.ID)
		fmt.Printf("Current Version: %s\n", feature.GetCurrentVersionKey())
		fmt.Printf("State: %s\n\n", feature.DeriveState())

		// Check if there are any versions
		if len(feature.Versions) == 0 {
			fmt.Println("No version history available.")
			return nil
		}

		// Display each version using the consolidated sorting method
		versionKeys := feature.GetSortedVersionKeys()

		fmt.Println("Versions:")
		fmt.Println("─────────────────────────────────────────────────────────────")

		for _, vKey := range versionKeys {
			v := feature.Versions[vKey]

			fmt.Printf("\nVersion %s:\n", vKey)
			fmt.Printf("  Created:  %s\n", common.FormatDateTime(v.CreatedAt))

			if v.ClosedAt != nil {
				fmt.Printf("  Closed:   %s\n", common.FormatDateTime(*v.ClosedAt))
				duration := v.ClosedAt.Sub(v.CreatedAt)
				fmt.Printf("  Duration: %s\n", common.FormatDurationLong(duration))
			} else {
				fmt.Printf("  Closed:   (still open)\n")
			}

			if v.Branch != "" {
				fmt.Printf("  Branch:   %s\n", v.Branch)
			}

			if len(v.Authors) > 0 {
				fmt.Printf("  Authors:  %s\n", formatVersionAuthors(v.Authors))
			}

			if v.Notes != "" {
				fmt.Printf("  Notes:    %s\n", v.Notes)
			}
		}

		fmt.Println("\n─────────────────────────────────────────────────────────────")

		return nil
	},
}

func formatVersionAuthors(authors []string) string {
	if len(authors) == 0 {
		return ""
	}
	if len(authors) == 1 {
		return authors[0]
	}
	return fmt.Sprintf("%s and %d others", authors[0], len(authors)-1)
}

func init() {
	rootCmd.AddCommand(versionsCmd)
}
