package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/git"
)

var (
	mergeNoDelete bool
	mergeSquash   bool
	mergeContinue bool
	mergeAbort    bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge [feature]",
	Short: "Merge and close feature",
	Long: `Merge feature branch and close feature(s).

This command:
- Closes the current feature (sets closed_at timestamp)
- Sets feature state to 'closed'
- Updates feature metadata
- Optionally deletes the feature branch

In branch-per-feature mode:
- Merges feature branch into main
- Closes all features on the current branch

In trunk-based mode:
- Simply closes the feature (no merge needed)

Conflict Resolution:
  If merge conflicts occur:
  1. FoGit pauses and reports the conflict
  2. Resolve conflicts manually (edit files, git add)
  3. Run 'fogit merge --continue' to complete
  4. Or 'fogit merge --abort' to cancel

Examples:
  fogit merge
  fogit finish
  fogit merge --no-delete
  fogit merge --squash
  fogit merge --continue   # After resolving conflicts
  fogit merge --abort      # Cancel merge`,
	Aliases: []string{"finish"},
	RunE:    runMerge,
}

func init() {
	mergeCmd.Flags().BoolVar(&mergeNoDelete, "no-delete", false, "Keep branch after merge")
	mergeCmd.Flags().BoolVar(&mergeSquash, "squash", false, "Squash commits")
	mergeCmd.Flags().BoolVar(&mergeContinue, "continue", false, "Continue merge after resolving conflicts")
	mergeCmd.Flags().BoolVar(&mergeAbort, "abort", false, "Abort the current merge")

	rootCmd.AddCommand(mergeCmd)
}

func runMerge(cmd *cobra.Command, args []string) error {
	// Find git root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	gitRoot, err := git.FindGitRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Open git repository
	gitRepo, err := git.OpenRepository(gitRoot)
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	// Find .fogit directory
	fogitDir := filepath.Join(gitRoot, ".fogit")
	if _, statErr := os.Stat(fogitDir); os.IsNotExist(statErr) {
		return fmt.Errorf(".fogit directory not found. Run 'fogit init' first")
	}

	// Load config to get base branch
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Open repository
	repo := getRepository(fogitDir)

	// Prepare merge options
	var featureName string
	if len(args) > 0 {
		featureName = args[0]
	}

	opts := features.MergeOptions{
		FeatureName: featureName,
		NoDelete:    mergeNoDelete,
		Squash:      mergeSquash,
		Continue:    mergeContinue,
		Abort:       mergeAbort,
		FogitDir:    fogitDir,
		BaseBranch:  cfg.Workflow.BaseBranch,
	}

	// Execute merge
	result, err := features.Merge(cmd.Context(), repo, gitRepo, opts)
	if err != nil {
		return err
	}

	// Handle abort result
	if result.Aborted {
		fmt.Printf("✓ Merge aborted\n")
		fmt.Printf("✓ Returned to branch: %s\n", result.Branch)
		fmt.Println("Feature(s) reopened.")
		return nil
	}

	// Handle conflict result
	if result.ConflictDetected {
		fmt.Println("⚠ Merge conflict detected!")
		fmt.Println("")
		fmt.Println("Resolve the conflicts, then:")
		fmt.Println("  1. Stage resolved files:  git add <files>")
		fmt.Println("  2. Complete the merge:    fogit merge --continue")
		fmt.Println("")
		fmt.Println("Or abort the merge:         fogit merge --abort")
		return nil
	}

	// Output results based on mode
	if result.IsMainBranch {
		// Trunk-based mode: just closed features
		fmt.Println("Trunk-based mode: closing feature(s)")
		for _, feature := range result.ClosedFeatures {
			fmt.Printf("✓ Closed feature: %s\n", feature.Name)
		}
	} else {
		// Branch-per-feature mode: merged and closed
		if result.MergePerformed {
			fmt.Printf("✓ Merged %s → %s\n", result.Branch, result.BaseBranch)
		}
		for _, feature := range result.ClosedFeatures {
			fmt.Printf("✓ Closed feature: %s\n", feature.Name)
		}
		if result.BranchDeleted {
			fmt.Printf("✓ Deleted branch: %s\n", result.Branch)
		} else if !result.NoDelete {
			fmt.Printf("  Branch '%s' kept (use --no-delete to suppress this)\n", result.Branch)
		}
	}

	fmt.Println("\nFeature(s) completed! 🎉")

	return nil
}
