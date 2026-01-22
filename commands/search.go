package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/common"
	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/internal/printer"
	"github.com/eg3r/fogit/pkg/fogit"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search features by text",
	Long: `Search for features by text in name, description, type, category, and tags.

The search is case-insensitive and matches partial text.
In branch-per-feature mode (default), automatically searches across all branches.
In trunk-based mode, searches only the current branch.

Examples:
  # Search for features containing "auth" (automatic cross-branch in branch-per-feature mode)
  fogit search auth

  # Search only current branch (override auto-discovery)
  fogit search auth --current-branch

  # Search with multiple words (AND logic)
  fogit search "user authentication"

  # Search and filter by state
  fogit search login --state open

  # Search and show in JSON format
  fogit search api --format json
`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

var (
	searchState       string
	searchPriority    string
	searchType        string
	searchCategory    string
	searchFormat      string
	searchAllBranches bool // Cross-branch discovery per spec
)

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringVar(&searchState, "state", "", "Filter by state")
	searchCmd.Flags().StringVar(&searchPriority, "priority", "", "Filter by priority")
	searchCmd.Flags().StringVar(&searchType, "type", "", "Filter by type")
	searchCmd.Flags().StringVar(&searchCategory, "category", "", "Filter by category")
	searchCmd.Flags().StringVar(&searchFormat, "format", "table", "Output format: table, json, csv")

	// Cross-branch discovery is automatic in branch-per-feature mode per spec/specification/07-git-integration.md
	// This flag allows overriding to only search current branch
	searchCmd.Flags().BoolVarP(&searchAllBranches, "current-branch", "c", false, "Search features from current branch only (override auto cross-branch discovery)")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Get repository
	fogitDir, err := getFogitDir()
	if err != nil {
		return fmt.Errorf("failed to get .fogit directory: %w", err)
	}

	repo := getRepository(fogitDir)

	// Load config to check workflow mode
	cfg, err := config.Load(fogitDir)
	if err != nil {
		// Fall back to default config if not found
		cfg = fogit.DefaultConfig()
	}

	// Build filter with search query
	filter := &fogit.Filter{
		Search:   query,
		State:    fogit.State(searchState),
		Priority: fogit.Priority(searchPriority),
		Type:     searchType,
		Category: searchCategory,
	}

	// Validate filter
	if validateErr := filter.Validate(); validateErr != nil {
		return validateErr
	}

	// Apply timeout to prevent hanging on slow filesystems
	ctx, cancel := WithSearchTimeout(cmd.Context())
	defer cancel()

	var featuresList []*fogit.Feature

	// Per spec/specification/07-git-integration.md#cross-branch-feature-discovery:
	// In branch-per-feature mode, cross-branch discovery is AUTOMATIC
	// Use --current-branch to override and only search current branch
	useCrossBranch := !searchAllBranches && cfg.Workflow.Mode == "branch-per-feature"

	if useCrossBranch {
		featuresList, err = searchFeaturesAllBranches(ctx, repo, filter)
		if err != nil {
			return err
		}
	} else {
		// trunk-based mode or --current-branch flag: search features on current branch only
		featuresList, err = repo.List(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to search features: %w", err)
		}
	}

	// Check if empty
	if len(featuresList) == 0 {
		fmt.Printf("No features found matching '%s'\n", query)
		return nil
	}

	// Display results
	fmt.Printf("Found %d feature(s) matching '%s':\n\n", len(featuresList), query)

	// Format output
	switch searchFormat {
	case "json":
		return printer.OutputJSON(os.Stdout, featuresList)
	case "csv":
		return printer.OutputCSV(os.Stdout, featuresList)
	default:
		outputSearchResults(featuresList, query)
		return nil
	}
}

// searchFeaturesAllBranches searches features from all branches using cross-branch discovery
func searchFeaturesAllBranches(ctx context.Context, repo fogit.Repository, filter *fogit.Filter) ([]*fogit.Feature, error) {
	// Get git repository for cross-branch operations
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	gitRoot, err := git.FindGitRoot(cwd)
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}

	gitRepo, err := git.OpenRepository(gitRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	// Use cross-branch feature discovery
	crossBranchFeatures, err := features.ListFeaturesAcrossBranches(ctx, repo, gitRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to search features across branches: %w", err)
	}

	// Extract features and apply filter
	var featuresList []*fogit.Feature
	for _, cbf := range crossBranchFeatures {
		// Apply filter
		if filter.Matches(cbf.Feature) {
			featuresList = append(featuresList, cbf.Feature)
		}
	}

	return featuresList, nil
}

func outputSearchResults(features []*fogit.Feature, query string) {
	for i, f := range features {
		if i > 0 {
			fmt.Println()
		}

		// Show basic info
		fmt.Printf("ID:       %s\n", f.ID)
		fmt.Printf("Name:     %s\n", f.Name)
		fmt.Printf("State:    %s\n", f.DeriveState())
		if priority := f.GetPriority(); priority != "" {
			fmt.Printf("Priority: %s\n", priority)
		}

		if fType := f.GetType(); fType != "" {
			fmt.Printf("Type:     %s\n", fType)
		}
		if category := f.GetCategory(); category != "" {
			fmt.Printf("Category: %s\n", category)
		}

		// Show description snippet if it matches
		if f.Description != "" && common.ContainsIgnoreCase(f.Description, query) {
			snippet := common.GetSnippet(f.Description, query, 100)
			fmt.Printf("Description: %s\n", snippet)
		}

		// Show matching tags
		if len(f.Tags) > 0 {
			matchingTags := []string{}
			for _, tag := range f.Tags {
				if common.ContainsIgnoreCase(tag, query) {
					matchingTags = append(matchingTags, tag)
				}
			}
			if len(matchingTags) > 0 {
				fmt.Printf("Tags:     %s\n", strings.Join(matchingTags, ", "))
			}
		}
	}
}
