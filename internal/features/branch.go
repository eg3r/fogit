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

// HandleBranchCreation creates and checks out a Git branch for the feature if needed.
// Parameters:
//   - featureName: the name of the feature (used to generate branch name)
//   - cfg: the fogit configuration
//   - sameBranch: --same flag (create on current branch, shared strategy)
//   - isolateBranch: --isolate flag (force new branch)
//   - fromCurrent: --from-current flag (override create_branch_from, create from wherever you are)
func HandleBranchCreation(featureName string, cfg *fogit.Config, sameBranch, isolateBranch, fromCurrent bool) error {
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

		// Handle create_branch_from strategy (unless --from-current is set)
		if !fromCurrent {
			if err := handleCreateBranchFromStrategy(gitRepo, cfg); err != nil {
				return err
			}
		}

		// Create the branch
		if err := gitRepo.CreateBranch(branchName); err != nil {
			if err == git.ErrBranchExists {
				return fmt.Errorf("branch %s already exists. Use --same to create feature on current branch", branchName)
			}
			if err == git.ErrEmptyRepository {
				return fmt.Errorf("Git repository has no commits. Make an initial commit first:\n  git commit --allow-empty -m \"Initial commit\"")
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

// handleCreateBranchFromStrategy ensures we're on the right branch before creating a feature branch.
// Based on workflow.create_branch_from setting:
//   - "trunk": switch to base_branch if not already on it
//   - "warn": warn if not on base_branch but continue from current
//   - "current": do nothing (create from wherever you are)
func handleCreateBranchFromStrategy(gitRepo *git.Repository, cfg *fogit.Config) error {
	createBranchFrom := cfg.Workflow.CreateBranchFrom
	if createBranchFrom == "" {
		createBranchFrom = "trunk" // Default
	}

	// "current" means create from wherever you are - nothing to do
	if createBranchFrom == "current" {
		return nil
	}

	currentBranch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	baseBranch := cfg.Workflow.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	// Already on base branch - nothing to do
	if currentBranch == baseBranch {
		return nil
	}

	switch createBranchFrom {
	case "trunk":
		// Switch to base branch before creating feature branch
		fmt.Printf("Switching to %s before creating feature branch...\n", baseBranch)
		if err := gitRepo.CheckoutBranch(baseBranch); err != nil {
			return fmt.Errorf("failed to switch to %s: %w\nUse --from-current to create from current branch", baseBranch, err)
		}

	case "warn":
		// Warn but continue from current branch
		fmt.Printf("⚠️  Warning: Creating feature branch from '%s' instead of '%s'\n", currentBranch, baseBranch)
		fmt.Printf("   This may cause stale feature state on this branch.\n")
		fmt.Printf("   Use --from-current to suppress this warning, or switch to %s first.\n", baseBranch)
	}

	return nil
}
