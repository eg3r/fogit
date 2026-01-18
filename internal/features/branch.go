package features

import (
	"fmt"
	"os"

	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/internal/logger"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// SanitizeBranchName converts a feature name to a valid Git branch name
// Format: feature/<slug>
// Uses shared slugify logic with branch-specific options:
// - Normalizes unicode (removes accents)
// - Allows forward slashes (for branch hierarchies)
// - Longer length limit (240 chars for Git compatibility)
func SanitizeBranchName(name string) string {
	opts := storage.SlugifyOptions{
		MaxLength:        240, // Git recommends < 250, leave room for "feature/" prefix
		AllowSlashes:     true,
		NormalizeUnicode: true,
		EmptyFallback:    "unnamed",
	}

	slug := storage.Slugify(name, opts)

	return fmt.Sprintf("feature/%s", slug)
}

// HandleBranchCreation creates and checks out a Git branch for the feature if needed
func HandleBranchCreation(featureName string, cfg *fogit.Config, sameBranch, isolateBranch bool) error {
	action, err := DetermineBranchAction(cfg.Workflow.Mode, cfg.Workflow.AllowSharedBranches, sameBranch, isolateBranch)
	if err != nil {
		return err
	}

	switch action {
	case BranchActionNone:
		return nil

	case BranchActionStay:
		// Get current directory to check Git repo
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Check if we're in a Git repository
		gitRoot, err := git.FindGitRoot(cwd)
		if err != nil {
			return fmt.Errorf("not in a git repository: %w", err)
		}

		gitRepo, err := git.OpenRepository(gitRoot)
		if err != nil {
			return fmt.Errorf("failed to open git repository: %w", err)
		}

		currentBranch, err := gitRepo.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		fmt.Printf("Creating feature on current branch: %s\n", currentBranch)
		return nil

	case BranchActionCreate:
		// Generate branch name
		branchName := SanitizeBranchName(featureName)

		// Get current directory to check Git repo
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Check if we're in a Git repository
		gitRoot, err := git.FindGitRoot(cwd)
		if err != nil {
			// Not in a Git repo - warn but don't fail
			logger.Warn("not in a Git repository, branch creation skipped")
			fmt.Println("Initialize Git: git init")
			return nil
		}

		gitRepo, err := git.OpenRepository(gitRoot)
		if err != nil {
			return fmt.Errorf("failed to open git repository: %w", err)
		}

		// Create the branch
		if err := gitRepo.CreateBranch(branchName); err != nil {
			if err == git.ErrBranchExists {
				return fmt.Errorf("branch %s already exists. Use --same to create feature on current branch", branchName)
			}
			return fmt.Errorf("failed to create branch: %w", err)
		}

		// Checkout the branch
		if err := gitRepo.CheckoutBranch(branchName); err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}

		fmt.Printf("✓ Created and checked out branch: %s\n", branchName)
		return nil
	}

	return nil
}
