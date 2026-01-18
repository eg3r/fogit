package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	logFeature string
	logAuthor  string
	logSince   string
	logLimit   int
	logFormat  string
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show commit history",
	Long: `Show commit history with optional filtering.

Can filter by feature, author, date, and limit results.
Uses git log under the hood.

Examples:
  fogit log
  fogit log --feature "User Auth"
  fogit log --author alice --since 2025-01-01
  fogit log --limit 10 --format oneline`,
	RunE: runLog,
}

func init() {
	logCmd.Flags().StringVar(&logFeature, "feature", "", "Show commits for specific feature")
	logCmd.Flags().StringVar(&logAuthor, "author", "", "Filter by author")
	logCmd.Flags().StringVar(&logSince, "since", "", "Show commits since date (YYYY-MM-DD)")
	logCmd.Flags().IntVar(&logLimit, "limit", 0, "Limit number of results")
	logCmd.Flags().StringVar(&logFormat, "format", "full", "Output format: full, oneline, short")

	rootCmd.AddCommand(logCmd)
}

func runLog(cmd *cobra.Command, args []string) error {
	// Find git root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	gitRoot, err := git.FindGitRoot(cwd)
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Find .fogit directory
	fogitDir := filepath.Join(gitRoot, ".fogit")
	if _, statErr := os.Stat(fogitDir); os.IsNotExist(statErr) {
		return fmt.Errorf(".fogit directory not found. Run 'fogit init' first")
	}

	// Open git repository
	gitRepo, err := git.OpenRepository(gitRoot)
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	// If feature is specified, get commits for that feature's file
	var featurePath string
	if logFeature != "" {
		// Find the feature
		repo := getRepository(fogitDir)
		featuresList, listErr := repo.List(cmd.Context(), &fogit.Filter{})
		if listErr != nil {
			return fmt.Errorf("failed to list features: %w", listErr)
		}

		var targetFeature *fogit.Feature
		for _, f := range featuresList {
			if f.Name == logFeature || f.ID == logFeature {
				targetFeature = f
				break
			}
		}

		if targetFeature == nil {
			return fmt.Errorf("feature not found: %s", logFeature)
		}

		// Get feature file path
		opts := storage.SlugifyOptions{
			MaxLength:        50,
			AllowSlashes:     false,
			NormalizeUnicode: true,
			EmptyFallback:    "unnamed",
		}
		featurePath = filepath.Join("features", storage.Slugify(targetFeature.Name, opts)+".yml")
	}

	// Parse since date if provided
	var sinceTime *time.Time
	if logSince != "" {
		parsedTime, parseErr := time.Parse("2006-01-02", logSince)
		if parseErr != nil {
			return fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", parseErr)
		}
		sinceTime = &parsedTime
	}

	// Get commit log from git
	commits, err := gitRepo.GetLog(featurePath, logAuthor, sinceTime, logLimit)
	if err != nil {
		return fmt.Errorf("failed to get log: %w", err)
	}

	if len(commits) == 0 {
		fmt.Println("No commits found")
		return nil
	}

	// Display commits based on format
	switch logFormat {
	case "oneline":
		for _, commit := range commits {
			fmt.Printf("%s %s (%s)\n", commit.Hash[:8], commit.Message, commit.Author)
		}
	case "short":
		for _, commit := range commits {
			fmt.Printf("commit %s\n", commit.Hash[:8])
			fmt.Printf("Author: %s\n", commit.Author)
			fmt.Printf("\n    %s\n\n", commit.Message)
		}
	case "full":
		fallthrough
	default:
		for i, commit := range commits {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("commit %s\n", commit.Hash)
			fmt.Printf("Author: %s\n", commit.Author)
			fmt.Printf("Date:   %s\n", commit.Date.Format(time.RFC1123))
			fmt.Printf("\n    %s\n", commit.Message)
			if commit.Files > 0 {
				fmt.Printf("\n    %d file(s) changed\n", commit.Files)
			}
		}
	}

	fmt.Printf("\nTotal: %d commits\n", len(commits))

	return nil
}
