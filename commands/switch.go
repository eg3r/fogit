package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/pkg/fogit"
)

var switchCmd = &cobra.Command{
	Use:   "switch <feature>",
	Short: "Switch to a different feature",
	Long: `Switch to a different feature.

In branch-per-feature mode:
  - Stashes current changes if any
  - Switches to the feature's branch
  - Restores previously stashed changes if applicable

In trunk-based mode:
  - Sets the active feature context (future: tracked in .fogit/context)

Examples:
  fogit switch "OAuth Implementation"
  fogit switch feature-oauth
  fogit switch abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runSwitch,
}

func init() {
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	// Get git repository
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	gitRoot, err := git.FindGitRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	gitRepo, err := git.OpenRepository(gitRoot)
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	// Execute switch via service
	opts := features.SwitchOptions{
		Identifier: identifier,
		FogitDir:   cmdCtx.FogitDir,
	}

	result, err := features.Switch(cmd.Context(), cmdCtx.Repo, gitRepo, cmdCtx.Config, opts)
	if err != nil {
		// Handle suggestions for not found
		if err == fogit.ErrNotFound {
			findResult, _ := features.Find(cmd.Context(), cmdCtx.Repo, identifier, cmdCtx.Config)
			if findResult != nil && len(findResult.Suggestions) > 0 {
				fmt.Fprintf(os.Stdout, "Feature '%s' not found.\n\nDid you mean:\n", identifier)
				for _, s := range findResult.Suggestions {
					fmt.Fprintf(os.Stdout, "  fogit switch %s\n", s.Feature.ID)
				}
			}
			return fmt.Errorf("feature not found")
		}
		return err
	}

	// Format output
	printSwitchResult(result)
	return nil
}

// printSwitchResult outputs the switch operation result
func printSwitchResult(result *features.SwitchResult) {
	if result.IsTrunkBased {
		fmt.Printf("Switched to feature: %s\n", result.Feature.Name)
		fmt.Printf("  ID: %s\n", result.Feature.ID)
		fmt.Printf("  State: %s\n", result.Feature.DeriveState())
		fmt.Printf("\nNote: In trunk-based mode, feature context is informational only.\n")
		return
	}

	if result.AlreadyOnBranch {
		fmt.Printf("Already on feature '%s' (branch: %s)\n", result.Feature.Name, result.TargetBranch)
		return
	}

	fmt.Printf("Switched to feature: %s\n", result.Feature.Name)
	fmt.Printf("  Branch: %s (was: %s)\n", result.TargetBranch, result.PreviousBranch)
	fmt.Printf("  State: %s\n", result.Feature.DeriveState())
}
