package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/git"
)

var (
	commitMessage    string
	commitAuthor     string
	commitAutoLink   bool
	commitAllowDirty bool
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit changes with feature tracking",
	Long: `Commit current changes and update feature metadata.

This command:
- Creates a Git commit with the specified message
- Updates the feature file with new timestamps
- Extracts and records commit authors
- Optionally links changed files to the feature

Examples:
  fogit commit -m "Add authentication features"
  fogit commit -m "Update API" --author bob@example.com
  fogit commit -m "Implement login" --auto-link`,
	RunE: runCommit,
}

func init() {
	commitCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "Commit message (required)")
	commitCmd.Flags().StringVar(&commitAuthor, "author", "", "Override commit author")
	commitCmd.Flags().BoolVar(&commitAutoLink, "auto-link", false, "Auto-link changed files to feature")
	commitCmd.Flags().BoolVar(&commitAllowDirty, "allow-dirty", false, "Allow commit with uncommitted changes in .fogit/")
	_ = commitCmd.MarkFlagRequired("message")

	rootCmd.AddCommand(commitCmd)
}

func runCommit(cmd *cobra.Command, args []string) error {
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

	// Open repository
	repo := getRepository(fogitDir)

	// Prepare commit options
	opts := features.CommitOptions{
		Message:    commitMessage,
		Author:     commitAuthor,
		AutoLink:   commitAutoLink,
		AllowDirty: commitAllowDirty,
	}

	// Execute commit
	result, err := features.Commit(cmd.Context(), repo, gitRepo, opts)
	if err != nil {
		return err
	}

	// Output results
	fmt.Printf("On branch: %s\n", result.Branch)
	fmt.Printf("Feature: %s\n", result.PrimaryFeature.Name)
	if len(result.Features) > 1 {
		fmt.Printf("  (and %d other feature(s) on this branch)\n", len(result.Features)-1)
	}

	if result.NothingToCommit {
		fmt.Println("Nothing to commit (working tree clean)")
		return nil
	}

	fmt.Printf("\nChanges to be committed:\n")
	for _, file := range result.ChangedFiles {
		fmt.Printf("  - %s\n", file)
	}

	fmt.Printf("✓ Updated %d feature(s) metadata\n", len(result.Features))
	fmt.Printf("\n✓ Committed: %s\n", result.Hash[:8])
	fmt.Printf("Author: %s <%s>\n", result.Author.Name, result.Author.Email)

	// Show linked files if auto-link was used
	if commitAutoLink && len(result.PrimaryFeature.Files) > 0 {
		fmt.Printf("\nLinked files (%d):\n", len(result.PrimaryFeature.Files))
		for _, file := range result.PrimaryFeature.Files {
			fmt.Printf("  - %s\n", file)
		}
	}

	return nil
}
