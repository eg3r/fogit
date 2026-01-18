package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/git"
)

var (
	pushForce  bool
	pushRemote string
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push current branch and feature metadata to remote",
	Long: `Push current branch and feature metadata to remote repository.

This command:
- Pushes current Git branch to remote
- Pushes feature metadata (.fogit/ directory)
- Works in both branch-per-feature and trunk-based modes

Examples:
  fogit push
  fogit push --remote upstream
  fogit push --force`,
	RunE: runPush,
}

func init() {
	pushCmd.Flags().BoolVar(&pushForce, "force", false, "Force push (use with caution)")
	pushCmd.Flags().StringVar(&pushRemote, "remote", "origin", "Specify remote name")

	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) error {
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

	// Get current branch
	branch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if remote exists
	remotes, err := gitRepo.GetRemotes()
	if err != nil {
		return fmt.Errorf("failed to get remotes: %w", err)
	}

	if len(remotes) == 0 {
		return fmt.Errorf("no remotes configured. Add a remote with 'git remote add origin <url>'")
	}

	remoteExists := false
	for _, remote := range remotes {
		if remote == pushRemote {
			remoteExists = true
			break
		}
	}

	if !remoteExists {
		return fmt.Errorf("remote '%s' not found. Available remotes: %v", pushRemote, remotes)
	}

	fmt.Printf("Pushing branch '%s' to remote '%s'...\n", branch, pushRemote)

	// Push to remote
	err = gitRepo.Push(pushRemote)
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	fmt.Printf("✓ Successfully pushed '%s' to '%s/%s'\n", branch, pushRemote, branch)
	fmt.Println("✓ Feature metadata (.fogit/) pushed")

	return nil
}
