package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/git"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Tag management",
	Long: `Manage Git tags for marking significant feature baselines.

Tags are Git annotated tags that mark specific points in your repository history.
Use tags to mark releases, milestones, or significant feature versions.

Subcommands:
  create <name>  - Create a new annotated tag
  list           - List all tags
  delete <name>  - Delete a tag

Examples:
  fogit tag create v1.0.0 -m "First stable release"
  fogit tag list
  fogit tag delete old-tag`,
}

var tagCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new annotated tag",
	Long: `Create a new annotated Git tag at the current HEAD.

Tags are useful for marking releases or significant milestones.
The tag will be created with your Git user configuration.

Examples:
  fogit tag create v1.0.0 -m "First stable release"
  fogit tag create release-2024-01 -m "January 2024 release"
  fogit tag create milestone-alpha`,
	Args: cobra.ExactArgs(1),
	RunE: runTagCreate,
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	Long: `List all tags in the repository.

Shows tag name, commit hash, message (for annotated tags),
and creation date.

Examples:
  fogit tag list`,
	Args: cobra.NoArgs,
	RunE: runTagList,
}

var tagDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a tag",
	Long: `Delete a tag from the repository.

This only removes the local tag. To remove from remote, use:
  git push origin --delete <tagname>

Examples:
  fogit tag delete old-tag
  fogit tag delete v0.0.1`,
	Args: cobra.ExactArgs(1),
	RunE: runTagDelete,
}

var (
	tagMessage string
)

func init() {
	tagCreateCmd.Flags().StringVarP(&tagMessage, "message", "m", "", "Tag message (required for annotated tag)")

	tagCmd.AddCommand(tagCreateCmd)
	tagCmd.AddCommand(tagListCmd)
	tagCmd.AddCommand(tagDeleteCmd)

	rootCmd.AddCommand(tagCmd)
}

func runTagCreate(cmd *cobra.Command, args []string) error {
	tagName := args[0]

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

	// Use default message if not provided
	message := tagMessage
	if message == "" {
		message = fmt.Sprintf("Tag %s created by FoGit", tagName)
	}

	// Create the tag
	err = gitRepo.CreateTag(tagName, message)
	if err != nil {
		if err == git.ErrTagExists {
			return fmt.Errorf("tag '%s' already exists", tagName)
		}
		return fmt.Errorf("failed to create tag: %w", err)
	}

	fmt.Printf("Created tag: %s\n", tagName)
	if tagMessage != "" {
		fmt.Printf("  Message: %s\n", message)
	}

	return nil
}

func runTagList(cmd *cobra.Command, args []string) error {
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

	// Get all tags
	tags, err := gitRepo.ListTags()
	if err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}

	if len(tags) == 0 {
		fmt.Println("No tags found.")
		return nil
	}

	// Sort tags by date (newest first)
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Date.After(tags[j].Date)
	})

	fmt.Printf("Tags (%d):\n", len(tags))
	fmt.Println("─────────────────────────────────────────────────────────────")

	for _, tag := range tags {
		fmt.Printf("\n%s", tag.Name)
		if tag.IsLight {
			fmt.Printf(" (lightweight)")
		}
		fmt.Println()

		fmt.Printf("  Commit:  %s\n", tag.Hash)
		fmt.Printf("  Date:    %s\n", tag.Date.Format("2006-01-02 15:04:05"))

		if !tag.IsLight {
			fmt.Printf("  Tagger:  %s\n", tag.Tagger)
			if tag.Message != "" {
				fmt.Printf("  Message: %s\n", tag.Message)
			}
		}
	}

	fmt.Println("\n─────────────────────────────────────────────────────────────")

	return nil
}

func runTagDelete(cmd *cobra.Command, args []string) error {
	tagName := args[0]

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

	// Delete the tag
	err = gitRepo.DeleteTag(tagName)
	if err != nil {
		if err == git.ErrTagNotFound {
			return fmt.Errorf("tag '%s' not found", tagName)
		}
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	fmt.Printf("Deleted tag: %s\n", tagName)
	fmt.Println("\nNote: To remove from remote, run:")
	fmt.Printf("  git push origin --delete %s\n", tagName)

	return nil
}
